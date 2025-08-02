package server

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/user/GoJanus/pkg/models"
)


// EventHandler defines the function signature for event handlers
type EventHandler func(data interface{})

// ServerConfig defines server configuration options
type ServerConfig struct {
	SocketPath        string
	MaxConnections    int
	DefaultTimeout    int
	MaxMessageSize    int
	CleanupOnStart    bool
	CleanupOnShutdown bool
}

// JanusServerEvents defines the available server events
type JanusServerEvents struct {
	Listening     []EventHandler
	Connection    []EventHandler
	Disconnection []EventHandler
	Command       []EventHandler
	Response      []EventHandler
	Error         []EventHandler
}

// JanusServer provides a high-level API for listening on Unix datagram sockets
// SOCK_DGRAM connectionless implementation for cross-language compatibility
type JanusServer struct {
	handlerRegistry *HandlerRegistry
	socketPath      string
	conn            *net.UnixConn
	running         bool
	mutex           sync.RWMutex
	events          *JanusServerEvents
	config          *ServerConfig
}

// NewJanusServer creates a new server instance with event architecture
func NewJanusServer(config *ServerConfig) *JanusServer {
	if config == nil {
		config = &ServerConfig{
			MaxConnections:    100,
			DefaultTimeout:    30,
			MaxMessageSize:    65536,
			CleanupOnStart:    true,
			CleanupOnShutdown: true,
		}
	}
	
	return &JanusServer{
		handlerRegistry: NewHandlerRegistry(),
		running:         false,
		events: &JanusServerEvents{
			Listening:     make([]EventHandler, 0),
			Connection:    make([]EventHandler, 0),
			Disconnection: make([]EventHandler, 0),
			Command:       make([]EventHandler, 0),
			Response:      make([]EventHandler, 0),
			Error:         make([]EventHandler, 0),
		},
		config: config,
	}
}

// Event system methods (EventEmitter pattern)

// On registers an event handler for the specified event type
func (s *JanusServer) On(eventType string, handler EventHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.events == nil {
		return
	}
	
	switch eventType {
	case "listening":
		s.events.Listening = append(s.events.Listening, handler)
	case "connection":
		s.events.Connection = append(s.events.Connection, handler)
	case "disconnection":
		s.events.Disconnection = append(s.events.Disconnection, handler)
	case "command":
		s.events.Command = append(s.events.Command, handler)
	case "response":
		s.events.Response = append(s.events.Response, handler)
	case "error":
		s.events.Error = append(s.events.Error, handler)
	}
}

// Emit triggers all handlers for the specified event type
func (s *JanusServer) Emit(eventType string, data interface{}) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if s.events == nil {
		return
	}
	
	var handlers []EventHandler
	switch eventType {
	case "listening":
		handlers = s.events.Listening
	case "connection":
		handlers = s.events.Connection
	case "disconnection":
		handlers = s.events.Disconnection
	case "command":
		handlers = s.events.Command
	case "response":
		handlers = s.events.Response
	case "error":
		handlers = s.events.Error
	default:
		return
	}
	
	// Execute handlers asynchronously to avoid blocking
	for _, handler := range handlers {
		go handler(data)
	}
}

// CleanupSocketFile removes the socket file if it exists
func (s *JanusServer) CleanupSocketFile() error {
	if s.config == nil || s.config.SocketPath == "" {
		return nil
	}
	
	if _, err := os.Stat(s.config.SocketPath); err == nil {
		return os.Remove(s.config.SocketPath)
	}
	return nil
}

// RegisterHandler registers an enhanced command handler
//
// Example:
//   server.RegisterHandler("ping", NewStringHandler(func(cmd *models.SocketCommand) (string, error) {
//       return "pong", nil
//   }))
func (s *JanusServer) RegisterHandler(command string, handler CommandHandler) {
	s.handlerRegistry.RegisterHandler(command, handler)
}

// StartListening starts the server and begins listening for commands
// This method blocks until the server is stopped
//
// Example:
//   server := NewJanusServer(&ServerConfig{SocketPath: "/tmp/my-server.sock"})
//   server.RegisterHandler("ping", pingHandler)
//   err := server.StartListening()
func (s *JanusServer) StartListening() error {
	if s.config == nil || s.config.SocketPath == "" {
		s.Emit("error", fmt.Errorf("socket path not configured"))
		return fmt.Errorf("socket path not configured")
	}
	
	socketPath := s.config.SocketPath
	
	s.mutex.Lock()
	s.socketPath = socketPath
	s.running = true
	s.mutex.Unlock()

	fmt.Printf("Starting Unix datagram server on: %s\n", socketPath)

	// Cleanup existing socket file if configured
	if s.config.CleanupOnStart {
		if err := s.CleanupSocketFile(); err != nil {
			s.Emit("error", fmt.Errorf("failed to cleanup socket file: %w", err))
			return fmt.Errorf("failed to cleanup socket file: %w", err)
		}
	}

	// Create Unix datagram socket
	addr, err := net.ResolveUnixAddr("unixgram", socketPath)
	if err != nil {
		s.Emit("error", fmt.Errorf("failed to resolve socket address: %w", err))
		return fmt.Errorf("failed to resolve socket address: %w", err)
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		s.Emit("error", fmt.Errorf("failed to bind datagram socket: %w", err))
		return fmt.Errorf("failed to bind datagram socket: %w", err)
	}
	
	s.mutex.Lock()
	s.conn = conn
	s.mutex.Unlock()
	
	defer conn.Close()
	defer func() {
		if s.config.CleanupOnShutdown {
			s.CleanupSocketFile()
		}
	}()

	fmt.Println("Server ready to receive commands")
	s.Emit("listening", nil)

	// Buffer for incoming datagrams
	buffer := make([]byte, 64*1024) // 64KB buffer for datagrams

	for {
		s.mutex.RLock()
		running := s.running
		s.mutex.RUnlock()

		if !running {
			break
		}

		// Set read timeout to allow periodic running checks
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		// Read datagram with sender address
		n, clientAddr, err := conn.ReadFromUnix(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is expected, check running flag
			}
			if s.isRunning() {
				fmt.Printf("Read error: %v\n", err)
			}
			continue
		}

		// Handle datagram in goroutine
		go s.handleDatagram(buffer[:n], clientAddr)
	}

	fmt.Println("Server stopped")
	return nil
}

// Stop stops the server
func (s *JanusServer) Stop() {
	s.mutex.Lock()
	s.running = false
	conn := s.conn
	s.mutex.Unlock()

	if conn != nil {
		conn.Close()
	}
	fmt.Println("Server stop requested")
}

// isRunning checks if server is running (thread-safe)
func (s *JanusServer) isRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// handleDatagram processes a single datagram
// SOCK_DGRAM connectionless implementation
func (s *JanusServer) handleDatagram(data []byte, clientAddr *net.UnixAddr) {
	// Parse command from datagram
	var cmd models.SocketCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		fmt.Printf("Failed to decode command: %v\n", err)
		s.Emit("error", fmt.Errorf("failed to decode command: %w", err))
		return
	}

	fmt.Printf("Received command: %s (ID: %s)\n", cmd.Command, cmd.ID)
	
	// Emit command event
	s.Emit("command", map[string]interface{}{
		"command":  &cmd,
		"clientId": clientAddr.String(),
	})

	// Process command
	response := s.processCommand(&cmd)

	// Send response back to reply_to address if specified
	if cmd.ReplyTo != nil && *cmd.ReplyTo != "" {
		s.sendResponse(response, *cmd.ReplyTo)
		
		// Emit response event
		s.Emit("response", map[string]interface{}{
			"response": response,
			"clientId": clientAddr.String(),
		})
	}
}

// sendResponse sends a response to the specified reply-to address
// SOCK_DGRAM reply mechanism
func (s *JanusServer) sendResponse(response *models.SocketResponse, replyToPath string) {
	// Marshal response to JSON
	responseData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Failed to marshal response: %v\n", err)
		return
	}

	// Resolve reply-to address
	replyAddr, err := net.ResolveUnixAddr("unixgram", replyToPath)
	if err != nil {
		fmt.Printf("Failed to resolve reply address %s: %v\n", replyToPath, err)
		return
	}

	// Create client connection to send response
	conn, err := net.DialUnix("unixgram", nil, replyAddr)
	if err != nil {
		fmt.Printf("Failed to dial reply address %s: %v\n", replyToPath, err)
		return
	}
	defer conn.Close()

	// Send response datagram
	if _, err := conn.Write(responseData); err != nil {
		fmt.Printf("Failed to send response to %s: %v\n", replyToPath, err)
	}
}

// processCommand executes the appropriate handler for a command
func (s *JanusServer) processCommand(cmd *models.SocketCommand) *models.SocketResponse {
	// Check for built-in commands first
	if builtinResult, handled := s.handleBuiltinCommand(cmd); handled {
		return builtinResult
	}

	// Execute handler using enhanced handler registry
	result, err := s.handlerRegistry.ExecuteHandler(cmd.Command, cmd)
	
	var response *models.SocketResponse
	
	if err != nil {
		response = &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   false,
			Result:    nil,
			Error:     err,
			Timestamp: float64(time.Now().Unix()),
		}
	} else {
		response = &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result:    result,
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}
	}

	return response
}

// handleBuiltinCommand handles built-in commands that are always available
func (s *JanusServer) handleBuiltinCommand(cmd *models.SocketCommand) (*models.SocketResponse, bool) {
	switch cmd.Command {
	case "ping":
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result: map[string]interface{}{
				"message":   "pong",
				"timestamp": float64(time.Now().Unix()),
			},
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	case "echo":
		message := ""
		if cmd.Args != nil {
			if msg, exists := cmd.Args["message"]; exists {
				if msgStr, ok := msg.(string); ok {
					message = msgStr
				}
			}
		}
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result: map[string]interface{}{
				"echo":      message,
				"timestamp": float64(time.Now().Unix()),
			},
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	case "get_info":
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result: map[string]interface{}{
				"implementation": "go",
				"version":        "1.0.0",
				"architecture":   "SOCK_DGRAM",
				"timestamp":      float64(time.Now().Unix()),
			},
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	case "spec":
		// Return a basic Manifest matching the expected format
		manifest := map[string]interface{}{
			"version":     "1.0.0",
			"name":        "Go Janus Test API",
			"description": "Test Manifest for Go implementation",
			"channels": map[string]interface{}{
				"test": map[string]interface{}{
					"name":        "Test Channel",
					"description": "Test channel for Go implementation",
					"commands": map[string]interface{}{
						"ping": map[string]interface{}{
							"name":        "Ping",
							"description": "Basic connectivity test",
							"args":        map[string]interface{}{},
							"response": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"message": map[string]interface{}{
										"type":        "string",
										"description": "Pong response",
									},
									"timestamp": map[string]interface{}{
										"type":        "number",
										"description": "Response timestamp",
									},
								},
							},
							"errorCodes": []string{"INTERNAL_ERROR"},
						},
						"echo": map[string]interface{}{
							"name":        "Echo",
							"description": "Echo back the input",
							"args": map[string]interface{}{
								"message": map[string]interface{}{
									"name":        "Message",
									"type":        "string",
									"description": "Message to echo",
									"required":    true,
								},
							},
							"response": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"message": map[string]interface{}{
										"type":        "string",
										"description": "Echoed message",
									},
									"timestamp": map[string]interface{}{
										"type":        "number",
										"description": "Response timestamp",
									},
								},
							},
							"errorCodes": []string{"INVALID_ARGUMENT", "INTERNAL_ERROR"},
						},
					},
				},
			},
			"models": map[string]interface{}{},
		}
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result:    manifest,
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	case "validate":
		// Basic JSON validation
		result := map[string]interface{}{
			"valid":     true,
			"message":   "JSON is valid",
			"timestamp": float64(time.Now().Unix()),
		}
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result:    result,
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	case "slow_process":
		// Simulate slow processing
		time.Sleep(2 * time.Second)
		return &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   true,
			Result: map[string]interface{}{
				"processed":  true,
				"delay":      2,
				"timestamp":  float64(time.Now().Unix()),
			},
			Error:     nil,
			Timestamp: float64(time.Now().Unix()),
		}, true

	default:
		return nil, false // Not a built-in command
	}
}