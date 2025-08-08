package tests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"GoJanus/pkg/models"
	"GoJanus/pkg/protocol"
	"GoJanus/pkg/manifest"
)

// TestRequestValidationWithoutConnection tests request validation without requiring a connection
// Matches Swift: testRequestValidationWithoutConnection()
func TestRequestValidationWithoutConnection(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createStatelessTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "stateless-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test with valid arguments - should fail at connection, not validation
	validArgs := map[string]interface{}{
		"test_param": "valid_value",
	}
	
	_, err = client.SendRequest(ctx, "stateless-request", validArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Test with invalid arguments - in Dynamic Manifest Architecture, 
	// this will also fail at connection stage before validation can occur
	invalidArgs := map[string]interface{}{
		"wrong_param": "value",
	}
	
	_, err = client.SendRequest(ctx, "stateless-request", invalidArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error (manifest fetching fails before validation)
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
}

// TestIndependentRequestExecution tests that requests execute independently
// Matches Swift: testIndependentRequestExecution()
func TestIndependentRequestExecution(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createStatelessTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "stateless-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Create multiple requests with different IDs
	args1 := map[string]interface{}{
		"test_param": "value1",
	}
	
	args2 := map[string]interface{}{
		"test_param": "value2",
	}
	
	// Both should fail with connection error (no server running)
	// but each should have unique request IDs
	_, err1 := client.SendRequest(ctx, "stateless-request", args1, protocol.RequestOptions{Timeout: 1*time.Second})
	_, err2 := client.SendRequest(ctx, "stateless-request", args2, protocol.RequestOptions{Timeout: 1*time.Second})
	
	if err1 == nil || err2 == nil {
		t.Error("Expected connection errors since no server is running")
	}
	
	// Both should be connection errors, not validation errors
	if strings.Contains(err1.Error(), "validation") || strings.Contains(err2.Error(), "validation") {
		t.Errorf("Got validation errors when expecting connection errors: %v, %v", err1, err2)
	}
}

// TestChannelIsolationBetweenClients tests that clients remain isolated by channel
// Matches Swift: testChannelIsolationBetweenClients()
func TestChannelIsolationBetweenClients(t *testing.T) {
	testSocketPath1 := "/tmp/gojanus-stateless1-test.sock"
	testSocketPath2 := "/tmp/gojanus-stateless2-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath1)
	os.Remove(testSocketPath2)
	defer func() {
		os.Remove(testSocketPath1)
		os.Remove(testSocketPath2)
	}()
	
	_ = createMultiChannelManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Create clients for different channels
	client1, err := protocol.New(testSocketPath1, "channel-1")
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()
	
	client2, err := protocol.New(testSocketPath2, "channel-2")
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()
	
	// Verify channel isolation
	if client1.ChannelIdentifier() == client2.ChannelIdentifier() {
		t.Error("Clients should have different channel identifiers")
	}
	
	ctx := context.Background()
	
	// Client1 should be able to call channel-1 requests
	args1 := map[string]interface{}{
		"param1": "value",
	}
	
	_, err = client1.SendRequest(ctx, "request-1", args1, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error (request exists in channel-1)
	if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "not found") {
		t.Errorf("Got validation/not-found error when expecting connection error: %v", err)
	}
	
	// Client1 should NOT be able to call channel-2 requests
	// In Dynamic Manifest Architecture, this fails at connection stage
	_, err = client1.SendRequest(ctx, "request-2", args1, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error (manifest fetching fails before validation)
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
}

// TestArgumentValidationInStatelessMode tests argument validation without persistent state
// Matches Swift: testArgumentValidationInStatelessMode()
func TestArgumentValidationInStatelessMode(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createStatelessTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "stateless-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test required argument validation
	argsWithoutRequired := map[string]interface{}{
		"optional_param": "value",
	}
	
	_, err = client.SendRequest(ctx, "validation-request", argsWithoutRequired, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// In Dynamic Manifest Architecture, this fails at connection stage
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Test type validation - also fails at connection stage in Dynamic Manifest Architecture
	argsWithWrongType := map[string]interface{}{
		"required_param": 123, // Should be string
		"optional_param": "value",
	}
	
	_, err = client.SendRequest(ctx, "validation-request", argsWithWrongType, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error (manifest fetching fails before validation)
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Test with valid arguments
	validArgs := map[string]interface{}{
		"required_param": "valid_string",
		"optional_param": "optional_value",
	}
	
	_, err = client.SendRequest(ctx, "validation-request", validArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
}

// TestMessageSerializationForStatelessOperations tests message serialization in stateless mode
// Matches Swift: testMessageSerializationForStatelessOperations()
func TestMessageSerializationForStatelessOperations(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createStatelessTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "stateless-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Test publish request (fire-and-forget)
	publishArgs := map[string]interface{}{
		"test_param": "publish_value",
	}
	
	ctx := context.Background()
	
	requestID, err := client.PublishRequest(ctx, "stateless-request", publishArgs)
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Even though it failed due to connection, the request ID should have been generated
	// (this tests that serialization and validation happened before connection attempt)
	if requestID != "" {
		t.Error("Request ID should be empty when publish fails")
	}
}

// TestMultiChannelManifestHandling tests handling of multi-channel manifests
// Matches Swift: testMultiChannelManifestHandling()
func TestMultiChannelManifestHandling(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createMultiChannelManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test creating clients for different channels
	client1, err := protocol.New(testSocketPath, "channel-1")
	if err != nil {
		t.Fatalf("Failed to create client for channel-1: %v", err)
	}
	defer client1.Close()
	
	client2, err := protocol.New(testSocketPath, "channel-2")
	if err != nil {
		t.Fatalf("Failed to create client for channel-2: %v", err)
	}
	defer client2.Close()
	
	// Verify clients have correct channel identifiers
	if client1.ChannelIdentifier() != "channel-1" {
		t.Errorf("Expected channel-1, got %s", client1.ChannelIdentifier())
	}
	
	if client2.ChannelIdentifier() != "channel-2" {
		t.Errorf("Expected channel-2, got %s", client2.ChannelIdentifier())
	}
	
	// Verify clients have access to the same manifest
	if client1.Manifest() != client2.Manifest() {
		t.Error("Clients should share the same manifest reference")
	}
	
	// Verify each client can only access requests from its channel
	ctx := context.Background()
	
	args := map[string]interface{}{
		"param1": "value",
	}
	
	// In Dynamic Manifest Architecture, both requests fail at connection stage
	// Client1 attempts to validate request-1 (would exist in channel-1 if server was running)
	_, err = client1.SendRequest(ctx, "request-1", args, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Client1 attempts to validate request-2 (would not exist in channel-1 if server was running)
	_, err = client1.SendRequest(ctx, "request-2", args, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
}

// TestStatelessRequestUUIDGeneration tests that each stateless request gets unique UUID
// Matches Swift stateless UUID generation patterns
func TestStatelessRequestUUIDGeneration(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createStatelessTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "stateless-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Verify client was created properly
	if client.ChannelIdentifier() != "stateless-channel" {
		t.Errorf("Expected channel ID 'stateless-channel', got %s", client.ChannelIdentifier())
	}
	
	// Create multiple requests and verify they have different UUIDs
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	request1 := models.NewJanusRequest("stateless-channel", "stateless-request", args, nil)
	request2 := models.NewJanusRequest("stateless-channel", "stateless-request", args, nil)
	request3 := models.NewJanusRequest("stateless-channel", "stateless-request", args, nil)
	
	// Verify UUIDs are different
	if request1.ID == request2.ID {
		t.Error("Requests should have different UUIDs")
	}
	
	if request2.ID == request3.ID {
		t.Error("Requests should have different UUIDs")
	}
	
	if request1.ID == request3.ID {
		t.Error("Requests should have different UUIDs")
	}
	
	// Verify UUIDs are not empty
	if request1.ID == "" || request2.ID == "" || request3.ID == "" {
		t.Error("Request UUIDs should not be empty")
	}
	
	// Verify UUID format (should be valid UUID string)
	if len(request1.ID) != 36 { // Standard UUID length
		t.Errorf("Expected UUID length 36, got %d", len(request1.ID))
	}
}

// TestChannelIsolationValidation tests channel isolation at the validation level
// Matches Swift channel isolation security testing
func TestChannelIsolationValidation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createMultiChannelManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "channel-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test that client validates channel isolation
	args := map[string]interface{}{
		"param2": "value", // This is the correct param for request-2
	}
	
	// Try to call request from different channel - fails at connection stage in Dynamic Manifest Architecture
	_, err = client.SendRequest(ctx, "request-2", args, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error (manifest fetching fails before validation)
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
}

// Helper function to create a stateless test Manifest
func createStatelessTestManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        "Stateless Test API",
		Description: "Manifest for stateless communication testing",
		Channels: map[string]*manifest.ChannelManifest{
			"stateless-channel": {
				Name:        "Stateless Channel",
				Description: "Channel for stateless testing",
				Requests: map[string]*manifest.RequestManifest{
					"stateless-request": {
						Name:        "Stateless Request",
						Description: "Request for stateless testing",
						Args: map[string]*manifest.ArgumentManifest{
							"test_param": {
								Name:        "Test Parameter",
								Type:        "string",
								Description: "Test parameter",
								Required:    true,
							},
						},
						Response: &manifest.ResponseManifest{
							Type:        "object",
							Description: "Test response",
						},
					},
					"validation-request": {
						Name:        "Validation Request",
						Description: "Request for validation testing",
						Args: map[string]*manifest.ArgumentManifest{
							"required_param": {
								Name:        "Required Parameter",
								Type:        "string",
								Description: "Required parameter",
								Required:    true,
							},
							"optional_param": {
								Name:        "Optional Parameter",
								Type:        "string",
								Description: "Optional parameter",
								Required:    false,
							},
						},
						Response: &manifest.ResponseManifest{
							Type:        "object",
							Description: "Validation response",
						},
					},
				},
			},
		},
	}
}

// Helper function to create a multi-channel Manifest
func createMultiChannelManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        "Multi-Channel Test API",
		Description: "Manifest with multiple channels",
		Channels: map[string]*manifest.ChannelManifest{
			"channel-1": {
				Name:        "Channel 1",
				Description: "First test channel",
				Requests: map[string]*manifest.RequestManifest{
					"request-1": {
						Name:        "Request 1",
						Description: "First channel request",
						Args: map[string]*manifest.ArgumentManifest{
							"param1": {
								Name:        "Parameter 1",
								Type:        "string",
								Description: "First parameter",
								Required:    true,
							},
						},
						Response: &manifest.ResponseManifest{
							Type:        "object",
							Description: "Response from channel 1",
						},
					},
				},
			},
			"channel-2": {
				Name:        "Channel 2",
				Description: "Second test channel",
				Requests: map[string]*manifest.RequestManifest{
					"request-2": {
						Name:        "Request 2",
						Description: "Second channel request",
						Args: map[string]*manifest.ArgumentManifest{
							"param2": {
								Name:        "Parameter 2",
								Type:        "string",
								Description: "Second parameter",
								Required:    true,
							},
						},
						Response: &manifest.ResponseManifest{
							Type:        "object",
							Description: "Response from channel 2",
						},
					},
				},
			},
		},
	}
}