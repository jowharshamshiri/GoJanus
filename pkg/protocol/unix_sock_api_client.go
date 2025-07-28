package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/GoUnixSocketAPI/pkg/core"
	"github.com/user/GoUnixSocketAPI/pkg/models"
	"github.com/user/GoUnixSocketAPI/pkg/specification"
)

// UnixSockAPIClient is the main client interface for Unix socket communication
// Matches Swift UnixSockAPIClient functionality exactly for cross-language compatibility
type UnixSockAPIClient struct {
	socketPath     string
	channelID      string
	apiSpec        *specification.APISpecification
	config         UnixSockAPIClientConfig
	
	connectionPool *core.ConnectionPool
	validator      *core.SecurityValidator
	
	// Command handler registry (thread-safe)
	handlers      map[string]models.CommandHandler
	handlerMutex  sync.RWMutex
	
	// Timeout management
	timeoutManager *TimeoutManager
	
	// Listening state
	isListening   bool
	listenMutex   sync.Mutex
	stopListening chan struct{}
}

// UnixSockAPIClientConfig holds configuration options for the API client
// Matches Swift configuration structure exactly
type UnixSockAPIClientConfig struct {
	MaxConcurrentConnections int           `json:"maxConcurrentConnections"`
	MaxMessageSize          int           `json:"maxMessageSize"`
	ConnectionTimeout       time.Duration `json:"connectionTimeout"`
	MaxPendingCommands      int           `json:"maxPendingCommands"`
	MaxCommandHandlers      int           `json:"maxCommandHandlers"`
	EnableResourceMonitoring bool          `json:"enableResourceMonitoring"`
	MaxChannelNameLength    int           `json:"maxChannelNameLength"`
	MaxCommandNameLength    int           `json:"maxCommandNameLength"`
	MaxArgsDataSize         int           `json:"maxArgsDataSize"`
}

// DefaultUnixSockAPIClientConfig returns default configuration matching Swift defaults exactly
func DefaultUnixSockAPIClientConfig() UnixSockAPIClientConfig {
	return UnixSockAPIClientConfig{
		MaxConcurrentConnections: 100,                // Matches Swift default
		MaxMessageSize:          10 * 1024 * 1024,   // 10MB matches Swift
		ConnectionTimeout:       30 * time.Second,    // 30s matches Swift
		MaxPendingCommands:      1000,               // Matches Swift default
		MaxCommandHandlers:      500,                // Matches Swift default
		EnableResourceMonitoring: true,              // Matches Swift default
		MaxChannelNameLength:    256,                // Matches Swift default
		MaxCommandNameLength:    256,                // Matches Swift default
		MaxArgsDataSize:         5 * 1024 * 1024,    // 5MB matches Swift
	}
}

// NewUnixSockAPIClient creates a new Unix socket API client
// Matches Swift: init(socketPath: String, channelId: String, apiSpec: APISpecification, config: UnixSockAPIClientConfig)
func NewUnixSockAPIClient(socketPath, channelID string, apiSpec *specification.APISpecification, config ...UnixSockAPIClientConfig) (*UnixSockAPIClient, error) {
	cfg := DefaultUnixSockAPIClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	validator := core.NewSecurityValidator()
	
	// Validate socket path
	if err := validator.ValidateSocketPath(socketPath); err != nil {
		return nil, fmt.Errorf("invalid socket path: %w", err)
	}
	
	// Validate channel ID
	if err := validator.ValidateChannelID(channelID); err != nil {
		return nil, fmt.Errorf("invalid channel ID: %w", err)
	}
	
	// Validate API specification
	if apiSpec == nil {
		return nil, fmt.Errorf("API specification cannot be nil")
	}
	
	if err := apiSpec.Validate(); err != nil {
		return nil, fmt.Errorf("API specification validation failed: %w", err)
	}
	
	// Check if channel exists in API specification
	if apiSpec.Channels == nil {
		return nil, fmt.Errorf("API specification has no channels")
	}
	
	if _, exists := apiSpec.Channels[channelID]; !exists {
		return nil, fmt.Errorf("channel '%s' not found in API specification", channelID)
	}
	
	// Validate resource limits
	if err := validator.ValidateResourceLimits(cfg.MaxConcurrentConnections, cfg.MaxCommandHandlers, cfg.MaxPendingCommands); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create connection pool
	poolConfig := core.ConnectionPoolConfig{
		MaxConnections:    cfg.MaxConcurrentConnections,
		ConnectionTimeout: cfg.ConnectionTimeout,
		MaxMessageSize:    cfg.MaxMessageSize,
	}
	
	pool, err := core.NewConnectionPool(socketPath, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	
	// Create timeout manager
	timeoutManager := NewTimeoutManager()
	
	return &UnixSockAPIClient{
		socketPath:     socketPath,
		channelID:      channelID,
		apiSpec:        apiSpec,
		config:         cfg,
		connectionPool: pool,
		validator:      validator,
		handlers:       make(map[string]models.CommandHandler),
		timeoutManager: timeoutManager,
		stopListening:  make(chan struct{}),
	}, nil
}

// RegisterCommandHandler registers a handler for a specific command
// Matches Swift: func registerCommandHandler(_ commandName: String, handler: @escaping CommandHandler) throws
func (client *UnixSockAPIClient) RegisterCommandHandler(commandName string, handler models.CommandHandler) error {
	// Validate command name
	if err := client.validator.ValidateCommandName(commandName); err != nil {
		return fmt.Errorf("invalid command name: %w", err)
	}
	
	// Check if command exists in API specification
	if !client.apiSpec.HasCommand(client.channelID, commandName) {
		return fmt.Errorf("command '%s' not found in API specification for channel '%s'", commandName, client.channelID)
	}
	
	client.handlerMutex.Lock()
	defer client.handlerMutex.Unlock()
	
	// Check handler limit
	if len(client.handlers) >= client.config.MaxCommandHandlers {
		return fmt.Errorf("maximum number of command handlers (%d) reached", client.config.MaxCommandHandlers)
	}
	
	client.handlers[commandName] = handler
	return nil
}

// SendCommand sends a command and waits for response
// Matches Swift: func sendCommand(_ commandName: String, args: [String: AnyCodable]?, timeout: TimeInterval, onTimeout: TimeoutHandler?) async throws -> SocketResponse
func (client *UnixSockAPIClient) SendCommand(ctx context.Context, commandName string, args map[string]interface{}, timeout time.Duration, onTimeout models.TimeoutHandler) (*models.SocketResponse, error) {
	// Validate command name
	if err := client.validator.ValidateCommandName(commandName); err != nil {
		return nil, fmt.Errorf("invalid command name: %w", err)
	}
	
	// Validate timeout
	if err := client.validator.ValidateTimeout(timeout.Seconds()); err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}
	
	// Validate command against API specification
	commandSpec, err := client.apiSpec.GetCommand(client.channelID, commandName)
	if err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}
	
	// Validate arguments against specification
	if err := client.apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}
	
	// Create command
	timeoutSeconds := timeout.Seconds()
	command := models.NewSocketCommand(client.channelID, commandName, args, &timeoutSeconds)
	
	// Serialize command to JSON
	commandData, err := command.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %w", err)
	}
	
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Register timeout handler if provided
	if onTimeout != nil {
		client.timeoutManager.RegisterTimeout(command.ID, timeout, func() {
			onTimeout(command.ID)
		})
		defer client.timeoutManager.CancelTimeout(command.ID)
	}
	
	// Send command through connection pool (stateless communication)
	responseData, err := client.connectionPool.SendMessage(timeoutCtx, commandData)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	
	// Parse response
	var response models.SocketResponse
	if err := response.FromJSON(responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Validate channel isolation
	if err := client.validator.ValidateChannelIsolation(client.channelID, response.ChannelID); err != nil {
		return nil, fmt.Errorf("channel isolation violation: %w", err)
	}
	
	return &response, nil
}

// PublishCommand sends a fire-and-forget command (no response expected)
// Matches Swift: func publishCommand(_ commandName: String, args: [String: AnyCodable]?) async throws -> String
func (client *UnixSockAPIClient) PublishCommand(ctx context.Context, commandName string, args map[string]interface{}) (string, error) {
	// Validate command name
	if err := client.validator.ValidateCommandName(commandName); err != nil {
		return "", fmt.Errorf("invalid command name: %w", err)
	}
	
	// Validate command against API specification
	commandSpec, err := client.apiSpec.GetCommand(client.channelID, commandName)
	if err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}
	
	// Validate arguments against specification
	if err := client.apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
		return "", fmt.Errorf("argument validation failed: %w", err)
	}
	
	// Create command
	command := models.NewSocketCommand(client.channelID, commandName, args, nil)
	
	// Serialize command to JSON
	commandData, err := command.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize command: %w", err)
	}
	
	// Send command through connection pool (fire and forget)
	if err := client.connectionPool.SendMessageNoResponse(ctx, commandData); err != nil {
		return "", fmt.Errorf("failed to publish command: %w", err)
	}
	
	return command.ID, nil
}

// StartListening starts listening for incoming commands
// Matches Swift: func startListening() async throws
func (client *UnixSockAPIClient) StartListening(ctx context.Context) error {
	client.listenMutex.Lock()
	defer client.listenMutex.Unlock()
	
	if client.isListening {
		return fmt.Errorf("already listening")
	}
	
	// Create a dedicated connection for listening
	clientConfig := core.UnixSocketClientConfig{
		MaxMessageSize:    client.config.MaxMessageSize,
		ConnectionTimeout: client.config.ConnectionTimeout,
	}
	
	listener, err := core.NewUnixSocketClient(client.socketPath, clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	
	// Add message handler for incoming commands
	listener.AddMessageHandler(func(data []byte) {
		client.handleIncomingMessage(data)
	})
	
	// Start listening
	if err := listener.StartListening(ctx); err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}
	
	client.isListening = true
	
	// Start goroutine to handle stop signal
	go func() {
		select {
		case <-client.stopListening:
			listener.Disconnect()
			client.listenMutex.Lock()
			client.isListening = false
			client.listenMutex.Unlock()
		case <-ctx.Done():
			listener.Disconnect()
			client.listenMutex.Lock()
			client.isListening = false
			client.listenMutex.Unlock()
		}
	}()
	
	return nil
}

// StopListening stops listening for incoming commands
func (client *UnixSockAPIClient) StopListening() {
	client.listenMutex.Lock()
	defer client.listenMutex.Unlock()
	
	if client.isListening {
		close(client.stopListening)
		client.stopListening = make(chan struct{})
	}
}

// handleIncomingMessage processes incoming command messages
// Implements the command execution logic that was incomplete in Rust
func (client *UnixSockAPIClient) handleIncomingMessage(data []byte) {
	// Parse command
	var command models.SocketCommand
	if err := command.FromJSON(data); err != nil {
		// Invalid command format, ignore
		return
	}
	
	// Validate channel isolation
	if err := client.validator.ValidateChannelIsolation(client.channelID, command.ChannelID); err != nil {
		// Send error response
		errorResponse := models.NewErrorResponse(command.ID, client.channelID, &models.SocketError{
			Code:    "CHANNEL_ISOLATION_VIOLATION",
			Message: err.Error(),
		})
		client.sendResponse(errorResponse)
		return
	}
	
	// Find handler
	client.handlerMutex.RLock()
	handler, exists := client.handlers[command.Command]
	client.handlerMutex.RUnlock()
	
	if !exists {
		// Command not found
		errorResponse := models.NewErrorResponse(command.ID, client.channelID, &models.SocketError{
			Code:    "COMMAND_NOT_FOUND",
			Message: fmt.Sprintf("No handler registered for command: %s", command.Command),
		})
		client.sendResponse(errorResponse)
		return
	}
	
	// Execute handler in goroutine
	go func() {
		response, err := handler(&command)
		if err != nil {
			errorResponse := models.NewErrorResponse(command.ID, client.channelID, &models.SocketError{
				Code:    "HANDLER_ERROR",
				Message: err.Error(),
			})
			client.sendResponse(errorResponse)
			return
		}
		
		client.sendResponse(response)
	}()
}

// sendResponse sends a response back through the connection pool
func (client *UnixSockAPIClient) sendResponse(response *models.SocketResponse) {
	responseData, err := response.ToJSON()
	if err != nil {
		return // Failed to serialize response
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), client.config.ConnectionTimeout)
	defer cancel()
	
	// Send response (fire and forget)
	client.connectionPool.SendMessageNoResponse(ctx, responseData)
}

// Configuration returns the current configuration (read-only)
// Matches Swift: var configuration: UnixSockAPIClientConfig { get }
func (client *UnixSockAPIClient) Configuration() UnixSockAPIClientConfig {
	return client.config
}

// Specification returns the API specification (read-only)
// Matches Swift: var specification: APISpecification { get }
func (client *UnixSockAPIClient) Specification() *specification.APISpecification {
	return client.apiSpec
}

// SocketPathString returns the socket path (read-only)
// Matches Swift: var socketPathString: String { get }
func (client *UnixSockAPIClient) SocketPathString() string {
	return client.socketPath
}

// ChannelIdentifier returns the channel ID (read-only)
// Matches Swift: var channelIdentifier: String { get }
func (client *UnixSockAPIClient) ChannelIdentifier() string {
	return client.channelID
}

// IsListening returns whether the client is currently listening
func (client *UnixSockAPIClient) IsListening() bool {
	client.listenMutex.Lock()
	defer client.listenMutex.Unlock()
	return client.isListening
}

// Close closes the client and cleans up resources
func (client *UnixSockAPIClient) Close() error {
	client.StopListening()
	client.timeoutManager.Close()
	return client.connectionPool.Close()
}