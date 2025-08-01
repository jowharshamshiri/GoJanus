package core

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// JanusClient provides low-level Unix domain datagram socket communication  
// SOCK_DGRAM connectionless implementation for cross-language compatibility
type JanusClient struct {
	socketPath        string
	maxMessageSize    int
	datagramTimeout   time.Duration
	validator        *SecurityValidator
	messageHandlers  []func([]byte)
	handlerMutex     sync.RWMutex
}

// JanusClientConfig holds configuration options for the datagram client
// SOCK_DGRAM configuration structure
type JanusClientConfig struct {
	MaxMessageSize   int
	DatagramTimeout  time.Duration
}

// DefaultJanusClientConfig returns default configuration for SOCK_DGRAM
func DefaultJanusClientConfig() JanusClientConfig {
	return JanusClientConfig{
		MaxMessageSize:  64 * 1024,       // 64KB datagram limit
		DatagramTimeout: 5 * time.Second, // 5s timeout
	}
}

// NewJanusClient creates a new Unix datagram client
// SOCK_DGRAM connectionless implementation
func NewJanusClient(socketPath string, config ...JanusClientConfig) (*JanusClient, error) {
	cfg := DefaultJanusClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	validator := NewSecurityValidator()
	
	// Validate socket path using security validator
	if err := validator.ValidateSocketPath(socketPath); err != nil {
		return nil, fmt.Errorf("invalid socket path: %w", err)
	}
	
	return &JanusClient{
		socketPath:        socketPath,
		maxMessageSize:    cfg.MaxMessageSize,
		datagramTimeout:   cfg.DatagramTimeout,
		validator:        validator,
		messageHandlers:  make([]func([]byte), 0),
	}, nil
}

// BindResponseSocket creates a datagram socket for receiving responses
// Connectionless SOCK_DGRAM implementation
func (udc *JanusClient) BindResponseSocket(ctx context.Context, responsePath string) (net.Conn, error) {
	
	// Create UDP-style Unix datagram socket
	addr, err := net.ResolveUnixAddr("unixgram", responsePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve response socket address %s: %w", responsePath, err)
	}
	
	// Listen on datagram socket for responses
	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind response socket at %s: %w", responsePath, err)
	}
	
	return conn, nil
}

// CloseSocket closes a datagram socket and cleans up the socket file
// Connectionless SOCK_DGRAM implementation
func (udc *JanusClient) CloseSocket(conn net.Conn, socketPath string) error {
	
	if conn == nil {
		return nil // Already closed
	}
	
	// Close the socket connection
	err := conn.Close()
	
	// Remove the socket file from filesystem
	if socketPath != "" {
		os.Remove(socketPath) // Best effort cleanup
	}
	
	return err
}

// SendDatagram sends data via connectionless Unix datagram socket
// SOCK_DGRAM implementation for connectionless communication
func (udc *JanusClient) SendDatagram(ctx context.Context, data []byte, responsePath string) ([]byte, error) {
	// Validate message data using security validator
	if err := udc.validator.ValidateMessageData(data); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}
	
	// Create response socket for receiving reply
	responseConn, err := udc.BindResponseSocket(ctx, responsePath)
	if err != nil {
		return nil, fmt.Errorf("failed to bind response socket: %w", err)
	}
	defer udc.CloseSocket(responseConn, responsePath)
	
	// Resolve server socket address
	serverAddr, err := net.ResolveUnixAddr("unixgram", udc.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address %s: %w", udc.socketPath, err)
	}
	
	// Create client socket for sending command
	clientConn, err := net.DialUnix("unixgram", nil, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial server socket: %w", err)
	}
	defer clientConn.Close()
	
	// Set write timeout
	if err := clientConn.SetWriteDeadline(time.Now().Add(udc.datagramTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}
	
	// Send datagram (no framing needed for SOCK_DGRAM)
	if _, err := clientConn.Write(data); err != nil {
		// Check for message too long error
		if strings.Contains(err.Error(), "message too long") {
			return nil, fmt.Errorf("payload too large for SOCK_DGRAM (size: %d bytes): Unix domain datagram sockets have system-imposed size limits, typically around 64KB. Consider reducing payload size or using chunked messages", len(data))
		}
		return nil, fmt.Errorf("failed to send datagram: %w", err)
	}
	
	// Set read timeout for response
	if err := responseConn.SetReadDeadline(time.Now().Add(udc.datagramTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	
	// Read response datagram
	buffer := make([]byte, udc.maxMessageSize)
	n, err := responseConn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response datagram: %w", err)
	}
	
	return buffer[:n], nil
}

// SendDatagramNoResponse sends datagram without expecting a response
// Fire-and-forget pattern for SOCK_DGRAM
func (udc *JanusClient) SendDatagramNoResponse(ctx context.Context, data []byte) error {
	// Validate message data using security validator
	if err := udc.validator.ValidateMessageData(data); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}
	
	// Resolve server socket address
	serverAddr, err := net.ResolveUnixAddr("unixgram", udc.socketPath)
	if err != nil {
		return fmt.Errorf("failed to resolve server address %s: %w", udc.socketPath, err)
	}
	
	// Create client socket for sending datagram
	clientConn, err := net.DialUnix("unixgram", nil, serverAddr)
	if err != nil {
		return fmt.Errorf("failed to dial server socket: %w", err)
	}
	defer clientConn.Close()
	
	// Set write timeout
	if err := clientConn.SetWriteDeadline(time.Now().Add(udc.datagramTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	
	// Send datagram (no framing needed for SOCK_DGRAM)
	if _, err := clientConn.Write(data); err != nil {
		// Check for message too long error
		if strings.Contains(err.Error(), "message too long") {
			return fmt.Errorf("payload too large for SOCK_DGRAM (size: %d bytes): Unix domain datagram sockets have system-imposed size limits, typically around 64KB. Consider reducing payload size or using chunked messages", len(data))
		}
		return fmt.Errorf("failed to send datagram: %w", err)
	}
	
	return nil
}

// TestDatagramSocket tests the datagram socket connectivity
// SOCK_DGRAM connectivity test
func (udc *JanusClient) TestDatagramSocket(ctx context.Context) error {
	// Resolve server socket address
	serverAddr, err := net.ResolveUnixAddr("unixgram", udc.socketPath)
	if err != nil {
		return fmt.Errorf("failed to resolve server address %s: %w", udc.socketPath, err)
	}
	
	// Try to create client socket
	clientConn, err := net.DialUnix("unixgram", nil, serverAddr)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer clientConn.Close()
	
	return nil
}

// AddMessageHandler adds a handler for incoming messages
// SOCK_DGRAM message handler pattern
func (udc *JanusClient) AddMessageHandler(handler func([]byte)) {
	udc.handlerMutex.Lock()
	defer udc.handlerMutex.Unlock()
	
	udc.messageHandlers = append(udc.messageHandlers, handler)
}

// RemoveAllMessageHandlers removes all message handlers
// SOCK_DGRAM handler cleanup
func (udc *JanusClient) RemoveAllMessageHandlers() {
	udc.handlerMutex.Lock()
	defer udc.handlerMutex.Unlock()
	
	udc.messageHandlers = make([]func([]byte), 0)
}

// GenerateResponseSocketPath generates a unique response socket path
// Used for SOCK_DGRAM reply-to mechanism
func (udc *JanusClient) GenerateResponseSocketPath() string {
	timestamp := time.Now().UnixNano()
	pid := os.Getpid()
	return fmt.Sprintf("/tmp/go_janus_client_%d_%d.sock", pid, timestamp)
}

// MaximumMessageSize returns the maximum message size (read-only property)
// SOCK_DGRAM datagram size limit
func (udc *JanusClient) MaximumMessageSize() int {
	return udc.maxMessageSize
}

// SocketPath returns the socket path (read-only property)
func (udc *JanusClient) SocketPath() string {
	return udc.socketPath
}