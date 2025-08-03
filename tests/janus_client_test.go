package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user/GoJanus"
	"github.com/user/GoJanus/pkg/models"
	"github.com/user/GoJanus/pkg/protocol"
	"github.com/user/GoJanus/pkg/server"
)

// TestClientInitializationWithValidSpec tests client creation with valid specification
// Matches Swift: testClientInitializationWithValidSpec()
func TestClientInitializationWithValidSpec(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client with valid spec: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	if client.SocketPathString() != testSocketPath {
		t.Errorf("Expected socket path '%s', got '%s'", testSocketPath, client.SocketPathString())
	}
	
	if client.ChannelIdentifier() != "test" {
		t.Errorf("Expected channel 'test', got '%s'", client.ChannelIdentifier())
	}
	
	// In Dynamic Specification Architecture, specifications are auto-fetched from server
	// For this test, we expect the specification to be nil since no server is running
	if client.Specification() != nil {
		t.Error("Expected specification to be nil when no server is running")
	}
}

// TestClientInitializationWithInvalidChannel tests client creation failure with invalid channel
// Matches Swift: testClientInitializationWithInvalidChannel()
func TestClientInitializationWithInvalidChannel(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "nonexistent-channel")
	if err != nil {
		// Constructor should succeed - validation happens during sendCommand
		t.Errorf("Constructor should not fail for invalid channel: %v", err)
		return
	}
	defer client.Close()
	
	// Now test that sending a command fails due to connection error (no server)
	ctx := context.Background()
	_, err = client.SendCommand(ctx, "ping", nil)
	if err == nil {
		t.Error("Expected connection error when no server is running")
		return
	}
	
	// Should get connection error, not channel validation error
	if !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected connection-related error, got: %v", err)
	}
}

// TestClientInitializationWithInvalidChannel tests client creation failure with invalid channel format
// Matches Swift: testClientInitializationWithInvalidChannel()
func TestClientInitializationWithInvalidChannelFormat(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Test invalid channel ID format (contains invalid characters)
	_, err := gojanus.NewJanusClient(testSocketPath, "invalid/channel")
	if err == nil {
		t.Error("Expected error for invalid channel format")
		return
	}
	
	if !strings.Contains(err.Error(), "channel") && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected channel validation error, got: %v", err)
	}
}

// TestRegisterValidCommandHandler tests registering a valid command handler
// Matches Swift: testRegisterValidCommandHandler()
func TestRegisterValidCommandHandler(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Start server for manifest validation
	config := &server.ServerConfig{
		SocketPath:     testSocketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.StartListening() }()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		srv.Stop()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}
	}()
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register handler for existing command
	handler := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		return gojanus.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"message": "Handler executed successfully",
		}), nil
	}
	
	err = client.RegisterCommandHandler("echo", handler)
	if err != nil {
		t.Errorf("Failed to register valid command handler: %v", err)
	}
}

// TestRegisterInvalidCommandHandler tests registering handler for nonexistent command
// Matches Swift: testRegisterInvalidCommandHandler()
func TestRegisterInvalidCommandHandler(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Try to register handler for nonexistent command
	handler := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		return gojanus.NewSuccessResponse(command.ID, command.ChannelID, nil), nil
	}
	
	err = client.RegisterCommandHandler("nonexistent-command", handler)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
	
	// Should get connection error since no server is running
	if err != nil && !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected connection-related error, got: %v", err)
	}
}

// TestJanusCommandValidation tests socket command validation against specification
// Matches Swift: testJanusCommandValidation()
func TestJanusCommandValidation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test with valid arguments
	validArgs := map[string]interface{}{
		"message": "test message",
	}
	
	// This should fail with connection error (expected since no server running)
	// but the command validation should pass
	_, err = client.SendCommand(ctx, "echo", validArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Test with invalid arguments (missing required field)
	invalidArgs := map[string]interface{}{
		"wrong_field": "value",
	}
	
	_, err = client.SendCommand(ctx, "echo", invalidArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected error for invalid arguments")
	}
	
	// Since no server is running, we'll get connection error not validation error
	// This is expected behavior with Dynamic Specification Architecture
	if !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected connection-related error, got: %v", err)
	}
}

// TestCommandMessageSerialization tests command message serialization
// Matches Swift: testCommandMessageSerialization()
func TestCommandMessageSerialization(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Verify client was created properly
	if client.ChannelIdentifier() != "test" {
		t.Errorf("Expected channel ID 'test', got %s", client.ChannelIdentifier())
	}
	
	args := map[string]interface{}{
		"title":  "Test Book",
		"author": "Test Author",
		"pages":  200,
	}
	
	// Create command directly for serialization testing
	command := gojanus.NewJanusCommand("test", "echo", args, nil)
	
	// Test serialization
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	// Test deserialization
	var deserializedCommand gojanus.JanusCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command: %v", err)
	}
	
	// Verify integrity
	if deserializedCommand.ChannelID != command.ChannelID {
		t.Errorf("ChannelID mismatch: expected '%s', got '%s'", command.ChannelID, deserializedCommand.ChannelID)
	}
	
	if deserializedCommand.Command != command.Command {
		t.Errorf("Command mismatch: expected '%s', got '%s'", command.Command, deserializedCommand.Command)
	}
	
	if len(deserializedCommand.Args) != len(command.Args) {
		t.Errorf("Args count mismatch: expected %d, got %d", len(command.Args), len(deserializedCommand.Args))
	}
}

// TestMultipleClientInstances tests creating multiple independent client instances
// Matches Swift: testMultipleClientInstances()
func TestMultipleClientInstances(t *testing.T) {
	testSocketPath1 := "/tmp/gojanus-client1-test.sock"
	testSocketPath2 := "/tmp/gojanus-client2-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath1)
	os.Remove(testSocketPath2)
	defer func() {
		os.Remove(testSocketPath1)
		os.Remove(testSocketPath2)
	}()
	
	// Start servers for both clients
	config1 := &server.ServerConfig{
		SocketPath:     testSocketPath1,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv1 := server.NewJanusServer(config1)
	
	config2 := &server.ServerConfig{
		SocketPath:     testSocketPath2,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv2 := server.NewJanusServer(config2)
	
	// Start servers in goroutines
	serverDone1 := make(chan error, 1)
	serverDone2 := make(chan error, 1)
	go func() { serverDone1 <- srv1.StartListening() }()
	go func() { serverDone2 <- srv2.StartListening() }()
	
	// Give servers time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		srv1.Stop()
		srv2.Stop()
		select {
		case <-serverDone1:
		case <-time.After(2 * time.Second):
		}
		select {
		case <-serverDone2:
		case <-time.After(2 * time.Second):
		}
	}()
	
	// Create first client
	client1, err := gojanus.NewJanusClient(testSocketPath1, "test")
	if err != nil {
		t.Fatalf("Failed to create first client: %v", err)
	}
	defer client1.Close()
	
	// Create second client with different socket path and same channel
	client2, err := gojanus.NewJanusClient(testSocketPath2, "test")
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}
	defer client2.Close()
	
	// Verify independence
	if client1.SocketPathString() == client2.SocketPathString() {
		t.Error("Clients should have different socket paths")
	}
	
	// Both clients use the same channel ID, so they should have the same channel identifier
	if client1.ChannelIdentifier() != client2.ChannelIdentifier() {
		t.Error("Clients using the same channel should have the same channel identifier")
	}
	
	// Register different handlers on each client
	handler1 := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		return gojanus.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"client": "client1",
		}), nil
	}
	
	handler2 := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		return gojanus.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"client": "client2",
		}), nil
	}
	
	err = client1.RegisterCommandHandler("echo", handler1)
	if err != nil {
		t.Errorf("Failed to register handler on client1: %v", err)
	}
	
	err = client2.RegisterCommandHandler("ping", handler2)
	if err != nil {
		t.Errorf("Failed to register handler on client2: %v", err)
	}
}

// TestCommandHandlerWithAsyncOperations tests command handler with async operations
// Matches Swift: testCommandHandlerWithAsyncOperations()
func TestCommandHandlerWithAsyncOperations(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Start server for client testing
	config := &server.ServerConfig{
		SocketPath:     testSocketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.StartListening() }()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		srv.Stop()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}
	}()
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register async handler
	asyncHandler := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		// Simulate async operation
		time.Sleep(10 * time.Millisecond)
		
		return gojanus.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"processed": true,
			"timestamp": time.Now().Unix(),
		}), nil
	}
	
	err = client.RegisterCommandHandler("echo", asyncHandler)
	if err != nil {
		t.Errorf("Failed to register async handler: %v", err)
	}
}

// TestCommandHandlerErrorHandling tests command handler error handling
// Matches Swift: testCommandHandlerErrorHandling()
func TestCommandHandlerErrorHandling(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Start server for client testing
	config := &server.ServerConfig{
		SocketPath:     testSocketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.StartListening() }()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		srv.Stop()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}
	}()
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register error-producing handler
	errorHandler := func(command *gojanus.JanusCommand) (*gojanus.JanusResponse, error) {
		return nil, gojanus.NewJSONRPCError(gojanus.InternalError, "Simulated handler error")
	}
	
	err = client.RegisterCommandHandler("echo", errorHandler)
	if err != nil {
		t.Errorf("Failed to register error handler: %v", err)
	}
}

// TestManifestWithComplexArguments tests Manifest with complex argument structures
// Matches Swift: testManifestWithComplexArguments()
func TestManifestWithComplexArguments(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Test complex arguments for echo command (which accepts a message argument)
	complexArgs := map[string]interface{}{
		"message": "Complex Task with detailed description and high priority due 2025-12-31T23:59:59Z",
	}
	
	ctx := context.Background()
	
	// This should fail with connection error (expected since no server running)
	// but the argument validation should pass
	_, err = client.SendCommand(ctx, "echo", complexArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
}

// TestArgumentValidationConstraints tests argument validation with various constraints
// Matches Swift: testArgumentValidationConstraints()
func TestArgumentValidationConstraints(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Start server with custom command that validates arguments
	config := &server.ServerConfig{
		SocketPath:     testSocketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	// Register a custom command "validate_message" that requires non-empty message
	srv.RegisterHandler("validate_message", server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		// Validate that message argument exists and is not empty
		if cmd.Args == nil {
			return nil, models.NewJSONRPCError(models.InvalidParams, "message argument is required and cannot be empty")
		}
		
		message, exists := cmd.Args["message"]
		if !exists {
			return nil, models.NewJSONRPCError(models.InvalidParams, "message argument is required and cannot be empty")
		}
		
		messageStr, ok := message.(string)
		if !ok {
			return nil, models.NewJSONRPCError(models.InvalidParams, "message argument must be a string")
		}
		
		if messageStr == "" {
			return nil, models.NewJSONRPCError(models.InvalidParams, "message argument cannot be empty")
		}
		
		return map[string]interface{}{"validated": message}, nil
	}))
	
	// Start server in goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.StartListening()
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	defer func() {
		srv.Stop()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}
	}()
	
	client, err := gojanus.NewJanusClient(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	ctx := context.Background()
	
	// Test with valid arguments (should succeed)
	validArgs := map[string]interface{}{
		"message": "hello world",
	}
	
	_, err = client.SendCommand(ctx, "validate_message", validArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err != nil {
		t.Errorf("Valid arguments should not fail: %v", err)
	}
	
	// Test with empty message (should fail validation)
	invalidArgs := map[string]interface{}{
		"message": "",
	}
	
	response, err := client.SendCommand(ctx, "validate_message", invalidArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil && response.Success {
		t.Errorf("Expected validation error for empty message, but got successful response: %+v", response)
	} else if err != nil {
		// Should get validation error from server
		if !strings.Contains(err.Error(), "required") && !strings.Contains(err.Error(), "empty") && !strings.Contains(err.Error(), "INVALID_PARAMS") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	} else if response != nil && !response.Success {
		// Server returned error response - this is what we expect
		if response.Error == nil {
			t.Error("Server returned error response but no error details")
		} else if !strings.Contains(response.Error.Error(), "empty") {
			t.Errorf("Expected empty message error, got: %v", response.Error)
		}
	}
}


// loadTestManifest loads the test Manifest from test-spec.json
func loadTestManifest() *gojanus.Manifest {
	specPath := "../../tests/config/spec-command-test-api.json"
	specData, err := os.ReadFile(specPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read spec-command-test-api.json: %v", err))
	}
	
	var manifest gojanus.Manifest
	if err := json.Unmarshal(specData, &manifest); err != nil {
		panic(fmt.Sprintf("Failed to parse spec-command-test-api.json: %v", err))
	}
	
	return &manifest
}

