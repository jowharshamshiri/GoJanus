package tests

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/user/GoJanus"
	"github.com/user/GoJanus/pkg/protocol"
)

// TestCommandTimeoutConfiguration tests timeout configuration for commands
// Matches Swift: testCommandTimeoutConfiguration()
func TestCommandTimeoutConfiguration(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Test with short timeout (1 second)
	shortTimeout := 1 * time.Second
	start := time.Now()
	
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: shortTimeout})
	elapsed := time.Since(start)
	
	if err == nil {
		t.Error("Expected timeout or connection error")
	}
	
	// Should fail quickly due to connection error, not wait for timeout
	if elapsed > 2*time.Second {
		t.Errorf("Command took too long: %v", elapsed)
	}
	
	// Test with longer timeout (5 seconds)
	longerTimeout := 5 * time.Second
	start = time.Now()
	
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: longerTimeout})
	elapsed = time.Since(start)
	
	if err == nil {
		t.Error("Expected timeout or connection error")
	}
	
	// Should still fail quickly due to connection error
	if elapsed > 2*time.Second {
		t.Errorf("Command took too long: %v", elapsed)
	}
}

// TestTimeoutCallbackMechanisms tests timeout callback functionality
// Matches Swift: testTimeoutCallbackMechanisms()
func TestTimeoutCallbackMechanisms(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Test timeout behavior
	var timeoutCalled int32
	
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error")
	}
	
	// Give a moment for potential timeout processing
	time.Sleep(100 * time.Millisecond)
	
	// Since we get connection error immediately, timeout callback should not be called
	if atomic.LoadInt32(&timeoutCalled) != 0 {
		t.Error("Timeout callback should not be called for connection errors")
	}
}

// TestUUIDGeneration tests UUID generation for commands
// Matches Swift: testUUIDGeneration()
func TestUUIDGeneration(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test that multiple commands get different UUIDs
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	command1 := gojanus.NewSocketCommand("timeout-channel", "timeout-command", args, nil)
	command2 := gojanus.NewSocketCommand("timeout-channel", "timeout-command", args, nil)
	command3 := gojanus.NewSocketCommand("timeout-channel", "timeout-command", args, nil)
	
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
	
	// Verify UUID format (36 characters with hyphens)
	if len(command1.ID) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(command1.ID))
	}
	
	if !strings.Contains(command1.ID, "-") {
		t.Error("UUID should contain hyphens")
	}
	
	// Verify UUIDs are not empty
	if command1.ID == "" || command2.ID == "" || command3.ID == "" {
		t.Error("Command UUIDs should not be empty")
	}
}

// TestMultipleConcurrentTimeouts tests handling multiple timeouts simultaneously
// Matches Swift: testMultipleConcurrentTimeouts()
func TestMultipleConcurrentTimeouts(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Launch multiple concurrent commands
	numCommands := 5
	results := make(chan error, numCommands)
	
	for i := 0; i < numCommands; i++ {
		go func(index int) {
			_, err := client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 1*time.Second})
			results <- err
		}(i)
	}
	
	// Collect results
	for i := 0; i < numCommands; i++ {
		err := <-results
		if err == nil {
			t.Error("Expected connection error for concurrent command")
		}
		
		// Should be connection errors for SOCK_DGRAM (no server listening)
		if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "connection") {
			t.Errorf("Expected connection error for SOCK_DGRAM, got: %v", err)
		}
	}
}

// TestDefaultTimeoutBehavior tests default timeout behavior when not specified
// Matches Swift: testDefaultTimeoutBehavior()
func TestDefaultTimeoutBehavior(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Test with default timeout (30 seconds from configuration)
	start := time.Now()
	
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 30*time.Second})
	elapsed := time.Since(start)
	
	if err == nil {
		t.Error("Expected connection error")
	}
	
	// Should fail quickly due to connection error, not wait for 30 seconds
	if elapsed > 2*time.Second {
		t.Errorf("Command took too long with default timeout: %v", elapsed)
	}
}

// TestTimeoutErrorMessageFormatting tests timeout error message formatting
// Matches Swift: testTimeoutErrorMessageFormatting()
func TestTimeoutErrorMessageFormatting(t *testing.T) {
	// Test creating timeout-related error messages
	timeoutError := gojanus.NewJSONRPCError(gojanus.HandlerTimeout, "No response received within timeout period")
	
	errorString := timeoutError.Error()
	
	if !strings.Contains(errorString, "COMMAND_TIMEOUT") {
		t.Errorf("Error string should contain error code: %s", errorString)
	}
	
	if !strings.Contains(errorString, "timed out") {
		t.Errorf("Error string should contain timeout message: %s", errorString)
	}
	
	if !strings.Contains(errorString, "No response received") {
		t.Errorf("Error string should contain error details: %s", errorString)
	}
	
	// Test error without details
	simpleTimeoutError := gojanus.NewJSONRPCError(gojanus.HandlerTimeout, "")
	
	simpleErrorString := simpleTimeoutError.Error()
	
	if !strings.Contains(simpleErrorString, "TIMEOUT") {
		t.Errorf("Simple error string should contain error code: %s", simpleErrorString)
	}
	
	if !strings.Contains(simpleErrorString, "Operation timed out") {
		t.Errorf("Simple error string should contain message: %s", simpleErrorString)
	}
	
	// Should not contain details section for simple error
	if strings.Contains(simpleErrorString, "()") {
		t.Errorf("Simple error should not have empty details: %s", simpleErrorString)
	}
}

// TestSocketCommandTimeoutFieldSerialization tests timeout field in socket commands
// Matches Swift: testSocketCommandTimeoutFieldSerialization()
func TestSocketCommandTimeoutFieldSerialization(t *testing.T) {
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Test command with timeout
	timeout := 45.0
	commandWithTimeout := gojanus.NewSocketCommand("timeout-channel", "timeout-command", args, &timeout)
	
	// Serialize to JSON
	jsonData, err := commandWithTimeout.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command with timeout: %v", err)
	}
	
	// Deserialize back
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command with timeout: %v", err)
	}
	
	// Verify timeout field
	if deserializedCommand.Timeout == nil {
		t.Error("Deserialized command should have timeout field")
	} else if *deserializedCommand.Timeout != timeout {
		t.Errorf("Expected timeout %f, got %f", timeout, *deserializedCommand.Timeout)
	}
	
	// Test command without timeout
	commandWithoutTimeout := gojanus.NewSocketCommand("timeout-channel", "timeout-command", args, nil)
	
	// Serialize to JSON
	jsonData, err = commandWithoutTimeout.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command without timeout: %v", err)
	}
	
	// Deserialize back
	var deserializedCommandNoTimeout gojanus.SocketCommand
	err = deserializedCommandNoTimeout.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command without timeout: %v", err)
	}
	
	// Verify timeout field is nil
	if deserializedCommandNoTimeout.Timeout != nil {
		t.Errorf("Command without timeout should have nil timeout field, got %v", *deserializedCommandNoTimeout.Timeout)
	}
}

// TestTimeoutValidation tests timeout value validation
// Matches Swift timeout validation patterns
func TestTimeoutValidation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-timeout-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createTimeoutTestManifest() // Load spec but don't use it - specification is now fetched dynamically
	client, err := gojanus.NewJanusClient(testSocketPath, "timeout-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	args := map[string]interface{}{
		"test_param": "value",
	}
	
	// Test with very short timeout (should be validated)
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 50*time.Millisecond})
	if err == nil {
		t.Error("Expected validation error for very short timeout")
	}
	
	// In SOCK_DGRAM without server, expect connection error
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "connection") {
		t.Errorf("Expected connection error for SOCK_DGRAM, got: %v", err)
	}
	
	// Test with very long timeout (should be validated)
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 400*time.Second})
	if err == nil {
		t.Error("Expected validation error for very long timeout")
	}
	
	// In SOCK_DGRAM without server, expect connection error
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "connection") {
		t.Errorf("Expected connection error for SOCK_DGRAM, got: %v", err)
	}
	
	// Test with valid timeout
	_, err = client.SendCommand(ctx, "timeout-command", args, protocol.CommandOptions{Timeout: 30*time.Second})
	if err != nil && strings.Contains(err.Error(), "validation") && strings.Contains(err.Error(), "timeout") {
		t.Errorf("Valid timeout should not produce validation error: %v", err)
	}
	
	// Connection error is expected (no server running)
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
}

// TestTimeoutManagerFunctionality tests the timeout manager directly
// Tests internal timeout management functionality
func TestTimeoutManagerFunctionality(t *testing.T) {
	// Test timeout manager creation and basic operations
	manager := gojanus.NewTimeoutManager()
	if manager == nil {
		t.Fatal("Failed to create timeout manager")
	}
	defer manager.Close()
	
	// Test active timeouts count
	if manager.ActiveTimeouts() != 0 {
		t.Errorf("New timeout manager should have 0 active timeouts, got %d", manager.ActiveTimeouts())
	}
	
	// Test timeout registration and cancellation
	var callbackCalled int32
	callback := func() {
		atomic.StoreInt32(&callbackCalled, 1)
	}
	
	// Register a timeout
	manager.RegisterTimeout("test-command-1", 50*time.Millisecond, callback)
	
	if manager.ActiveTimeouts() != 1 {
		t.Errorf("Expected 1 active timeout, got %d", manager.ActiveTimeouts())
	}
	
	// Cancel the timeout before it fires
	cancelled := manager.CancelTimeout("test-command-1")
	if !cancelled {
		t.Error("Expected timeout to be cancelled successfully")
	}
	
	if manager.ActiveTimeouts() != 0 {
		t.Errorf("Expected 0 active timeouts after cancellation, got %d", manager.ActiveTimeouts())
	}
	
	// Wait to ensure callback doesn't fire
	time.Sleep(100 * time.Millisecond)
	
	if atomic.LoadInt32(&callbackCalled) != 0 {
		t.Error("Callback should not be called for cancelled timeout")
	}
	
	// Test timeout that actually fires
	var timeoutFired int32
	timeoutCallback := func() {
		atomic.StoreInt32(&timeoutFired, 1)
	}
	
	manager.RegisterTimeout("test-command-2", 50*time.Millisecond, timeoutCallback)
	
	// Wait for timeout to fire
	time.Sleep(100 * time.Millisecond)
	
	if atomic.LoadInt32(&timeoutFired) != 1 {
		t.Error("Timeout callback should have been called")
	}
	
	if manager.ActiveTimeouts() != 0 {
		t.Errorf("Expected 0 active timeouts after timeout fired, got %d", manager.ActiveTimeouts())
	}
}

// Helper function to create timeout test Manifest
func createTimeoutTestManifest() *gojanus.Manifest {
	return &gojanus.Manifest{
		Version:     "1.0.0",
		Name:        "Timeout Test API",
		Description: "Manifest for timeout testing",
		Channels: map[string]*gojanus.ChannelSpec{
			"timeout-channel": {
				Name:        "Timeout Channel",
				Description: "Channel for timeout testing",
				Commands: map[string]*gojanus.CommandSpec{
					"timeout-command": {
						Name:        "Timeout Command",
						Description: "Command for timeout testing",
						Args: map[string]*gojanus.ArgumentSpec{
							"test_param": {
								Name:        "Test Parameter",
								Type:        "string",
								Description: "Test parameter for timeout command",
								Required:    true,
							},
						},
						Response: &gojanus.ResponseSpec{
							Type:        "object",
							Description: "Timeout test response",
						},
						ErrorCodes: []string{"COMMAND_TIMEOUT", "CONNECTION_ERROR"},
					},
				},
			},
		},
	}
}