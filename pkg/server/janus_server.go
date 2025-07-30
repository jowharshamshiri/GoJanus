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

// CommandHandler defines the function signature for command handlers
type CommandHandler func(*models.SocketCommand) (interface{}, *models.SocketError)

// JanusServer provides a high-level API for listening on Unix sockets
type JanusServer struct {
	handlers   map[string]CommandHandler
	socketPath string
	listener   net.Listener
	running    bool
	mutex      sync.RWMutex
}

// NewJanusServer creates a new server instance (DEPRECATED: use JanusServer{} directly)
func NewJanusServer() *JanusServer {
	return &JanusServer{
		handlers: make(map[string]CommandHandler),
		running:  false,
	}
}

// RegisterHandler registers a command handler
//
// Example:
//   server.RegisterHandler("ping", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
//       return map[string]interface{}{"message": "pong", "timestamp": time.Now().Unix()}, nil
//   })
func (s *JanusServer) RegisterHandler(command string, handler CommandHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.handlers == nil {
		s.handlers = make(map[string]CommandHandler)
	}
	s.handlers[command] = handler
}

// StartListening starts the server and begins listening for commands
// This method blocks until the server is stopped
//
// Example:
//   server := &JanusServer{}
//   server.RegisterHandler("ping", pingHandler)
//   err := server.StartListening("/tmp/my-server.sock")
func (s *JanusServer) StartListening(socketPath string) error {
	s.mutex.Lock()
	s.socketPath = socketPath
	s.running = true
	s.mutex.Unlock()

	fmt.Printf("Starting Unix socket server on: %s\n", socketPath)

	// Remove existing socket file
	os.Remove(socketPath)

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind socket: %w", err)
	}
	s.listener = listener
	defer listener.Close()
	defer os.Remove(socketPath)

	fmt.Println("Server ready to receive commands")

	for {
		s.mutex.RLock()
		running := s.running
		s.mutex.RUnlock()

		if !running {
			break
		}

		// Accept connections (Go doesn't have direct SOCK_DGRAM listen, so we simulate with short-lived connections)
		conn, err := listener.Accept()
		if err != nil {
			if s.isRunning() {
				fmt.Printf("Accept error: %v\n", err)
			}
			continue
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}

	fmt.Println("Server stopped")
	return nil
}

// Stop stops the server
func (s *JanusServer) Stop() {
	s.mutex.Lock()
	s.running = false
	s.mutex.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}
	fmt.Println("Server stop requested")
}

// isRunning checks if server is running (thread-safe)
func (s *JanusServer) isRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// handleConnection processes a single connection
func (s *JanusServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read command
	decoder := json.NewDecoder(conn)
	var cmd models.SocketCommand
	if err := decoder.Decode(&cmd); err != nil {
		fmt.Printf("Failed to decode command: %v\n", err)
		return
	}

	fmt.Printf("Received command: %s (ID: %s)\n", cmd.Command, cmd.ID)

	// Process command
	response := s.processCommand(&cmd)

	// Send response back through the same connection
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(response); err != nil {
		fmt.Printf("Failed to send response: %v\n", err)
	}
}

// processCommand executes the appropriate handler for a command
func (s *JanusServer) processCommand(cmd *models.SocketCommand) *models.SocketResponse {
	s.mutex.RLock()
	handler, exists := s.handlers[cmd.Command]
	s.mutex.RUnlock()

	var response *models.SocketResponse

	if exists {
		// Execute handler
		result, err := handler(cmd)
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
	} else {
		// Command not found
		response = &models.SocketResponse{
			CommandID: cmd.ID,
			ChannelID: cmd.ChannelID,
			Success:   false,
			Result:    nil,
			Error: &models.SocketError{
				Code:    "COMMAND_NOT_FOUND",
				Message: fmt.Sprintf("Command '%s' not registered", cmd.Command),
				Details: "",
			},
			Timestamp: float64(time.Now().Unix()),
		}
	}

	return response
}