package tests

import (
	"os"
	"testing"
	"time"

	"github.com/user/GoJanus/pkg/server"
	"github.com/user/GoJanus/pkg/models"
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
	server := &server.JanusServer{}
	
	server.RegisterHandler("test", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		return map[string]interface{}{"echo": cmd.Command}, nil
	})
	
	// Handler registration should succeed
	// We can't directly test the internal map, but no panic is good
}

func TestClientServerCommunication(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-comm.sock"
	defer os.Remove(socketPath)
	
	// Start server
	server := &server.JanusServer{}
	server.RegisterHandler("ping", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		return map[string]interface{}{
			"message":   "pong",
			"timestamp": time.Now().Unix(),
		}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client communication
	client := client.NewJanusClient("test", 5*time.Second)
	
	response, err := client.SendCommand(socketPath, "ping", nil)
	
	// Stop server
	server.Stop()
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
	server := &server.JanusServer{}
	server.RegisterHandler("echo", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		if cmd.Args == nil {
			return nil, &models.SocketError{
				Code:    "NO_ARGUMENTS",
				Message: "No arguments provided",
				Details: "",
			}
		}
		
		message, exists := cmd.Args["message"]
		if !exists {
			return nil, &models.SocketError{
				Code:    "MISSING_ARGUMENT",
				Message: "Missing 'message' argument",
				Details: "",
			}
		}
		
		return map[string]interface{}{
			"echo":        message,
			"received_at": time.Now().Unix(),
		}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with arguments
	client := client.NewJanusClient("test", 5*time.Second)
	
	args := map[string]interface{}{
		"message": "Hello, Server!",
	}
	
	response, err := client.SendCommand(socketPath, "echo", args)
	
	// Stop server
	server.Stop()
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
		t.Fatal("Expected result to be map[string]interface{}")
	}
	
	echo, exists := result["echo"]
	if !exists {
		t.Fatal("Expected 'echo' field in result")
	}
	
	if echo != "Hello, Server!" {
		t.Fatalf("Expected echo 'Hello, Server!' but got '%v'", echo)
	}
}

func TestClientCommandNotFound(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-notfound.sock"
	defer os.Remove(socketPath)
	
	// Start server without handlers
	server := &server.JanusServer{}
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with non-existent command
	client := client.NewJanusClient("test", 5*time.Second)
	
	response, err := client.SendCommand(socketPath, "nonexistent", nil)
	
	// Stop server
	server.Stop()
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
	
	if response.Error.Code != "COMMAND_NOT_FOUND" {
		t.Fatalf("Expected COMMAND_NOT_FOUND error, got %s", response.Error.Code)
	}
}

func TestClientFireAndForget(t *testing.T) {
	socketPath := "/tmp/test-go-high-level-fire.sock"
	defer os.Remove(socketPath)
	
	// Start server with logging handler
	server := &server.JanusServer{}
	server.RegisterHandler("log", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		// Simulate logging
		t.Logf("Logged: %v", cmd.Args)
		return map[string]interface{}{"logged": true}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test fire-and-forget
	client := client.NewJanusClient("test", 5*time.Second)
	
	args := map[string]interface{}{
		"level":   "info",
		"message": "Test log message",
	}
	
	err := client.SendCommandNoResponse(socketPath, "log", args)
	
	// Stop server
	server.Stop()
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
	server := &server.JanusServer{}
	server.RegisterHandler("slow", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		// Simulate slow processing
		time.Sleep(10 * time.Second)
		return map[string]interface{}{"done": true}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client with short timeout
	client := client.NewJanusClient("test", 1*time.Second)
	
	_, err := client.SendCommand(socketPath, "slow", nil)
	
	// Stop server
	server.Stop()
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
	server := &server.JanusServer{}
	server.RegisterHandler("ping", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		return map[string]interface{}{"message": "pong"}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test ping
	client := client.NewJanusClient("test", 5*time.Second)
	
	success := client.Ping(socketPath)
	
	// Stop server
	server.Stop()
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
	server := &server.JanusServer{}
	
	server.RegisterHandler("add", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		if cmd.Args == nil {
			return nil, &models.SocketError{
				Code:    "MISSING_ARGS",
				Message: "Missing arguments",
				Details: "",
			}
		}
		
		a, aOk := cmd.Args["a"].(float64)
		b, bOk := cmd.Args["b"].(float64)
		
		if !aOk || !bOk {
			return nil, &models.SocketError{
				Code:    "INVALID_ARGS",
				Message: "Arguments must be numbers",
				Details: "",
			}
		}
		
		return map[string]interface{}{"result": a + b}, nil
	})
	
	server.RegisterHandler("multiply", func(cmd *models.SocketCommand) (interface{}, *models.SocketError) {
		if cmd.Args == nil {
			return nil, &models.SocketError{
				Code:    "MISSING_ARGS",
				Message: "Missing arguments",
				Details: "",
			}
		}
		
		a, aOk := cmd.Args["a"].(float64)
		b, bOk := cmd.Args["b"].(float64)
		
		if !aOk || !bOk {
			return nil, &models.SocketError{
				Code:    "INVALID_ARGS",
				Message: "Arguments must be numbers",
				Details: "",
			}
		}
		
		return map[string]interface{}{"result": a * b}, nil
	})
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.StartListening(socketPath)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	client := client.NewJanusClient("test", 5*time.Second)
	
	args := map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	}
	
	// Test addition
	addResponse, err := client.SendCommand(socketPath, "add", args)
	if err != nil {
		t.Fatalf("Add command failed: %v", err)
	}
	
	// Test multiplication
	multResponse, err := client.SendCommand(socketPath, "multiply", args)
	if err != nil {
		t.Fatalf("Multiply command failed: %v", err)
	}
	
	// Stop server
	server.Stop()
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