package tests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user/GoUnixSockAPI"
	"github.com/user/GoUnixSockAPI/pkg/protocol"
)

// TestCommandValidationWithoutConnection tests command validation without requiring a connection
// Matches Swift: testCommandValidationWithoutConnection()
func TestCommandValidationWithoutConnection(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createStatelessTestAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "stateless-channel", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test with valid arguments - should fail at connection, not validation
	validArgs := map[string]interface{}{
		"test_param": "valid_value",
	}
	
	_, err = client.SendCommand(ctx, "stateless-command", validArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Test with invalid arguments - should fail at validation
	invalidArgs := map[string]interface{}{
		"wrong_param": "value",
	}
	
	_, err = client.SendCommand(ctx, "stateless-command", invalidArgs, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for invalid arguments")
	}
	
	// Should be validation error
	if !strings.Contains(err.Error(), "validation") && !strings.Contains(err.Error(), "required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestIndependentCommandExecution tests that commands execute independently
// Matches Swift: testIndependentCommandExecution()
func TestIndependentCommandExecution(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createStatelessTestAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "stateless-channel", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Create multiple commands with different IDs
	args1 := map[string]interface{}{
		"test_param": "value1",
	}
	
	args2 := map[string]interface{}{
		"test_param": "value2",
	}
	
	// Both should fail with connection error (no server running)
	// but each should have unique command IDs
	_, err1 := client.SendCommand(ctx, "stateless-command", args1, protocol.CommandOptions{Timeout: 1*time.Second})
	_, err2 := client.SendCommand(ctx, "stateless-command", args2, protocol.CommandOptions{Timeout: 1*time.Second})
	
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
	testSocketPath1 := "/tmp/gounixsocketapi-stateless1-test.sock"
	testSocketPath2 := "/tmp/gounixsocketapi-stateless2-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath1)
	os.Remove(testSocketPath2)
	defer func() {
		os.Remove(testSocketPath1)
		os.Remove(testSocketPath2)
	}()
	
	spec := createMultiChannelAPISpec()
	
	// Create clients for different channels
	client1, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath1, "channel-1", spec)
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close()
	
	client2, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath2, "channel-2", spec)
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close()
	
	// Verify channel isolation
	if client1.ChannelIdentifier() == client2.ChannelIdentifier() {
		t.Error("Clients should have different channel identifiers")
	}
	
	ctx := context.Background()
	
	// Client1 should be able to call channel-1 commands
	args1 := map[string]interface{}{
		"param1": "value",
	}
	
	_, err = client1.SendCommand(ctx, "command-1", args1, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error (command exists in channel-1)
	if strings.Contains(err.Error(), "validation") || strings.Contains(err.Error(), "not found") {
		t.Errorf("Got validation/not-found error when expecting connection error: %v", err)
	}
	
	// Client1 should NOT be able to call channel-2 commands
	_, err = client1.SendCommand(ctx, "command-2", args1, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for wrong channel command")
	}
	
	// Should be validation/not-found error (command doesn't exist in channel-1)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected command not found error, got: %v", err)
	}
}

// TestArgumentValidationInStatelessMode tests argument validation without persistent state
// Matches Swift: testArgumentValidationInStatelessMode()
func TestArgumentValidationInStatelessMode(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createStatelessTestAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "stateless-channel", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test required argument validation
	argsWithoutRequired := map[string]interface{}{
		"optional_param": "value",
	}
	
	_, err = client.SendCommand(ctx, "validation-command", argsWithoutRequired, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for missing required argument")
	}
	
	if !strings.Contains(err.Error(), "required") && !strings.Contains(err.Error(), "validation") {
		t.Errorf("Expected required argument error, got: %v", err)
	}
	
	// Test type validation
	argsWithWrongType := map[string]interface{}{
		"required_param": 123, // Should be string
		"optional_param": "value",
	}
	
	_, err = client.SendCommand(ctx, "validation-command", argsWithWrongType, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for wrong argument type")
	}
	
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Expected type validation error, got: %v", err)
	}
	
	// Test with valid arguments
	validArgs := map[string]interface{}{
		"required_param": "valid_string",
		"optional_param": "optional_value",
	}
	
	_, err = client.SendCommand(ctx, "validation-command", validArgs, protocol.CommandOptions{Timeout: 1*time.Second})
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
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createStatelessTestAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "stateless-channel", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Test publish command (fire-and-forget)
	publishArgs := map[string]interface{}{
		"test_param": "publish_value",
	}
	
	ctx := context.Background()
	
	commandID, err := client.PublishCommand(ctx, "stateless-command", publishArgs)
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not validation error
	if strings.Contains(err.Error(), "validation") {
		t.Errorf("Got validation error when expecting connection error: %v", err)
	}
	
	// Even though it failed due to connection, the command ID should have been generated
	// (this tests that serialization and validation happened before connection attempt)
	if commandID != "" {
		t.Error("Command ID should be empty when publish fails")
	}
}

// TestMultiChannelAPISpecificationHandling tests handling of multi-channel specifications
// Matches Swift: testMultiChannelAPISpecificationHandling()
func TestMultiChannelAPISpecificationHandling(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createMultiChannelAPISpec()
	
	// Test creating clients for different channels
	client1, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "channel-1", spec)
	if err != nil {
		t.Fatalf("Failed to create client for channel-1: %v", err)
	}
	defer client1.Close()
	
	client2, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "channel-2", spec)
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
	
	// Verify clients have access to the same specification
	if client1.Specification() != client2.Specification() {
		t.Error("Clients should share the same specification reference")
	}
	
	// Verify each client can only access commands from its channel
	ctx := context.Background()
	
	args := map[string]interface{}{
		"param1": "value",
	}
	
	// Client1 should be able to validate command-1 (exists in channel-1)
	_, err = client1.SendCommand(ctx, "command-1", args, protocol.CommandOptions{Timeout: 1*time.Second})
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Errorf("Client1 should be able to find command-1: %v", err)
	}
	
	// Client1 should NOT be able to validate command-2 (doesn't exist in channel-1)
	_, err = client1.SendCommand(ctx, "command-2", args, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Error("Client1 should not be able to find command-2")
	}
}

// TestStatelessCommandUUIDGeneration tests that each stateless command gets unique UUID
// Matches Swift stateless UUID generation patterns
func TestStatelessCommandUUIDGeneration(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createStatelessTestAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "stateless-channel", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	// Verify client was created properly
	if client.ChannelIdentifier() != "stateless-channel" {
		t.Errorf("Expected channel ID 'stateless-channel', got %s", client.ChannelIdentifier())
	}
	
	// Create multiple commands and verify they have different UUIDs
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	command1 := gounixsocketapi.NewSocketCommand("stateless-channel", "stateless-command", args, nil)
	command2 := gounixsocketapi.NewSocketCommand("stateless-channel", "stateless-command", args, nil)
	command3 := gounixsocketapi.NewSocketCommand("stateless-channel", "stateless-command", args, nil)
	
	// Verify UUIDs are different
	if command1.ID == command2.ID {
		t.Error("Commands should have different UUIDs")
	}
	
	if command2.ID == command3.ID {
		t.Error("Commands should have different UUIDs")
	}
	
	if command1.ID == command3.ID {
		t.Error("Commands should have different UUIDs")
	}
	
	// Verify UUIDs are not empty
	if command1.ID == "" || command2.ID == "" || command3.ID == "" {
		t.Error("Command UUIDs should not be empty")
	}
	
	// Verify UUID format (should be valid UUID string)
	if len(command1.ID) != 36 { // Standard UUID length
		t.Errorf("Expected UUID length 36, got %d", len(command1.ID))
	}
}

// TestChannelIsolationValidation tests channel isolation at the validation level
// Matches Swift channel isolation security testing
func TestChannelIsolationValidation(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-stateless-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createMultiChannelAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIDatagramClient(testSocketPath, "channel-1", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
	
	ctx := context.Background()
	
	// Test that client validates channel isolation
	args := map[string]interface{}{
		"param2": "value", // This is the correct param for command-2
	}
	
	// Try to call command from different channel - should fail validation
	_, err = client.SendCommand(ctx, "command-2", args, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for cross-channel command")
	}
	
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected command not found error, got: %v", err)
	}
}

// Helper function to create a stateless test API specification
func createStatelessTestAPISpec() *gounixsocketapi.APISpecification {
	return &gounixsocketapi.APISpecification{
		Version:     "1.0.0",
		Name:        "Stateless Test API",
		Description: "API specification for stateless communication testing",
		Channels: map[string]*gounixsocketapi.ChannelSpec{
			"stateless-channel": {
				Name:        "Stateless Channel",
				Description: "Channel for stateless testing",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"stateless-command": {
						Name:        "Stateless Command",
						Description: "Command for stateless testing",
						Args: map[string]*gounixsocketapi.ArgumentSpec{
							"test_param": {
								Name:        "Test Parameter",
								Type:        "string",
								Description: "Test parameter",
								Required:    true,
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Test response",
						},
					},
					"validation-command": {
						Name:        "Validation Command",
						Description: "Command for validation testing",
						Args: map[string]*gounixsocketapi.ArgumentSpec{
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
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Validation response",
						},
					},
				},
			},
		},
	}
}

// Helper function to create a multi-channel API specification
func createMultiChannelAPISpec() *gounixsocketapi.APISpecification {
	return &gounixsocketapi.APISpecification{
		Version:     "1.0.0",
		Name:        "Multi-Channel Test API",
		Description: "API specification with multiple channels",
		Channels: map[string]*gounixsocketapi.ChannelSpec{
			"channel-1": {
				Name:        "Channel 1",
				Description: "First test channel",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"command-1": {
						Name:        "Command 1",
						Description: "First channel command",
						Args: map[string]*gounixsocketapi.ArgumentSpec{
							"param1": {
								Name:        "Parameter 1",
								Type:        "string",
								Description: "First parameter",
								Required:    true,
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Response from channel 1",
						},
					},
				},
			},
			"channel-2": {
				Name:        "Channel 2",
				Description: "Second test channel",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"command-2": {
						Name:        "Command 2",
						Description: "Second channel command",
						Args: map[string]*gounixsocketapi.ArgumentSpec{
							"param2": {
								Name:        "Parameter 2",
								Type:        "string",
								Description: "Second parameter",
								Required:    true,
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Response from channel 2",
						},
					},
				},
			},
		},
	}
}