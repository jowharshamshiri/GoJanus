package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jowharshamshiri/GoJanus/pkg/core"
	"github.com/jowharshamshiri/GoJanus/pkg/models"
	"github.com/jowharshamshiri/GoJanus/pkg/specification"
)

func main() {
	var (
		socketPath = flag.String("socket", "/tmp/go-janus.sock", "Unix socket path")
		listen     = flag.Bool("listen", false, "Listen for datagrams on socket")
		sendTo     = flag.String("send-to", "", "Send datagram to socket path")
		command    = flag.String("command", "ping", "Command to send")
		message    = flag.String("message", "hello", "Message to send")
		specPath   = flag.String("spec", "", "Manifest file (required for validation)")
		channelID  = flag.String("channel", "test", "Channel ID for command routing")
	)
	flag.Parse()

	// Load Manifest if provided
	var manifest *specification.Manifest
	if *specPath != "" {
		specData, err := os.ReadFile(*specPath)
		if err != nil {
			log.Fatalf("Failed to read Manifest: %v", err)
		}
		
		parser := specification.NewManifestParser()
		manifest, err = parser.ParseJSON(specData)
		if err != nil {
			log.Fatalf("Failed to parse Manifest: %v", err)
		}
		
		fmt.Printf("Loaded Manifest: %s v%s\n", manifest.Name, manifest.Version)
	}

	if *listen {
		listenForDatagrams(*socketPath, manifest, *channelID)
	} else if *sendTo != "" {
		sendDatagram(*sendTo, *command, *message, manifest, *channelID)
	} else {
		fmt.Println("Usage: either --listen or --send-to required")
		flag.Usage()
		os.Exit(1)
	}
}

func listenForDatagrams(socketPath string, manifest *specification.Manifest, channelID string) {
	fmt.Printf("Listening for SOCK_DGRAM on: %s\n", socketPath)
	if manifest != nil {
		fmt.Printf("API validation enabled for channel: %s\n", channelID)
	}
	
	// Ensure clean socket file
	os.Remove(socketPath)
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	addr, err := net.ResolveUnixAddr("unixgram", socketPath)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
	}
	
	// Cleanup function
	cleanup := func() {
		fmt.Println("\nShutting down server...")
		conn.Close()
		os.Remove(socketPath)
		fmt.Println("Socket cleaned up")
	}
	defer cleanup()
	
	// Handle signals in background
	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()

	// Initialize server state for client tracking
	serverState := NewServerState()
	
	fmt.Println("Ready to receive datagrams")

	for {
		buffer := make([]byte, 64*1024)
		n, clientAddr, err := conn.ReadFromUnix(buffer)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}

		var cmd models.JanusCommand
		if err := json.Unmarshal(buffer[:n], &cmd); err != nil {
			log.Printf("Failed to parse datagram: %v", err)
			continue
		}

		// Track client activity
		clientID := serverState.AddClient(clientAddr.String())
		
		fmt.Printf("Received datagram: %s (ID: %s) from client %s [Total clients: %d]\n", 
			cmd.Command, cmd.ID, clientID, serverState.GetClientCount())

		// Send response via reply_to if specified
		if cmd.ReplyTo != nil && *cmd.ReplyTo != "" {
			sendResponse(cmd.ID, cmd.ChannelID, cmd.Command, cmd.Args, *cmd.ReplyTo, manifest, serverState, clientID)
		}
	}
}

func sendDatagram(targetSocket, command, message string, manifest *specification.Manifest, channelID string) {
	fmt.Printf("Sending SOCK_DGRAM to: %s\n", targetSocket)

	client, err := core.NewJanusClient(targetSocket)
	if err != nil {
		log.Fatalf("Failed to create datagram client: %v", err)
	}

	// Create response socket path
	responseSocket := fmt.Sprintf("/tmp/go-response-%d.sock", os.Getpid())
	
	args := map[string]interface{}{}
	
	// Add arguments based on command type
	if command == "echo" || command == "get_info" || command == "validate" || command == "slow_process" {
		args["message"] = message
	}
	// spec and ping commands don't need message arguments

	// Validate command against Manifest if provided
	// Built-in commands (spec, ping, echo) are always allowed
	builtInCommands := map[string]bool{
		"spec": true,
		"ping": true,
		"echo": true,
		"get_info": true,
		"validate": true,
		"slow_process": true,
	}
	
	if manifest != nil && !builtInCommands[command] {
		if !manifest.HasCommand(channelID, command) {
			log.Fatalf("Command '%s' not found in channel '%s'", command, channelID)
		}
		
		commandSpec, err := manifest.GetCommand(channelID, command)
		if err != nil {
			log.Fatalf("Command validation failed: %v", err)
		}
		
		if err := manifest.ValidateCommandArgs(commandSpec, args); err != nil {
			log.Fatalf("Argument validation failed: %v", err)
		}
		
		fmt.Printf("Command validation passed for %s in channel %s\n", command, channelID)
	} else if builtInCommands[command] {
		fmt.Printf("Built-in command %s allowed\n", command)
	}

	cmd := models.JanusCommand{
		ID:        generateID(),
		ChannelID: channelID,
		Command:   command,
		ReplyTo:   &responseSocket,
		Args:      args,
		Timeout:   func() *float64 { f := 5.0; return &f }(),
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}

	cmdData, err := json.Marshal(cmd)
	if err != nil {
		log.Fatalf("Failed to marshal command: %v", err)
	}

	// Send datagram and wait for response
	ctx := context.Background()
	responseData, err := client.SendDatagram(ctx, cmdData, responseSocket)
	if err != nil {
		log.Fatalf("Failed to send datagram: %v", err)
	}

	var response models.JanusResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		log.Printf("Failed to parse response: %v", err)
		return
	}

	fmt.Printf("Response: Success=%v, Result=%+v\n", response.Success, response.Result)
}

// ClientConnection tracks information about clients that have sent datagrams
type ClientConnection struct {
	ID           string
	Address      string
	CreatedAt    time.Time
	LastActivity time.Time
	MessageCount int
}

// ServerState manages client connections and server metrics
type ServerState struct {
	clients         map[string]*ClientConnection
	clientIDCounter int
	clientsMutex    sync.RWMutex
}

// NewServerState creates a new server state manager
func NewServerState() *ServerState {
	return &ServerState{
		clients: make(map[string]*ClientConnection),
	}
}

// AddClient adds or updates client information based on their source address
func (s *ServerState) AddClient(addr string) string {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()
	
	// Look for existing client by address
	for _, client := range s.clients {
		if client.Address == addr {
			client.LastActivity = time.Now()
			client.MessageCount++
			return client.ID
		}
	}
	
	// Create new client
	s.clientIDCounter++
	clientID := fmt.Sprintf("client-%d", s.clientIDCounter)
	
	client := &ClientConnection{
		ID:           clientID,
		Address:      addr,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		MessageCount: 1,
	}
	
	s.clients[clientID] = client
	return clientID
}

// GetClientCount returns the number of tracked clients
func (s *ServerState) GetClientCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	return len(s.clients)
}

// GetClientInfo returns information about a specific client
func (s *ServerState) GetClientInfo(clientID string) *ClientConnection {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	return s.clients[clientID]
}

// GetAllClients returns information about all clients
func (s *ServerState) GetAllClients() map[string]*ClientConnection {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	
	result := make(map[string]*ClientConnection)
	for id, client := range s.clients {
		// Create a copy to avoid concurrent access issues
		result[id] = &ClientConnection{
			ID:           client.ID,
			Address:      client.Address,
			CreatedAt:    client.CreatedAt,
			LastActivity: client.LastActivity,
			MessageCount: client.MessageCount,
		}
	}
	return result
}

// CommandHandler defines the function signature for command handlers
type CommandHandler func(args map[string]interface{}) (map[string]interface{}, error)

// executeWithTimeout executes a command handler with a timeout
func executeWithTimeout(handler CommandHandler, args map[string]interface{}, timeoutSeconds int) (map[string]interface{}, error) {
	result := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("handler panicked: %v", r)
			}
		}()
		
		res, err := handler(args)
		if err != nil {
			errChan <- err
		} else {
			result <- res
		}
	}()

	timeout := time.Duration(timeoutSeconds) * time.Second
	select {
	case res := <-result:
		return res, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("handler execution timed out after %d seconds", timeoutSeconds)
	}
}

// getBuiltinCommandHandler returns the handler for built-in commands
func getBuiltinCommandHandler(command string, manifest *specification.Manifest, serverState *ServerState, clientID string) CommandHandler {
	switch command {
	case "spec":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			if manifest != nil {
				// Convert Manifest to JSON and back to ensure proper serialization
				specJSON, err := json.Marshal(manifest)
				if err != nil {
					return nil, fmt.Errorf("failed to serialize Manifest: %v", err)
				}
				var specData interface{}
				if err := json.Unmarshal(specJSON, &specData); err != nil {
					return nil, fmt.Errorf("failed to deserialize Manifest: %v", err)
				}
				return map[string]interface{}{
					"specification": specData,
				}, nil
			}
			return nil, fmt.Errorf("no Manifest loaded on server")
		}
	case "ping":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"pong": true,
				"echo": args,
			}, nil
		}
	case "echo":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"echo": args["message"],
			}, nil
		}
	case "get_info":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"implementation": "Go",
				"version":        "1.0.0",
				"protocol":       "SOCK_DGRAM",
				"client_count":   serverState.GetClientCount(),
				"client_id":      clientID,
			}, nil
		}
	case "server_stats":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			clients := serverState.GetAllClients()
			clientStats := make([]map[string]interface{}, 0, len(clients))
			
			for _, client := range clients {
				clientStats = append(clientStats, map[string]interface{}{
					"id":             client.ID,
					"address":        client.Address,
					"created_at":     client.CreatedAt.Unix(),
					"last_activity":  client.LastActivity.Unix(),
					"message_count":  client.MessageCount,
				})
			}
			
			return map[string]interface{}{
				"total_clients": len(clients),
				"clients":       clientStats,
				"server_info": map[string]interface{}{
					"implementation": "Go",
					"version":        "1.0.0",
					"protocol":       "SOCK_DGRAM",
				},
			}, nil
		}
	case "validate":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			// Validate JSON payload
			if message, ok := args["message"]; ok {
				if messageStr, ok := message.(string); ok {
					var jsonData interface{}
					if err := json.Unmarshal([]byte(messageStr), &jsonData); err != nil {
						return map[string]interface{}{
							"valid":  false,
							"error":  "Invalid JSON format",
							"reason": err.Error(),
						}, nil
					}
					return map[string]interface{}{
						"valid": true,
						"data":  jsonData,
					}, nil
				}
				return map[string]interface{}{
					"valid": false,
					"error": "Message must be a string",
				}, nil
			}
			return map[string]interface{}{
				"valid": false,
				"error": "No message provided for validation",
			}, nil
		}
	case "slow_process":
		return func(args map[string]interface{}) (map[string]interface{}, error) {
			// Simulate a slow process that might timeout
			time.Sleep(2 * time.Second) // 2 second delay
			return map[string]interface{}{
				"processed": true,
				"delay":     "2000ms",
				"message":   args["message"],
			}, nil
		}
	default:
		return nil
	}
}

func sendResponse(cmdID, channelID, command string, args map[string]interface{}, replyTo string, manifest *specification.Manifest, serverState *ServerState, clientID string) {
	var result map[string]interface{}
	var success = true
	var errorMsg *models.JSONRPCError

	// Validate command against Manifest if provided
	if manifest != nil {
		if !manifest.HasCommand(channelID, command) {
			success = false
			errorMsg = models.NewJSONRPCError(models.MethodNotFound, fmt.Sprintf("Command '%s' not found in channel '%s'", command, channelID))
		} else {
			// Validate command arguments
			commandSpec, err := manifest.GetCommand(channelID, command)
			if err != nil {
				success = false
				errorMsg = models.NewJSONRPCError(models.ValidationFailed, fmt.Sprintf("Command validation failed: %v", err))
			} else if err := manifest.ValidateCommandArgs(commandSpec, args); err != nil {
				success = false
				errorMsg = models.NewJSONRPCError(models.InvalidParams, fmt.Sprintf("Argument validation failed: %v", err))
			}
		}
	}

	// Only process command if validation passed
	if success {
		// Get built-in command handler
		handler := getBuiltinCommandHandler(command, manifest, serverState, clientID)
		if handler != nil {
			// Execute with timeout (default 30 seconds for built-in commands)
			timeoutSeconds := 30
			if command == "slow_process" {
				timeoutSeconds = 5 // Allow more time for slow_process command
			}
			
			handlerResult, err := executeWithTimeout(handler, args, timeoutSeconds)
			if err != nil {
				success = false
				code := models.InternalError
				if strings.Contains(err.Error(), "timed out") {
					code = models.HandlerTimeout
				}
				errorMsg = models.NewJSONRPCError(code, err.Error())
			} else {
				result = handlerResult
			}
		} else {
			success = false
			errorMsg = models.NewJSONRPCError(models.MethodNotFound, fmt.Sprintf("Unknown command: %s", command))
		}
	}

	response := models.JanusResponse{
		CommandID: cmdID,
		ChannelID: channelID,
		Success:   success,
		Result:    result,
		Error:     errorMsg,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Send response datagram to reply_to socket
	replyAddr, err := net.ResolveUnixAddr("unixgram", replyTo)
	if err != nil {
		log.Printf("Failed to resolve reply address: %v", err)
		return
	}

	conn, err := net.DialUnix("unixgram", nil, replyAddr)
	if err != nil {
		log.Printf("Failed to dial reply socket: %v", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(responseData)
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}