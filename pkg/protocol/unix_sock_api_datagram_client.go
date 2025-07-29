package protocol

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/user/GoUnixSockAPI/pkg/core"
	"github.com/user/GoUnixSockAPI/pkg/models"
	"github.com/user/GoUnixSockAPI/pkg/specification"
)

// UnixSockAPIDatagramClient is the main client interface for SOCK_DGRAM Unix socket communication
// Connectionless implementation for cross-language compatibility
type UnixSockAPIDatagramClient struct {
	socketPath     string
	channelID      string
	apiSpec        *specification.APISpecification
	config         UnixSockAPIDatagramClientConfig
	
	datagramClient *core.UnixDatagramClient
	validator      *core.SecurityValidator
	
	// Command handler registry (thread-safe)
	handlers      map[string]models.CommandHandler
	handlerMutex  sync.RWMutex
	
	// Timeout management
	timeoutManager *TimeoutManager
}

// UnixSockAPIDatagramClientConfig holds configuration for the datagram client
type UnixSockAPIDatagramClientConfig struct {
	MaxMessageSize   int
	DefaultTimeout   time.Duration
	DatagramTimeout  time.Duration
	EnableValidation bool
}

// DefaultUnixSockAPIDatagramClientConfig returns default configuration for SOCK_DGRAM
func DefaultUnixSockAPIDatagramClientConfig() UnixSockAPIDatagramClientConfig {
	return UnixSockAPIDatagramClientConfig{
		MaxMessageSize:   64 * 1024,      // 64KB datagram limit
		DefaultTimeout:   30 * time.Second,
		DatagramTimeout:  5 * time.Second,
		EnableValidation: true,
	}
}

// NewUnixSockAPIDatagramClient creates a new datagram API client
func NewUnixSockAPIDatagramClient(socketPath, channelID string, apiSpec *specification.APISpecification, config ...UnixSockAPIDatagramClientConfig) (*UnixSockAPIDatagramClient, error) {
	cfg := DefaultUnixSockAPIDatagramClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	// Create datagram client
	datagramConfig := core.UnixDatagramClientConfig{
		MaxMessageSize:  cfg.MaxMessageSize,
		DatagramTimeout: cfg.DatagramTimeout,
	}
	
	datagramClient, err := core.NewUnixDatagramClient(socketPath, datagramConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create datagram client: %w", err)
	}
	
	validator := core.NewSecurityValidator()
	timeoutManager := NewTimeoutManager()
	
	return &UnixSockAPIDatagramClient{
		socketPath:     socketPath,
		channelID:      channelID,
		apiSpec:        apiSpec,
		config:         cfg,
		datagramClient: datagramClient,
		validator:      validator,
		handlers:       make(map[string]models.CommandHandler),
		timeoutManager: timeoutManager,
	}, nil
}

// SendCommand sends a command via SOCK_DGRAM and waits for response
func (client *UnixSockAPIDatagramClient) SendCommand(ctx context.Context, command string, args map[string]interface{}, options ...CommandOptions) (*models.SocketResponse, error) {
	// Apply options
	opts := mergeCommandOptions(options...)
	
	// Generate command ID
	commandID := generateUUID()
	
	// Generate response socket path
	responseSocketPath := client.datagramClient.GenerateResponseSocketPath()
	
	// Create socket command
	socketCommand := models.SocketCommand{
		ID:        commandID,
		ChannelID: client.channelID,
		Command:   command,
		ReplyTo:   responseSocketPath,
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
	responseData, err := client.datagramClient.SendDatagram(commandCtx, commandData, responseSocketPath)
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
func (client *UnixSockAPIDatagramClient) SendCommandNoResponse(ctx context.Context, command string, args map[string]interface{}) error {
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
	return client.datagramClient.SendDatagramNoResponse(ctx, commandData)
}

// TestConnection tests connectivity to the server
func (client *UnixSockAPIDatagramClient) TestConnection(ctx context.Context) error {
	return client.datagramClient.TestDatagramSocket(ctx)
}

// Close cleans up client resources
func (client *UnixSockAPIDatagramClient) Close() error {
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
func (client *UnixSockAPIDatagramClient) GetChannelID() string {
	return client.channelID
}

// GetSocketPath returns the socket path
func (client *UnixSockAPIDatagramClient) GetSocketPath() string {
	return client.socketPath
}

// GetAPISpecification returns the API specification
func (client *UnixSockAPIDatagramClient) GetAPISpecification() *specification.APISpecification {
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