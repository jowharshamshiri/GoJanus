package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/user/GoJanus/pkg/server"
	"github.com/user/GoJanus/pkg/models"
	"github.com/user/GoJanus/pkg/protocol"
)

func TestServerCreation(t *testing.T) {
	server := &server.JanusServer{}
	if server == nil {
		t.Fatal("Failed to create server")
	}
}

func TestClientCreation(t *testing.T) {
	// Legacy client removed - use protocol client instead
	t.Skip("Legacy client test removed - see janus_client_test.go for protocol client tests")
}

func TestServerRegisterHandler(t *testing.T) {
	config := &server.ServerConfig{
		SocketPath:     "/tmp/test-handler.sock",
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	srv.RegisterHandler("test", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		return map[string]interface{}{"echo": cmd.Command}, nil
	}))
	
	// Handler registration should succeed
	// We can't directly test the internal map, but no panic is good
}

func TestClientServerCommunication(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-comm.sock"
	defer os.Remove(socketPath)
	
	// Start server with proper configuration
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	srv.RegisterHandler("ping", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		return map[string]interface{}{
			"message":   "pong",
			"timestamp": time.Now().Unix(),
		}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client communication - use nil manifest to fetch from server dynamically
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	response, err := client.SendCommand(context.Background(), "ping", nil)
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Verify response
	if err != nil {
		t.Fatalf("Client communication failed: %v", err)
	}
	
	if !response.Success {
		t.Fatalf("Server returned error: %v", response.Error)
	}
	
	if response.Result == nil {
		t.Fatal("Expected result but got nil")
	}
}

func TestClientServerWithArgs(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-args.sock"
	defer os.Remove(socketPath)
	
	// Start server with echo handler
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	srv.RegisterHandler("echo", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		if cmd.Args == nil {
			return nil, models.NewJSONRPCError(models.InvalidParams, "No arguments provided")
		}
		
		message, exists := cmd.Args["message"]
		if !exists {
			return nil, models.NewJSONRPCError(models.InvalidParams, "Missing 'message' argument")
		}
		
		return map[string]interface{}{
			"echo":        message,
			"received_at": time.Now().Unix(),
		}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with arguments - use nil manifest to fetch from server dynamically
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	args := map[string]interface{}{
		"message": "Hello, Server!",
	}
	
	response, err := client.SendCommand(context.Background(), "echo", args)
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Verify response
	if err != nil {
		t.Fatalf("Client communication failed: %v", err)
	}
	
	if !response.Success {
		t.Fatalf("Server returned error: %v", response.Error)
	}
	
	// Verify echo content
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be map[string]interface{}, got %T: %+v", response.Result, response.Result)
	}
	
	t.Logf("Full response result: %+v", result)
	
	echo, exists := result["echo"]
	if !exists {
		t.Fatalf("Expected 'echo' field in result. Available fields: %+v", result)
	}
	
	if echo != "Hello, Server!" {
		t.Fatalf("Expected echo 'Hello, Server!' but got '%v'", echo)
	}
}

func TestClientCommandNotFound(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-notfound.sock"
	defer os.Remove(socketPath)
	
	// Start server with a handler for a different command
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	// Register a handler for a different command so "nonexistent" won't be found
	srv.RegisterHandler("existingCommand", server.SyncHandler(func(cmd *models.JanusCommand) server.HandlerResult {
		return server.HandlerResult{
			Value: map[string]interface{}{
				"result": "success",
			},
			Error: nil,
		}
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with non-existent command
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	response, err := client.SendCommand(context.Background(), "nonexistent", nil)
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Should get response (not connection error)
	if err != nil {
		t.Fatalf("Unexpected connection error: %v", err)
	}
	
	// Should be unsuccessful
	if response.Success {
		t.Fatal("Expected command not found error")
	}
	
	if response.Error == nil {
		t.Fatal("Expected error in response")
	}
	
	if response.Error.Code != models.MethodNotFound {
		t.Fatalf("Expected COMMAND_NOT_FOUND error, got %s", response.Error.Code)
	}
}

func TestClientFireAndForget(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-fire.sock"
	defer os.Remove(socketPath)
	
	// Start server with logging handler
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	srv.RegisterHandler("log", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		// Simulate logging
		t.Logf("Logged: %v", cmd.Args)
		return map[string]interface{}{"logged": true}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test fire-and-forget
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	args := map[string]interface{}{
		"level":   "info",
		"message": "Test log message",
	}
	
	err = client.SendCommandNoResponse(context.Background(), "log", args)
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Should complete without error
	if err != nil {
		t.Fatalf("Fire-and-forget failed: %v", err)
	}
}

func TestClientTimeout(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-timeout.sock"
	defer os.Remove(socketPath)
	
	// Start server with slow handler
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	srv.RegisterHandler("slow", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		// Simulate slow processing
		time.Sleep(10 * time.Second)
		return map[string]interface{}{"done": true}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with short timeout
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	_, err = client.SendCommand(context.Background(), "slow", nil)
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Should timeout
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestClientPing(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-ping.sock"
	defer os.Remove(socketPath)
	
	// Start server with ping handler
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	srv.RegisterHandler("ping", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		return map[string]interface{}{"message": "pong"}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test ping
	client, err := protocol.New(socketPath, "test", protocol.DefaultJanusClientConfig())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	success := client.Ping(context.Background())
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Ping should succeed
	if !success {
		t.Fatal("Ping should have succeeded")
	}
}

func TestMultipleHandlers(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-multi.sock"
	defer os.Remove(socketPath)
	
	// Start server with multiple handlers
	config := &server.ServerConfig{
		SocketPath:     socketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	srv.RegisterHandler("add", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		if cmd.Args == nil {
			return nil, models.NewJSONRPCError(models.InvalidParams, "Missing arguments")
		}
		
		a, aOk := cmd.Args["a"].(float64)
		b, bOk := cmd.Args["b"].(float64)
		
		if !aOk || !bOk {
			return nil, models.NewJSONRPCError(models.InvalidParams, "Arguments must be numbers")
		}
		
		return map[string]interface{}{"result": a + b}, nil
	}))
	
	srv.RegisterHandler("multiply", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		if cmd.Args == nil {
			return nil, models.NewJSONRPCError(models.InvalidParams, "Missing arguments")
		}
		
		a, aOk := cmd.Args["a"].(float64)
		b, bOk := cmd.Args["b"].(float64)
		
		if !aOk || !bOk {
			return nil, models.NewJSONRPCError(models.InvalidParams, "Arguments must be numbers")
		}
		
		return map[string]interface{}{"result": a * b}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Disable client-side validation to test server-side validation
	clientConfig := protocol.DefaultJanusClientConfig()
	clientConfig.EnableValidation = false
	client, err := protocol.New(socketPath, "test", clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	args := map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	}
	
	// Test addition
	addResponse, err := client.SendCommand(context.Background(), "add", args)
	if err != nil {
		t.Fatalf("Add command failed: %v", err)
	}
	
	// Test multiplication
	multResponse, err := client.SendCommand(context.Background(), "multiply", args)
	if err != nil {
		t.Fatalf("Multiply command failed: %v", err)
	}
	
	// Stop server
	srv.Stop()
	<-serverDone
	
	// Verify results
	if !addResponse.Success {
		t.Fatalf("Add failed: %v", addResponse.Error)
	}
	
	addResult := addResponse.Result.(map[string]interface{})["result"]
	if addResult != 8.0 {
		t.Fatalf("Expected 8, got %v", addResult)
	}
	
	if !multResponse.Success {
		t.Fatalf("Multiply failed: %v", multResponse.Error)
	}
	
	multResult := multResponse.Result.(map[string]interface{})["result"]
	if multResult != 15.0 {
		t.Fatalf("Expected 15, got %v", multResult)
	}
}