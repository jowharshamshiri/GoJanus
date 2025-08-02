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
	manifest        *specification.Manifest
	config         JanusClientConfig
	
	janusClient *core.JanusClient
	validator      *core.SecurityValidator
	
	// Command handler registry (thread-safe)
	handlers      map[string]models.CommandHandler
	handlerMutex  sync.RWMutex
	
	// Timeout management
	timeoutManager *TimeoutManager
	
	// Response correlation system
	responseTracker *ResponseTracker
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
func validateConstructorInputs(socketPath, channelID string, config JanusClientConfig) error {
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
	
	// Manifest is always fetched from server - no validation needed here
	
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

// fetchSpecificationFromServer fetches the Manifest from the server
func fetchSpecificationFromServer(janusClient *core.JanusClient, socketPath string, cfg JanusClientConfig) (*specification.Manifest, error) {
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
	
	parser := specification.NewManifestParser()
	manifest, err := parser.ParseJSON(specJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server specification: %w", err)
	}
	
	return manifest, nil
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
// Always fetches specification from server - no hardcoded specs allowed
func New(socketPath, channelID string, config ...JanusClientConfig) (*JanusClient, error) {
	cfg := DefaultJanusClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	// Validate inputs 
	if err := validateConstructorInputs(socketPath, channelID, cfg); err != nil {
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
	
	// Manifest will be fetched when needed during operations
	
	validator := core.NewSecurityValidator()
	timeoutManager := NewTimeoutManager()
	
	// Initialize response tracker for advanced client features
	trackerConfig := TrackerConfig{
		MaxPendingCommands: 1000,
		CleanupInterval:    30 * time.Second,
		DefaultTimeout:     cfg.DefaultTimeout,
	}
	responseTracker := NewResponseTracker(trackerConfig)
	
	return &JanusClient{
		socketPath:      socketPath,
		channelID:       channelID,
		manifest:         nil,
		config:          cfg,
		janusClient:     janusClient,
		validator:       validator,
		handlers:        make(map[string]models.CommandHandler),
		timeoutManager:  timeoutManager,
		responseTracker: responseTracker,
	}, nil
}

// ensureManifestLoaded fetches Manifest from server if not already loaded
func (client *JanusClient) ensureManifestLoaded() error {
	if client.manifest != nil {
		return nil // Already loaded
	}
	
	if !client.config.EnableValidation {
		return nil // Validation disabled, no need to fetch
	}
	
	// Fetch specification from server
	fetchedSpec, err := fetchSpecificationFromServer(client.janusClient, client.socketPath, client.config)
	if err != nil {
		return fmt.Errorf("failed to fetch Manifest: %w", err)
	}
	
	// Validate channel exists in fetched specification
	if _, exists := fetchedSpec.Channels[client.channelID]; !exists {
		return fmt.Errorf("channel '%s' not found in server specification", client.channelID)
	}
	
	client.manifest = fetchedSpec
	return nil
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
	
	// Ensure Manifest is loaded for validation
	if client.config.EnableValidation {
		if err := client.ensureManifestLoaded(); err != nil {
			return nil, fmt.Errorf("failed to load Manifest for validation: %w", err)
		}
	}

	// Validate command against Manifest
	if client.config.EnableValidation && client.manifest != nil {
		if !client.manifest.HasCommand(client.channelID, command) {
			return nil, fmt.Errorf("command '%s' not found in channel '%s'", command, client.channelID)
		}
		
		commandSpec, err := client.manifest.GetCommand(client.channelID, command)
		if err != nil {
			return nil, fmt.Errorf("command validation failed: %w", err)
		}
		
		if err := client.manifest.ValidateCommandArgs(commandSpec, args); err != nil {
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
	
	// Validate command against Manifest
	if client.config.EnableValidation && client.manifest != nil {
		if !client.manifest.HasCommand(client.channelID, command) {
			return fmt.Errorf("command '%s' not found in channel '%s'", command, client.channelID)
		}
		
		commandSpec, err := client.manifest.GetCommand(client.channelID, command)
		if err != nil {
			return fmt.Errorf("command validation failed: %w", err)
		}
		
		if err := client.manifest.ValidateCommandArgs(commandSpec, args); err != nil {
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
	
	// Clean up response tracker
	if client.responseTracker != nil {
		client.responseTracker.Shutdown()
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

// GetManifest returns the Manifest
func (client *JanusClient) GetManifest() *specification.Manifest {
	return client.manifest
}

// Helper functions

// generateUUID generates a simple UUID for command correlation
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// isBuiltinCommand checks if a command is a built-in command
func (client *JanusClient) isBuiltinCommand(command string) bool {
	builtinCommands := map[string]bool{
		"ping":         true,
		"echo":         true,
		"get_info":     true,
		"spec":         true,
		"validate":     true,
		"slow_process": true,
	}
	return builtinCommands[command]
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

// Specification returns the Manifest for backward compatibility  
func (client *JanusClient) Specification() *specification.Manifest {
	return client.manifest
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
	// Validate command exists in the Manifest for the client's channel
	if client.manifest != nil {
		if channel, exists := client.manifest.Channels[client.channelID]; exists {
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

// MARK: - Advanced Client Features (Response Correlation System)

// SendCommandAsync sends a command and returns a channel for receiving the response
func (client *JanusClient) SendCommandAsync(ctx context.Context, command string, args map[string]interface{}) (<-chan *models.SocketResponse, <-chan error) {
	responseChan := make(chan *models.SocketResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		response, err := client.SendCommand(ctx, command, args)
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- response
	}()

	return responseChan, errorChan
}

// SendCommandWithCorrelation sends a command with response correlation tracking
func (client *JanusClient) SendCommandWithCorrelation(ctx context.Context, command string, args map[string]interface{}, timeout time.Duration) (<-chan *models.SocketResponse, <-chan error, string) {
	commandID := generateUUID()
	responseChan := make(chan *models.SocketResponse, 1)
	errorChan := make(chan error, 1)

	// Track the command in response tracker
	err := client.responseTracker.TrackCommand(commandID, responseChan, errorChan, timeout)
	if err != nil {
		go func() { errorChan <- err }()
		return responseChan, errorChan, commandID
	}

	// Send the command asynchronously
	go func() {
		// Create response socket path
		responseSocketPath := fmt.Sprintf("/tmp/janus_response_%d_%s.sock", time.Now().UnixNano(), generateRandomID())

		// Create socket command with specific ID
		timeoutSeconds := float64(timeout.Seconds())
		socketCommand := models.SocketCommand{
			ID:        commandID,
			ChannelID: client.channelID,
			Command:   command,
			Args:      args,
			ReplyTo:   &responseSocketPath,
			Timeout:   &timeoutSeconds,
			Timestamp: float64(time.Now().Unix()),
		}

		// Validate and send command
		if client.config.EnableValidation {
			if err := client.ensureManifestLoaded(); err != nil {
				client.responseTracker.CancelCommand(commandID, fmt.Sprintf("Manifest loading failed: %v", err))
				return
			}

			if client.manifest != nil && !client.isBuiltinCommand(command) {
				if !client.manifest.HasCommand(client.channelID, command) {
					client.responseTracker.CancelCommand(commandID, fmt.Sprintf("command '%s' not found in channel '%s'", command, client.channelID))
					return
				}

				commandSpec, err := client.manifest.GetCommand(client.channelID, command)
				if err != nil {
					client.responseTracker.CancelCommand(commandID, fmt.Sprintf("command validation failed: %v", err))
					return
				}

				if err := client.manifest.ValidateCommandArgs(commandSpec, args); err != nil {
					client.responseTracker.CancelCommand(commandID, fmt.Sprintf("command validation failed: %v", err))
					return
				}
			}
		}

		// Serialize and send command
		commandData, err := json.Marshal(socketCommand)
		if err != nil {
			client.responseTracker.CancelCommand(commandID, fmt.Sprintf("failed to serialize command: %v", err))
			return
		}

		// Send datagram and wait for response
		responseData, err := client.janusClient.SendDatagram(ctx, commandData, responseSocketPath)
		if err != nil {
			client.responseTracker.CancelCommand(commandID, fmt.Sprintf("failed to send command datagram: %v", err))
			return
		}

		// Parse response
		var response models.SocketResponse
		if err := json.Unmarshal(responseData, &response); err != nil {
			client.responseTracker.CancelCommand(commandID, fmt.Sprintf("failed to deserialize response: %v", err))
			return
		}

		// Handle response through tracker
		client.responseTracker.HandleResponse(&response)
	}()

	return responseChan, errorChan, commandID
}

// CancelCommand cancels a pending command by ID
func (client *JanusClient) CancelCommand(commandID string, reason string) bool {
	return client.responseTracker.CancelCommand(commandID, reason)
}

// CancelAllCommands cancels all pending commands
func (client *JanusClient) CancelAllCommands(reason string) int {
	return client.responseTracker.CancelAllCommands(reason)
}

// GetPendingCommandCount returns the number of pending commands
func (client *JanusClient) GetPendingCommandCount() int {
	return client.responseTracker.GetPendingCount()
}

// GetPendingCommandIDs returns the IDs of all pending commands
func (client *JanusClient) GetPendingCommandIDs() []string {
	return client.responseTracker.GetPendingCommandIDs()
}

// IsCommandPending checks if a command is currently pending
func (client *JanusClient) IsCommandPending(commandID string) bool {
	return client.responseTracker.IsTracking(commandID)
}

// GetCommandStatistics returns statistics about pending commands
func (client *JanusClient) GetCommandStatistics() CommandStatistics {
	return client.responseTracker.GetStatistics()
}

// ExecuteCommandsInParallel executes multiple commands in parallel
func (client *JanusClient) ExecuteCommandsInParallel(ctx context.Context, commands []ParallelCommand) []ParallelResult {
	results := make([]ParallelResult, len(commands))
	var wg sync.WaitGroup

	for i, cmd := range commands {
		wg.Add(1)
		go func(index int, command ParallelCommand) {
			defer wg.Done()

			response, err := client.SendCommand(ctx, command.Command, command.Args)
			results[index] = ParallelResult{
				CommandID: command.ID,
				Response:  response,
				Error:     err,
			}
		}(i, cmd)
	}

	wg.Wait()
	return results
}

// CreateChannelProxy creates a proxy for executing commands on a specific channel
func (client *JanusClient) CreateChannelProxy(channelID string) *ChannelProxy {
	return &ChannelProxy{
		client:    client,
		channelID: channelID,
	}
}

// MARK: - Helper Types for Advanced Features

// ParallelCommand represents a command to be executed in parallel
type ParallelCommand struct {
	ID      string                 `json:"id"`
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args"`
}

// ParallelResult represents the result of a parallel command execution  
type ParallelResult struct {
	CommandID string                  `json:"commandId"`
	Response  *models.SocketResponse  `json:"response,omitempty"`
	Error     error                   `json:"error,omitempty"`
}

// ChannelProxy provides channel-specific command execution
type ChannelProxy struct {
	client    *JanusClient
	channelID string
}

// SendCommand sends a command through this channel proxy
func (proxy *ChannelProxy) SendCommand(ctx context.Context, command string, args map[string]interface{}) (*models.SocketResponse, error) {
	// Temporarily override channel ID
	originalChannelID := proxy.client.channelID
	proxy.client.channelID = proxy.channelID
	defer func() {
		proxy.client.channelID = originalChannelID
	}()

	return proxy.client.SendCommand(ctx, command, args)
}

// GetChannelID returns the proxy's channel ID
func (proxy *ChannelProxy) GetChannelID() string {
	return proxy.channelID
}