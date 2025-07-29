package core

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// UnixSocketClient provides low-level Unix domain socket communication
// Matches Swift UnixSocketClient functionality exactly for cross-language compatibility
type UnixSocketClient struct {
	socketPath        string
	maxMessageSize    int
	connectionTimeout time.Duration
	framing          *MessageFraming
	validator        *SecurityValidator
	messageHandlers  []func([]byte)
	handlerMutex     sync.RWMutex
	conn             net.Conn
	connMutex        sync.Mutex
}

// UnixSocketClientConfig holds configuration options for the Unix socket client
// Matches Swift configuration structure
type UnixSocketClientConfig struct {
	MaxMessageSize    int
	ConnectionTimeout time.Duration
}

// DefaultUnixSocketClientConfig returns default configuration matching Swift defaults
func DefaultUnixSocketClientConfig() UnixSocketClientConfig {
	return UnixSocketClientConfig{
		MaxMessageSize:    10 * 1024 * 1024, // 10MB matches Swift
		ConnectionTimeout: 5 * time.Second,   // 5s matches Swift
	}
}

// NewUnixSocketClient creates a new Unix socket client
// Matches Swift: init(socketPath: String, maxMessageSize: Int, connectionTimeout: TimeInterval)
func NewUnixSocketClient(socketPath string, config ...UnixSocketClientConfig) (*UnixSocketClient, error) {
	cfg := DefaultUnixSocketClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	validator := NewSecurityValidator()
	
	// Validate socket path using security validator
	if err := validator.ValidateSocketPath(socketPath); err != nil {
		return nil, fmt.Errorf("invalid socket path: %w", err)
	}
	
	return &UnixSocketClient{
		socketPath:        socketPath,
		maxMessageSize:    cfg.MaxMessageSize,
		connectionTimeout: cfg.ConnectionTimeout,
		framing:          NewMessageFraming(cfg.MaxMessageSize),
		validator:        validator,
		messageHandlers:  make([]func([]byte), 0),
	}, nil
}

// Connect establishes a connection to the Unix socket
// Matches Swift: func connect() async throws
func (usc *UnixSocketClient) Connect(ctx context.Context) error {
	usc.connMutex.Lock()
	defer usc.connMutex.Unlock()
	
	if usc.conn != nil {
		return nil // Already connected
	}
	
	// Create context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, usc.connectionTimeout)
	defer cancel()
	
	// Dial Unix socket with context
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(connectCtx, "unix", usc.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Unix socket at %s: %w", usc.socketPath, err)
	}
	
	usc.conn = conn
	return nil
}

// Disconnect closes the connection to the Unix socket
// Matches Swift: func disconnect()
func (usc *UnixSocketClient) Disconnect() error {
	usc.connMutex.Lock()
	defer usc.connMutex.Unlock()
	
	if usc.conn == nil {
		return nil // Already disconnected
	}
	
	err := usc.conn.Close()
	usc.conn = nil
	return err
}

// Send sends data through the Unix socket and returns the response
// Matches Swift: func send(_ data: Data) async throws -> Data
func (usc *UnixSocketClient) Send(ctx context.Context, data []byte) ([]byte, error) {
	// Validate message data using security validator
	if err := usc.validator.ValidateMessageData(data); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}
	
	// Validate message format
	if err := usc.framing.ValidateMessageFormat(data); err != nil {
		return nil, fmt.Errorf("message format validation failed: %w", err)
	}
	
	// Connect if not already connected
	if err := usc.Connect(ctx); err != nil {
		return nil, err
	}
	
	usc.connMutex.Lock()
	conn := usc.conn
	usc.connMutex.Unlock()
	
	if conn == nil {
		return nil, fmt.Errorf("not connected to socket")
	}
	
	// Set write timeout
	if err := conn.SetWriteDeadline(time.Now().Add(usc.connectionTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}
	
	// Send message with framing
	if err := usc.framing.WriteMessage(conn, data); err != nil {
		usc.Disconnect() // Close connection on error
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	
	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(usc.connectionTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	
	// Read response with framing
	response, err := usc.framing.ReadMessage(conn)
	if err != nil {
		usc.Disconnect() // Close connection on error
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Validate response format
	if err := usc.framing.ValidateMessageFormat(response); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}
	
	return response, nil
}

// SendNoResponse sends data through the Unix socket without expecting a response
// Useful for fire-and-forget commands (matches Swift publish pattern)
func (usc *UnixSocketClient) SendNoResponse(ctx context.Context, data []byte) error {
	// Validate message data using security validator
	if err := usc.validator.ValidateMessageData(data); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}
	
	// Validate message format
	if err := usc.framing.ValidateMessageFormat(data); err != nil {
		return fmt.Errorf("message format validation failed: %w", err)
	}
	
	// Create temporary connection for stateless communication
	connectCtx, cancel := context.WithTimeout(ctx, usc.connectionTimeout)
	defer cancel()
	
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(connectCtx, "unix", usc.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Unix socket: %w", err)
	}
	defer conn.Close()
	
	// Set write timeout
	if err := conn.SetWriteDeadline(time.Now().Add(usc.connectionTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	
	// Send message with framing
	if err := usc.framing.WriteMessage(conn, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	return nil
}

// TestConnection tests the connection to the Unix socket
// Matches Swift test connection functionality
func (usc *UnixSocketClient) TestConnection(ctx context.Context) error {
	connectCtx, cancel := context.WithTimeout(ctx, usc.connectionTimeout)
	defer cancel()
	
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(connectCtx, "unix", usc.socketPath)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer conn.Close()
	
	return nil
}

// AddMessageHandler adds a handler for incoming messages
// Matches Swift: func addMessageHandler(_ handler: @escaping (Data) -> Void)
func (usc *UnixSocketClient) AddMessageHandler(handler func([]byte)) {
	usc.handlerMutex.Lock()
	defer usc.handlerMutex.Unlock()
	
	usc.messageHandlers = append(usc.messageHandlers, handler)
}

// RemoveAllMessageHandlers removes all message handlers
// Matches Swift: func removeAllMessageHandlers()
func (usc *UnixSocketClient) RemoveAllMessageHandlers() {
	usc.handlerMutex.Lock()
	defer usc.handlerMutex.Unlock()
	
	usc.messageHandlers = make([]func([]byte), 0)
}

// StartListening starts listening for incoming messages (for persistent connections)
// Matches Swift persistent listening functionality
func (usc *UnixSocketClient) StartListening(ctx context.Context) error {
	// Connect if not already connected
	if err := usc.Connect(ctx); err != nil {
		return err
	}
	
	usc.connMutex.Lock()
	conn := usc.conn
	usc.connMutex.Unlock()
	
	if conn == nil {
		return fmt.Errorf("not connected to socket")
	}
	
	// Start goroutine to handle incoming messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Read message with framing
				message, err := usc.framing.ReadMessage(conn)
				if err != nil {
					// Connection error, break the loop
					usc.Disconnect()
					return
				}
				
				// Validate message format
				if err := usc.framing.ValidateMessageFormat(message); err != nil {
					continue // Skip invalid messages
				}
				
				// Call all message handlers
				usc.handlerMutex.RLock()
				handlers := make([]func([]byte), len(usc.messageHandlers))
				copy(handlers, usc.messageHandlers)
				usc.handlerMutex.RUnlock()
				
				for _, handler := range handlers {
					go handler(message) // Handle each message in separate goroutine
				}
			}
		}
	}()
	
	return nil
}

// MaximumMessageSize returns the maximum message size (read-only property)
// Matches Swift: var maximumMessageSize: Int { get }
func (usc *UnixSocketClient) MaximumMessageSize() int {
	return usc.maxMessageSize
}

// SocketPath returns the socket path (read-only property)
func (usc *UnixSocketClient) SocketPath() string {
	return usc.socketPath
}

// IsConnected returns whether the client is currently connected
func (usc *UnixSocketClient) IsConnected() bool {
	usc.connMutex.Lock()
	defer usc.connMutex.Unlock()
	
	return usc.conn != nil
}

// ReceiveMessage receives a message from the Unix socket (for async listening)
func (usc *UnixSocketClient) ReceiveMessage(ctx context.Context) ([]byte, error) {
	// Connect if not already connected
	if err := usc.Connect(ctx); err != nil {
		return nil, err
	}
	
	usc.connMutex.Lock()
	conn := usc.conn
	usc.connMutex.Unlock()
	
	if conn == nil {
		return nil, fmt.Errorf("not connected to socket")
	}
	
	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(usc.connectionTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	
	// Read response with framing
	response, err := usc.framing.ReadMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	
	// Validate response format
	if err := usc.framing.ValidateMessageFormat(response); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}
	
	return response, nil
}