package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// Debug logger - can be disabled by setting output to io.Discard
	debugLog = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	// Info logger - for important operational messages
	infoLog = log.New(os.Stderr, "[INFO] ", log.LstdFlags)
	// Error logger - for errors
	errorLog = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile)
)

func init() {
	// Disable debug logging by default, enable with GO_JANUS_DEBUG=1
	if os.Getenv("GO_JANUS_DEBUG") == "" {
		debugLog.SetOutput(io.Discard)
	}
}

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
	debugLog.Printf("BindResponseSocket ENTER - Path: %s", responsePath)
	
	// Create UDP-style Unix datagram socket
	debugLog.Printf("Resolving Unix address: %s", responsePath)
	addr, err := net.ResolveUnixAddr("unixgram", responsePath)
	if err != nil {
		errorLog.Printf("Failed to resolve address %s: %v", responsePath, err)
		return nil, fmt.Errorf("failed to resolve response socket address %s: %w", responsePath, err)
	}
	debugLog.Printf("Address resolved successfully: %v", addr)
	
	// Listen on datagram socket for responses
	log.Printf("[GO-CLIENT] Attempting to bind socket at: %s", responsePath)
	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to bind socket at %s: %v", responsePath, err)
		return nil, fmt.Errorf("failed to bind response socket at %s: %w", responsePath, err)
	}
	log.Printf("[GO-CLIENT] Socket bound successfully at: %s", responsePath)
	
	// Verify socket file exists
	if _, err := os.Stat(responsePath); err == nil {
		log.Printf("[GO-CLIENT] ✅ Socket file verified on filesystem: %s", responsePath)
	} else {
		log.Printf("[GO-CLIENT] ❌ Socket file NOT found on filesystem: %s (error: %v)", responsePath, err)
	}
	
	log.Printf("[GO-CLIENT] BindResponseSocket EXIT - Path: %s", responsePath)
	return conn, nil
}

// CloseSocket closes a datagram socket and cleans up the socket file
// Connectionless SOCK_DGRAM implementation
func (udc *JanusClient) CloseSocket(conn net.Conn, socketPath string) error {
	log.Printf("[GO-CLIENT] CloseSocket ENTER - Path: %s", socketPath)
	
	if conn == nil {
		log.Printf("[GO-CLIENT] CloseSocket - Connection is nil, nothing to close")
		return nil // Already closed
	}
	
	// Check if socket file exists before closing
	if _, err := os.Stat(socketPath); err == nil {
		log.Printf("[GO-CLIENT] Socket file exists before closing: %s", socketPath)
	} else {
		log.Printf("[GO-CLIENT] Socket file already missing before closing: %s (error: %v)", socketPath, err)
	}
	
	// Close the socket connection
	log.Printf("[GO-CLIENT] Closing connection for: %s", socketPath)
	err := conn.Close()
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR closing connection: %v", err)
	} else {
		log.Printf("[GO-CLIENT] Connection closed successfully")
	}
	
	// Remove the socket file from filesystem
	if socketPath != "" {
		log.Printf("[GO-CLIENT] Removing socket file: %s", socketPath)
		if removeErr := os.Remove(socketPath); removeErr != nil {
			log.Printf("[GO-CLIENT] ERROR removing socket file %s: %v", socketPath, removeErr)
		} else {
			log.Printf("[GO-CLIENT] Socket file removed successfully: %s", socketPath)
		}
	}
	
	log.Printf("[GO-CLIENT] CloseSocket EXIT - Path: %s", socketPath)
	return err
}

// SendDatagram sends data via connectionless Unix datagram socket
// SOCK_DGRAM implementation for connectionless communication
func (udc *JanusClient) SendDatagram(ctx context.Context, data []byte, responsePath string) ([]byte, error) {
	log.Printf("[GO-CLIENT] SendDatagram START - Response socket path: %s", responsePath)
	log.Printf("[GO-CLIENT] Server socket path: %s", udc.socketPath)
	log.Printf("[GO-CLIENT] Request data size: %d bytes", len(data))
	
	// Validate message data using security validator
	if err := udc.validator.ValidateMessageData(data); err != nil {
		log.Printf("[GO-CLIENT] ERROR: Message validation failed: %v", err)
		return nil, fmt.Errorf("message validation failed: %w", err)
	}
	log.Printf("[GO-CLIENT] Message validation passed")
	
	// Create response socket for receiving reply
	log.Printf("[GO-CLIENT] Creating response socket at: %s", responsePath)
	responseConn, err := udc.BindResponseSocket(ctx, responsePath)
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to bind response socket at %s: %v", responsePath, err)
		return nil, fmt.Errorf("failed to bind response socket: %w", err)
	}
	log.Printf("[GO-CLIENT] SUCCESS: Response socket bound at %s", responsePath)
	
	// CRITICAL: Don't close response socket until AFTER we receive response
	// The defer was causing premature socket cleanup
	
	// Resolve server socket address
	log.Printf("[GO-CLIENT] Resolving server address: %s", udc.socketPath)
	serverAddr, err := net.ResolveUnixAddr("unixgram", udc.socketPath)
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to resolve server address %s: %v", udc.socketPath, err)
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to resolve server address %s: %w", udc.socketPath, err)
	}
	log.Printf("[GO-CLIENT] Server address resolved: %s", serverAddr)
	
	// Create client socket for sending request
	log.Printf("[GO-CLIENT] Dialing server socket: %s", udc.socketPath)
	clientConn, err := net.DialUnix("unixgram", nil, serverAddr)
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to dial server socket: %v", err)
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to dial server socket: %w", err)
	}
	log.Printf("[GO-CLIENT] SUCCESS: Connected to server socket")
	
	defer func() {
		log.Printf("[GO-CLIENT] Closing client connection")
		clientConn.Close()
	}()
	
	// Set write timeout
	writeDeadline := time.Now().Add(udc.datagramTimeout)
	log.Printf("[GO-CLIENT] Setting write deadline: %v (timeout: %v)", writeDeadline, udc.datagramTimeout)
	if err := clientConn.SetWriteDeadline(writeDeadline); err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to set write deadline: %v", err)
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}
	
	// Send datagram (no framing needed for SOCK_DGRAM)
	log.Printf("[GO-CLIENT] Sending datagram of %d bytes to server...", len(data))
	if _, err := clientConn.Write(data); err != nil {
		// Check for message too long error
		if strings.Contains(err.Error(), "message too long") {
			log.Printf("[GO-CLIENT] ERROR: Message too long (%d bytes)", len(data))
			// Cleanup response socket on error
			log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
			udc.CloseSocket(responseConn, responsePath)
			return nil, fmt.Errorf("payload too large for SOCK_DGRAM (size: %d bytes): Unix domain datagram sockets have system-imposed size limits, typically around 64KB. Consider reducing payload size or using chunked messages", len(data))
		}
		log.Printf("[GO-CLIENT] ERROR: Failed to write datagram: %v", err)
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to send datagram: %w", err)
	}
	log.Printf("[GO-CLIENT] SUCCESS: Datagram sent to server, waiting for response on %s", responsePath)
	
	// Set read timeout for response
	readDeadline := time.Now().Add(udc.datagramTimeout)
	log.Printf("[GO-CLIENT] Setting read deadline: %v (timeout: %v)", readDeadline, udc.datagramTimeout)
	if err := responseConn.SetReadDeadline(readDeadline); err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to set read deadline: %v", err)
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	
	// Read response datagram
	buffer := make([]byte, udc.maxMessageSize)
	log.Printf("[GO-CLIENT] Reading response from %s...", responsePath)
	
	// Check if socket file still exists before reading
	if _, err := os.Stat(responsePath); err == nil {
		log.Printf("[GO-CLIENT] ✅ Socket file exists before read: %s", responsePath)
	} else {
		log.Printf("[GO-CLIENT] ❌ Socket file missing before read: %s (error: %v)", responsePath, err)
	}
	
	n, err := responseConn.Read(buffer)
	if err != nil {
		log.Printf("[GO-CLIENT] ERROR: Failed to read response from %s: %v", responsePath, err)
		
		// Check if socket file still exists after read error
		if _, statErr := os.Stat(responsePath); statErr == nil {
			log.Printf("[GO-CLIENT] Socket file still exists after read error: %s", responsePath)
		} else {
			log.Printf("[GO-CLIENT] Socket file missing after read error: %s (stat error: %v)", responsePath, statErr)
		}
		
		// Cleanup response socket on error
		log.Printf("[GO-CLIENT] CLEANUP (ERROR): Closing response socket %s", responsePath)
		udc.CloseSocket(responseConn, responsePath)
		return nil, fmt.Errorf("failed to read response datagram: %w", err)
	}
	log.Printf("[GO-CLIENT] SUCCESS: Received response of %d bytes from %s", n, responsePath)
	
	// NOW it's safe to cleanup response socket after receiving response
	log.Printf("[GO-CLIENT] CLEANUP (SUCCESS): Closing response socket %s", responsePath)
	udc.CloseSocket(responseConn, responsePath)
	
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