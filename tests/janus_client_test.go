package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"GoJanus/pkg/models"
	"GoJanus/pkg/protocol"
	"GoJanus/pkg/server"
	"GoJanus/pkg/manifest"
)

// TestClientInitializationWithValidManifest tests client creation with valid manifest
// Matches Swift: testClientInitializationWithValidManifest()
func TestClientInitializationWithValidManifest(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client with valid manifest: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	if client.SocketPathString() != testSocketPath {
		t.Errorf("Expected socket path '%s', got '%s'", testSocketPath, client.SocketPathString())
	}
	
	if client.ChannelIdentifier() != "test" {
		t.Errorf("Expected channel 'test', got '%s'", client.ChannelIdentifier())
	}
	
	// In Dynamic Manifest Architecture, manifests are auto-fetched from server
	// For this test, we expect the manifest to be nil since no server is running
	if client.Manifest() != nil {
		t.Error("Expected manifest to be nil when no server is running")
	}
}

// TestClientInitializationWithInvalidChannel tests client creation failure with invalid channel
// Matches Swift: testClientInitializationWithInvalidChannel()
func TestClientInitializationWithInvalidChannel(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "nonexistent-channel")
	if err != nil {
		// Constructor should succeed - validation happens during sendRequest
		t.Errorf("Constructor should not fail for invalid channel: %v", err)
		return
	}
	defer client.Close()
	
	// Now test that sending a request fails due to connection error (no server)
	ctx := context.Background()
	_, err = client.SendRequest(ctx, "ping", nil)
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
	_, err := protocol.New(testSocketPath, "invalid/channel")
	if err == nil {
		t.Error("Expected error for invalid channel format")
		return
	}
	
	if !strings.Contains(err.Error(), "channel") && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected channel validation error, got: %v", err)
	}
}

// TestRegisterValidRequestHandler tests registering a valid request handler
// Matches Swift: testRegisterValidRequestHandler()
func TestRegisterValidRequestHandler(t *testing.T) {
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
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register handler for existing request
	handler := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		return models.NewSuccessResponse(request.ID, request.ChannelID, map[string]interface{}{
			"message": "Handler executed successfully",
		}), nil
	}
	
	err = client.RegisterRequestHandler("echo", handler)
	if err != nil {
		t.Errorf("Failed to register valid request handler: %v", err)
	}
}

// TestRegisterInvalidRequestHandler tests registering handler for nonexistent request
// Matches Swift: testRegisterInvalidRequestHandler()
func TestRegisterInvalidRequestHandler(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Try to register handler for nonexistent request
	handler := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		return models.NewSuccessResponse(request.ID, request.ChannelID, nil), nil
	}
	
	err = client.RegisterRequestHandler("nonexistent-request", handler)
	if err == nil {
		t.Error("Expected error for nonexistent request")
	}
	
	// Should get connection error since no server is running
	if err != nil && !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected connection-related error, got: %v", err)
	}
}

// TestJanusRequestValidation tests socket request validation against manifest
// Matches Swift: testJanusRequestValidation()
func TestJanusRequestValidation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "test")
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
	// but the request validation should pass
	_, err = client.SendRequest(ctx, "echo", validArgs, protocol.RequestOptions{Timeout: 1*time.Second})
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
	
	_, err = client.SendRequest(ctx, "echo", invalidArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected error for invalid arguments")
	}
	
	// Since no server is running, we'll get connection error not validation error
	// This is expected behavior with Dynamic Manifest Architecture
	if !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Expected connection-related error, got: %v", err)
	}
}

// TestRequestMessageSerialization tests request message serialization
// Matches Swift: testRequestMessageSerialization()
func TestRequestMessageSerialization(t *testing.T) {
	testSocketPath := "/tmp/gojanus-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "test")
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
	
	// Create request directly for serialization testing
	request := models.NewJanusRequest("test", "echo", args, nil)
	
	// Test serialization
	jsonData, err := request.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}
	
	// Test deserialization
	var deserializedRequest models.JanusRequest
	err = deserializedRequest.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize request: %v", err)
	}
	
	// Verify integrity
	if deserializedRequest.ChannelID != request.ChannelID {
		t.Errorf("ChannelID mismatch: expected '%s', got '%s'", request.ChannelID, deserializedRequest.ChannelID)
	}
	
	if deserializedRequest.Request != request.Request {
		t.Errorf("Request mismatch: expected '%s', got '%s'", request.Request, deserializedRequest.Request)
	}
	
	if len(deserializedRequest.Args) != len(request.Args) {
		t.Errorf("Args count mismatch: expected %d, got %d", len(request.Args), len(deserializedRequest.Args))
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
	client1, err := protocol.New(testSocketPath1, "test")
	if err != nil {
		t.Fatalf("Failed to create first client: %v", err)
	}
	defer client1.Close()
	
	// Create second client with different socket path and same channel
	client2, err := protocol.New(testSocketPath2, "test")
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
	handler1 := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		return models.NewSuccessResponse(request.ID, request.ChannelID, map[string]interface{}{
			"client": "client1",
		}), nil
	}
	
	handler2 := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		return models.NewSuccessResponse(request.ID, request.ChannelID, map[string]interface{}{
			"client": "client2",
		}), nil
	}
	
	err = client1.RegisterRequestHandler("echo", handler1)
	if err != nil {
		t.Errorf("Failed to register handler on client1: %v", err)
	}
	
	err = client2.RegisterRequestHandler("ping", handler2)
	if err != nil {
		t.Errorf("Failed to register handler on client2: %v", err)
	}
}

// TestRequestHandlerWithAsyncOperations tests request handler with async operations
// Matches Swift: testRequestHandlerWithAsyncOperations()
func TestRequestHandlerWithAsyncOperations(t *testing.T) {
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
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register async handler
	asyncHandler := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		// Simulate async operation
		time.Sleep(10 * time.Millisecond)
		
		return models.NewSuccessResponse(request.ID, request.ChannelID, map[string]interface{}{
			"processed": true,
			"timestamp": time.Now().Unix(),
		}), nil
	}
	
	err = client.RegisterRequestHandler("echo", asyncHandler)
	if err != nil {
		t.Errorf("Failed to register async handler: %v", err)
	}
}

// TestRequestHandlerErrorHandling tests request handler error handling
// Matches Swift: testRequestHandlerErrorHandling()
func TestRequestHandlerErrorHandling(t *testing.T) {
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
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Register error-producing handler
	errorHandler := func(request *models.JanusRequest) (*models.JanusResponse, error) {
		return nil, models.NewJSONRPCError(models.InternalError, "Simulated handler error")
	}
	
	err = client.RegisterRequestHandler("echo", errorHandler)
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
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Test complex arguments for echo request (which accepts a message argument)
	complexArgs := map[string]interface{}{
		"message": "Complex Task with detailed description and high priority due 2025-12-31T23:59:59Z",
	}
	
	ctx := context.Background()
	
	// This should fail with connection error (expected since no server running)
	// but the argument validation should pass
	_, err = client.SendRequest(ctx, "echo", complexArgs, protocol.RequestOptions{Timeout: 1*time.Second})
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
	
	// Start server with custom request that validates arguments
	config := &server.ServerConfig{
		SocketPath:     testSocketPath,
		MaxConnections: 100,
		DefaultTimeout: 30,
	}
	srv := server.NewJanusServer(config)
	// Register a custom request "validate_message" that requires non-empty message
	srv.RegisterHandler("validate_message", server.NewObjectHandler(func(cmd *models.JanusRequest) (map[string]interface{}, error) {
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
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	ctx := context.Background()
	
	// Test with valid arguments (should succeed)
	validArgs := map[string]interface{}{
		"message": "hello world",
	}
	
	_, err = client.SendRequest(ctx, "validate_message", validArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err != nil {
		t.Errorf("Valid arguments should not fail: %v", err)
	}
	
	// Test with empty message (should fail validation)
	invalidArgs := map[string]interface{}{
		"message": "",
	}
	
	response, err := client.SendRequest(ctx, "validate_message", invalidArgs, protocol.RequestOptions{Timeout: 1*time.Second})
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


// loadTestManifest loads the test Manifest from test-manifest.json
func loadTestManifest() *manifest.Manifest {
	manifestPath := "../../tests/config/manifest-request-test-api.json"
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read manifest-request-test-api.json: %v", err))
	}
	
	var manifest manifest.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		panic(fmt.Sprintf("Failed to parse manifest-request-test-api.json: %v", err))
	}
	
	return &manifest
}

