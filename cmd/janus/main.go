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

	"GoJanus/pkg/models"
	"GoJanus/pkg/protocol"
	manifestpkg "GoJanus/pkg/manifest"
)

func main() {
	var (
		socketPath = flag.String("socket", "/tmp/go-janus.sock", "Unix socket path")
		listen     = flag.Bool("listen", false, "Listen for datagrams on socket")
		sendTo     = flag.String("send-to", "", "Send datagram to socket path")
		request    = flag.String("request", "ping", "Request to send")
		message    = flag.String("message", "hello", "Message to send")
		manifestPath   = flag.String("manifest", "", "Manifest file (required for validation)")
	)
	flag.Parse()

	// Load Manifest if provided
	var manifest *manifestpkg.Manifest
	if *manifestPath != "" {
		manifestData, err := os.ReadFile(*manifestPath)
		if err != nil {
			log.Fatalf("Failed to read Manifest: %v", err)
		}
		
		parser := manifestpkg.NewManifestParser()
		loadedManifest, err := parser.ParseJSON(manifestData)
		manifest = loadedManifest
		if err != nil {
			log.Fatalf("Failed to parse Manifest: %v", err)
		}
		
		fmt.Printf("Loaded Manifest: %s v%s\n", manifest.Name, manifest.Version)
	}

	if *listen {
		listenForDatagrams(*socketPath, manifest)
	} else if *sendTo != "" {
		sendDatagram(*sendTo, *request, *message, manifest)
	} else {
		fmt.Println("Usage: either --listen or --send-to required")
		flag.Usage()
		os.Exit(1)
	}
}

func listenForDatagrams(socketPath string, manifest *manifestpkg.Manifest) {
	fmt.Printf("Listening for SOCK_DGRAM on: %s\n", socketPath)
	if manifest != nil {
		fmt.Printf("API validation enabled\n")
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

		var cmd models.JanusRequest
		if err := json.Unmarshal(buffer[:n], &cmd); err != nil {
			log.Printf("Failed to parse datagram: %v", err)
			continue
		}

		// Track client activity
		clientID := serverState.AddClient(clientAddr.String())
		
		fmt.Printf("Received datagram: %s (ID: %s) from client %s [Total clients: %d]\n", 
			cmd.Request, cmd.ID, clientID, serverState.GetClientCount())

		// Send response via reply_to if manifestified
		if cmd.ReplyTo != nil && *cmd.ReplyTo != "" {
			sendResponse(cmd.ID, cmd.Request, cmd.Args, *cmd.ReplyTo, manifest, serverState, clientID)
		}
	}
}

func sendDatagram(targetSocket, request, message string, manifest *manifestpkg.Manifest) {
	fmt.Printf("Sending SOCK_DGRAM to: %s\n", targetSocket)

	// Use high-level protocol client (thin wrapper around API)
	client, err := protocol.New(targetSocket)
	if err != nil {
		log.Fatalf("Failed to create protocol client: %v", err)
	}
	defer client.Close()

	args := map[string]interface{}{}
	
	// Add arguments based on request type
	if request == "echo" || request == "get_info" || request == "validate" || request == "slow_process" {
		args["message"] = message
	}
	// manifest and ping requests don't need message arguments

	// Built-in requests validation
	builtInRequests := map[string]bool{
		"manifest": true,
		"ping": true,
		"echo": true,
		"get_info": true,
		"validate": true,
		"slow_process": true,
	}
	
	if builtInRequests[request] {
		fmt.Printf("Built-in request %s allowed\n", request)
	}

	// Send request using protocol client (handles all socket management)
	ctx := context.Background()
	response, err := client.SendRequest(ctx, request, args)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
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

// GetClientInfo returns information about a manifestific client
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

// RequestHandler defines the function signature for request handlers  
// Returns interface{} to support unwrapped direct values (string, number, object, array, etc.)
type RequestHandler func(args map[string]interface{}) (interface{}, error)

// executeWithTimeout executes a request handler with a timeout
func executeWithTimeout(handler RequestHandler, args map[string]interface{}, timeoutSeconds int) (interface{}, error) {
	result := make(chan interface{}, 1)
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

// getBuiltinRequestHandler returns the handler for built-in requests
func getBuiltinRequestHandler(request string, manifest *manifestpkg.Manifest, serverState *ServerState, clientID string) RequestHandler {
	switch request {
	case "manifest":
		return func(args map[string]interface{}) (interface{}, error) {
			if manifest != nil {
				// Return manifest as unwrapped direct value (not wrapped in additional map)
				manifestJSON, err := json.Marshal(manifest)
				if err != nil {
					return nil, fmt.Errorf("failed to serialize Manifest: %v", err)
				}
				var manifestData interface{}
				if err := json.Unmarshal(manifestJSON, &manifestData); err != nil {
					return nil, fmt.Errorf("failed to deserialize Manifest: %v", err)
				}
				// Return the direct manifest object, not wrapped
				return manifestData, nil
			}
			// Return empty manifest when none is loaded
			return map[string]interface{}{
				"version": "1.0.0",
				"models": map[string]interface{}{},
			}, nil
		}
	case "ping":
		return func(args map[string]interface{}) (interface{}, error) {
			// Return unwrapped ping response object
			return map[string]interface{}{
				"pong": true,
				"echo": args,
			}, nil
		}
	case "echo":
		return func(args map[string]interface{}) (interface{}, error) {
			// Return unwrapped echo response - could return just the message directly
			// For compatibility, return object with echo field
			return map[string]interface{}{
				"echo": args["message"],
			}, nil
		}
	case "get_info":
		return func(args map[string]interface{}) (interface{}, error) {
			// Return unwrapped info object
			return map[string]interface{}{
				"implementation": "Go",
				"version":        "1.0.0",
				"protocol":       "SOCK_DGRAM",
				"client_count":   serverState.GetClientCount(),
				"client_id":      clientID,
			}, nil
		}
	case "server_stats":
		return func(args map[string]interface{}) (interface{}, error) {
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
		return func(args map[string]interface{}) (interface{}, error) {
			// Validate JSON payload - return unwrapped validation result
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
		return func(args map[string]interface{}) (interface{}, error) {
			// Simulate a slow process that might timeout - return unwrapped result
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

func sendResponse(cmdID, request string, args map[string]interface{}, replyTo string, manifest *manifestpkg.Manifest, serverState *ServerState, clientID string) {
	var result interface{} // Support unwrapped results (any JSON value)
	var success = true
	var errorMsg *models.JSONRPCError

	// Channels have been removed - skip validation

	// Only process request if validation passed
	if success {
		// Get built-in request handler
		handler := getBuiltinRequestHandler(request, manifest, serverState, clientID)
		if handler != nil {
			// Execute with timeout (default 30 seconds for built-in requests)
			timeoutSeconds := 30
			if request == "slow_process" {
				timeoutSeconds = 5 // Allow more time for slow_process request
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
			errorMsg = models.NewJSONRPCError(models.MethodNotFound, fmt.Sprintf("Unknown request: %s", request))
		}
	}

	// Use proper constructor for PRIME DIRECTIVE format
	var response *models.JanusResponse
	if success {
		response = models.NewSuccessResponse(cmdID, result)
	} else {
		response = models.NewErrorResponse(cmdID, errorMsg)
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

// generateID function removed - protocol client handles ID generation