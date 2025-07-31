package protocol

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/user/GoJanus/pkg/core"
	"github.com/user/GoJanus/pkg/models"
	"github.com/user/GoJanus/pkg/specification"
)

// JanusClient is the main client interface for SOCK_DGRAM Unix socket communication
// Connectionless implementation for cross-language compatibility
type JanusClient struct {
	socketPath     string
	channelID      string
	apiSpec        *specification.APISpecification
	config         JanusClientConfig
	
	janusClient *core.JanusClient
	validator      *core.SecurityValidator
	
	// Command handler registry (thread-safe)
	handlers      map[string]models.CommandHandler
	handlerMutex  sync.RWMutex
	
	// Timeout management
	timeoutManager *TimeoutManager
}

// JanusClientConfig holds configuration for the datagram client
type JanusClientConfig struct {
	MaxMessageSize   int
	DefaultTimeout   time.Duration
	DatagramTimeout  time.Duration
	EnableValidation bool
}

// DefaultJanusClientConfig returns default configuration for SOCK_DGRAM
func DefaultJanusClientConfig() JanusClientConfig {
	return JanusClientConfig{
		MaxMessageSize:   64 * 1024,      // 64KB datagram limit
		DefaultTimeout:   30 * time.Second,
		DatagramTimeout:  5 * time.Second,
		EnableValidation: true,
	}
}

// validateConstructorInputs validates constructor parameters
func validateConstructorInputs(socketPath, channelID string, apiSpec *specification.APISpecification, config JanusClientConfig) error {
	// Validate socket path
	if socketPath == "" {
		return fmt.Errorf("socket path cannot be empty")
	}
	
	// Validate channel ID for security
	if channelID == "" {
		return fmt.Errorf("channel ID cannot be empty")
	}
	
	// Check for malicious channel IDs
	if strings.ContainsAny(channelID, "\x00;`$|&\n\r\t") || 
	   strings.Contains(channelID, "..") || 
	   strings.HasPrefix(channelID, "/") {
		return fmt.Errorf("invalid channel ID: contains forbidden characters")
	}
	
	// API specification can be nil (will be fetched from server)
	if apiSpec != nil {
		// Validate specification content if provided
		if apiSpec.Version == "" {
			return fmt.Errorf("API specification validation error: version cannot be empty")
		}
		
		if apiSpec.Channels == nil || len(apiSpec.Channels) == 0 {
			return fmt.Errorf("API specification validation error: no channels defined")
		}
		
		// Validate channel exists in specification
		if _, exists := apiSpec.Channels[channelID]; !exists {
			return fmt.Errorf("channel '%s' not found in API specification", channelID)
		}
	}
	
	// Validate configuration security
	if config.MaxMessageSize < 1024 {
		return fmt.Errorf("configuration error: MaxMessageSize too small, minimum 1024 bytes")
	}
	
	if config.DefaultTimeout < time.Second {
		return fmt.Errorf("configuration error: DefaultTimeout too short, minimum 1 second")
	}
	
	if config.DatagramTimeout < time.Millisecond*100 {
		return fmt.Errorf("configuration error: DatagramTimeout too short, minimum 100ms")
	}
	
	return nil
}

// fetchSpecificationFromServer fetches the API specification from the server
func fetchSpecificationFromServer(janusClient *core.JanusClient, socketPath string, cfg JanusClientConfig) (*specification.APISpecification, error) {
	// Generate response socket path
	responseSocketPath := fmt.Sprintf("/tmp/janus_spec_%d_%s.sock", time.Now().UnixNano(), generateRandomID())
	
	// Create spec command JSON
	specCommand := map[string]interface{}{
		"command":  "spec",
		"reply_to": responseSocketPath,
	}
	
	commandJSON, err := json.Marshal(specCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec command: %w", err)
	}
	
	// Send spec command to server with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DefaultTimeout)
	defer cancel()
	
	responseData, err := janusClient.SendDatagram(ctx, commandJSON, responseSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch specification from server: %w", err)
	}
	
	// Parse the response JSON
	var response map[string]interface{}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}
	
	// Check for error in response
	if errorMsg, exists := response["error"]; exists {
		return nil, fmt.Errorf("server returned error: %v", errorMsg)
	}
	
	// Extract specification from response
	specData, exists := response["result"]
	if !exists {
		return nil, fmt.Errorf("server response missing 'result' field")
	}
	
	// Convert spec data to JSON and parse
	specJSON, err := json.Marshal(specData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal specification data: %w", err)
	}
	
	parser := specification.NewAPISpecificationParser()
	apiSpec, err := parser.ParseJSON(specJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server specification: %w", err)
	}
	
	return apiSpec, nil
}

// generateRandomID generates a random ID for unique socket paths
func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// New creates a new datagram API client
// If apiSpec is nil, automatically fetches specification from server
func New(socketPath, channelID string, apiSpec *specification.APISpecification, config ...JanusClientConfig) (*JanusClient, error) {
	cfg := DefaultJanusClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	// Validate inputs (apiSpec can be nil)
	if err := validateConstructorInputs(socketPath, channelID, apiSpec, cfg); err != nil {
		return nil, err
	}
	
	// Create datagram client
	datagramConfig := core.JanusClientConfig{
		MaxMessageSize:  cfg.MaxMessageSize,
		DatagramTimeout: cfg.DatagramTimeout,
	}
	
	janusClient, err := core.NewJanusClient(socketPath, datagramConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create datagram client: %w", err)
	}
	
	// If no API specification provided, fetch from server
	if apiSpec == nil {
		fetchedSpec, err := fetchSpecificationFromServer(janusClient, socketPath, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch specification from server: %w", err)
		}
		apiSpec = fetchedSpec
		
		// Validate channel exists in fetched specification
		if _, exists := apiSpec.Channels[channelID]; !exists {
			return nil, fmt.Errorf("channel '%s' not found in server specification", channelID)
		}
	}
	
	validator := core.NewSecurityValidator()
	timeoutManager := NewTimeoutManager()
	
	return &JanusClient{
		socketPath:     socketPath,
		channelID:      channelID,
		apiSpec:        apiSpec,
		config:         cfg,
		janusClient: janusClient,
		validator:      validator,
		handlers:       make(map[string]models.CommandHandler),
		timeoutManager: timeoutManager,
	}, nil
}

// SendCommand sends a command via SOCK_DGRAM and waits for response
func (client *JanusClient) SendCommand(ctx context.Context, command string, args map[string]interface{}, options ...CommandOptions) (*models.SocketResponse, error) {
	// Apply options
	opts := mergeCommandOptions(options...)
	
	// Generate command ID
	commandID := generateUUID()
	
	// Generate response socket path
	responseSocketPath := client.janusClient.GenerateResponseSocketPath()
	
	// Create socket command
	socketCommand := models.SocketCommand{
		ID:        commandID,
		ChannelID: client.channelID,
		Command:   command,
		ReplyTo:   &responseSocketPath,
		Args:      args,
		Timeout:   func() *float64 { f := opts.Timeout.Seconds(); return &f }(),
		Timestamp: float64(time.Now().Unix()),
	}
	
	// Validate command against API specification
	if client.config.EnableValidation && client.apiSpec != nil {
		if !client.apiSpec.HasCommand(client.channelID, command) {
			return nil, fmt.Errorf("command '%s' not found in channel '%s'", command, client.channelID)
		}
		
		commandSpec, err := client.apiSpec.GetCommand(client.channelID, command)
		if err != nil {
			return nil, fmt.Errorf("command validation failed: %w", err)
		}
		
		if err := client.apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
			return nil, fmt.Errorf("command validation failed: %w", err)
		}
	}
	
	// Apply timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = client.config.DefaultTimeout
	}
	
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Serialize command
	commandData, err := json.Marshal(socketCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %w", err)
	}
	
	// Send datagram and wait for response
	responseData, err := client.janusClient.SendDatagram(commandCtx, commandData, responseSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to send command datagram: %w", err)
	}
	
	// Deserialize response
	var response models.SocketResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}
	
	// Validate response correlation
	if response.CommandID != commandID {
		return nil, fmt.Errorf("response correlation mismatch: expected %s, got %s", commandID, response.CommandID)
	}
	
	if response.ChannelID != client.channelID {
		return nil, fmt.Errorf("channel mismatch: expected %s, got %s", client.channelID, response.ChannelID)
	}
	
	return &response, nil
}

// SendCommandNoResponse sends a command without expecting a response (fire-and-forget)
func (client *JanusClient) SendCommandNoResponse(ctx context.Context, command string, args map[string]interface{}) error {
	// Generate command ID
	commandID := generateUUID()
	
	// Create socket command (no reply_to field)
	socketCommand := models.SocketCommand{
		ID:        commandID,
		ChannelID: client.channelID,
		Command:   command,
		Args:      args,
		Timeout:   nil, // No timeout for fire-and-forget
		Timestamp: float64(time.Now().Unix()),
	}
	
	// Validate command against API specification
	if client.config.EnableValidation && client.apiSpec != nil {
		if !client.apiSpec.HasCommand(client.channelID, command) {
			return fmt.Errorf("command '%s' not found in channel '%s'", command, client.channelID)
		}
		
		commandSpec, err := client.apiSpec.GetCommand(client.channelID, command)
		if err != nil {
			return fmt.Errorf("command validation failed: %w", err)
		}
		
		if err := client.apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
			return fmt.Errorf("command validation failed: %w", err)
		}
	}
	
	// Serialize command
	commandData, err := json.Marshal(socketCommand)
	if err != nil {
		return fmt.Errorf("failed to serialize command: %w", err)
	}
	
	// Send datagram without waiting for response
	return client.janusClient.SendDatagramNoResponse(ctx, commandData)
}

// TestConnection tests connectivity to the server
func (client *JanusClient) TestConnection(ctx context.Context) error {
	return client.janusClient.TestDatagramSocket(ctx)
}

// Close cleans up client resources
func (client *JanusClient) Close() error {
	// Clean up timeout manager
	if client.timeoutManager != nil {
		client.timeoutManager.Close()
	}
	
	// Clear handlers
	client.handlerMutex.Lock()
	client.handlers = make(map[string]models.CommandHandler)
	client.handlerMutex.Unlock()
	
	return nil
}

// GetChannelID returns the channel ID
func (client *JanusClient) GetChannelID() string {
	return client.channelID
}

// GetSocketPath returns the socket path
func (client *JanusClient) GetSocketPath() string {
	return client.socketPath
}

// GetAPISpecification returns the API specification
func (client *JanusClient) GetAPISpecification() *specification.APISpecification {
	return client.apiSpec
}

// Helper functions

// generateUUID generates a simple UUID for command correlation
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// CommandOptions holds options for sending commands
type CommandOptions struct {
	Timeout time.Duration
}

// mergeCommandOptions merges command options with defaults
func mergeCommandOptions(options ...CommandOptions) CommandOptions {
	opts := CommandOptions{
		Timeout: 30 * time.Second, // default
	}
	
	for _, option := range options {
		if option.Timeout > 0 {
			opts.Timeout = option.Timeout
		}
	}
	
	return opts
}

// Backward compatibility methods for tests

// ChannelIdentifier returns the channel ID for backward compatibility
func (client *JanusClient) ChannelIdentifier() string {
	return client.channelID
}

// Specification returns the API specification for backward compatibility  
func (client *JanusClient) Specification() *specification.APISpecification {
	return client.apiSpec
}

// PublishCommand sends a command without expecting response for backward compatibility
func (client *JanusClient) PublishCommand(ctx context.Context, command string, args map[string]interface{}) (string, error) {
	err := client.SendCommandNoResponse(ctx, command, args)
	if err != nil {
		return "", err
	}
	// Return a generated command ID for compatibility
	return generateUUID(), nil
}

// SocketPathString returns the socket path for backward compatibility
func (client *JanusClient) SocketPathString() string {
	return client.socketPath
}

// RegisterCommandHandler validates command exists in specification (SOCK_DGRAM compatibility)
func (client *JanusClient) RegisterCommandHandler(command string, handler interface{}) error {
	// Validate command exists in the API specification for the client's channel
	if client.apiSpec != nil {
		if channel, exists := client.apiSpec.Channels[client.channelID]; exists {
			if _, commandExists := channel.Commands[command]; !commandExists {
				return fmt.Errorf("command '%s' not found in channel '%s'", command, client.channelID)
			}
		}
	}
	
	// SOCK_DGRAM doesn't actually use handlers, but validation passed
	return nil
}

// Disconnect is a no-op for backward compatibility (SOCK_DGRAM doesn't have persistent connections)
func (client *JanusClient) Disconnect() error {
	// SOCK_DGRAM doesn't have persistent connections - this is for backward compatibility only
	return nil
}

// IsConnected always returns true for backward compatibility (SOCK_DGRAM doesn't track connections)
func (client *JanusClient) IsConnected() bool {
	// SOCK_DGRAM doesn't track connections - return true for backward compatibility
	return true
}

// Ping sends a ping command and returns success/failure
// Convenience method for testing connectivity with a simple ping command
func (client *JanusClient) Ping(ctx context.Context) bool {
	_, err := client.SendCommand(ctx, "ping", nil)
	return err == nil
}