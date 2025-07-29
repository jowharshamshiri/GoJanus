package tests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user/GoUnixSockAPI"
)

// TestClientInitializationWithValidSpec tests client creation with valid specification
// Matches Swift: testClientInitializationWithValidSpec()
func TestClientInitializationWithValidSpec(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client with valid spec: %v", err)
	}
	defer client.Close()
	
	if client.SocketPathString() != testSocketPath {
		t.Errorf("Expected socket path '%s', got '%s'", testSocketPath, client.SocketPathString())
	}
	
	if client.ChannelIdentifier() != "library-management" {
		t.Errorf("Expected channel 'library-management', got '%s'", client.ChannelIdentifier())
	}
	
	if client.Specification() != spec {
		t.Error("Expected specification to match provided spec")
	}
}

// TestClientInitializationWithInvalidChannel tests client creation failure with invalid channel
// Matches Swift: testClientInitializationWithInvalidChannel()
func TestClientInitializationWithInvalidChannel(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	
	_, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "nonexistent-channel", spec)
	if err == nil {
		t.Error("Expected error for nonexistent channel")
	}
	
	if !strings.Contains(err.Error(), "channel") {
		t.Errorf("Expected channel-related error, got: %v", err)
	}
}

// TestClientInitializationWithInvalidSpec tests client creation failure with invalid specification
// Matches Swift: testClientInitializationWithInvalidSpec()
func TestClientInitializationWithInvalidSpec(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Create invalid spec (empty version)
	invalidSpec := &gounixsocketapi.APISpecification{
		Version: "", // Empty version should cause validation error
		Name:    "Invalid API",
		Channels: map[string]*gounixsocketapi.ChannelSpec{
			"test-channel": {
				Name: "Test Channel",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"test-command": {
						Name: "Test Command",
					},
				},
			},
		},
	}
	
	_, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "test-channel", invalidSpec)
	if err == nil {
		t.Error("Expected error for invalid specification")
	}
	
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestRegisterValidCommandHandler tests registering a valid command handler
// Matches Swift: testRegisterValidCommandHandler()
func TestRegisterValidCommandHandler(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Register handler for existing command
	handler := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"message": "Handler executed successfully",
		}), nil
	}
	
	err = client.RegisterCommandHandler("get-book", handler)
	if err != nil {
		t.Errorf("Failed to register valid command handler: %v", err)
	}
}

// TestRegisterInvalidCommandHandler tests registering handler for nonexistent command
// Matches Swift: testRegisterInvalidCommandHandler()
func TestRegisterInvalidCommandHandler(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Try to register handler for nonexistent command
	handler := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, nil), nil
	}
	
	err = client.RegisterCommandHandler("nonexistent-command", handler)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
	
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestSocketCommandValidation tests socket command validation against specification
// Matches Swift: testSocketCommandValidation()
func TestSocketCommandValidation(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// Test with valid arguments
	validArgs := map[string]interface{}{
		"id": "book-123",
	}
	
	// This should fail with connection error (expected since no server running)
	// but the command validation should pass
	_, err = client.SendCommand(ctx, "get-book", validArgs, 1*time.Second, nil)
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
	
	_, err = client.SendCommand(ctx, "get-book", invalidArgs, 1*time.Second, nil)
	if err == nil {
		t.Error("Expected validation error for invalid arguments")
	}
	
	if !strings.Contains(err.Error(), "validation") && !strings.Contains(err.Error(), "required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

// TestCommandMessageSerialization tests command message serialization
// Matches Swift: testCommandMessageSerialization()
func TestCommandMessageSerialization(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	args := map[string]interface{}{
		"title":  "Test Book",
		"author": "Test Author",
		"pages":  200,
	}
	
	// Create command directly for serialization testing
	command := gounixsocketapi.NewSocketCommand("library-management", "add-book", args, nil)
	
	// Test serialization
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	// Test deserialization
	var deserializedCommand gounixsocketapi.SocketCommand
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
	testSocketPath1 := "/tmp/gounixsocketapi-client1-test.sock"
	testSocketPath2 := "/tmp/gounixsocketapi-client2-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath1)
	os.Remove(testSocketPath2)
	defer func() {
		os.Remove(testSocketPath1)
		os.Remove(testSocketPath2)
	}()
	
	spec := createComplexAPISpec()
	
	// Create first client
	client1, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath1, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create first client: %v", err)
	}
	defer client1.Close()
	
	// Create second client with different socket path
	client2, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath2, "task-management", spec)
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}
	defer client2.Close()
	
	// Verify independence
	if client1.SocketPathString() == client2.SocketPathString() {
		t.Error("Clients should have different socket paths")
	}
	
	if client1.ChannelIdentifier() == client2.ChannelIdentifier() {
		t.Error("Clients should have different channel identifiers")
	}
	
	// Register different handlers on each client
	handler1 := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"client": "client1",
		}), nil
	}
	
	handler2 := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"client": "client2",
		}), nil
	}
	
	err = client1.RegisterCommandHandler("get-book", handler1)
	if err != nil {
		t.Errorf("Failed to register handler on client1: %v", err)
	}
	
	err = client2.RegisterCommandHandler("create-task", handler2)
	if err != nil {
		t.Errorf("Failed to register handler on client2: %v", err)
	}
}

// TestCommandHandlerWithAsyncOperations tests command handler with async operations
// Matches Swift: testCommandHandlerWithAsyncOperations()
func TestCommandHandlerWithAsyncOperations(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Register async handler
	asyncHandler := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		// Simulate async operation
		time.Sleep(10 * time.Millisecond)
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, map[string]interface{}{
			"processed": true,
			"timestamp": time.Now().Unix(),
		}), nil
	}
	
	err = client.RegisterCommandHandler("get-book", asyncHandler)
	if err != nil {
		t.Errorf("Failed to register async handler: %v", err)
	}
}

// TestCommandHandlerErrorHandling tests command handler error handling
// Matches Swift: testCommandHandlerErrorHandling()
func TestCommandHandlerErrorHandling(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Register error-producing handler
	errorHandler := func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		return nil, &gounixsocketapi.SocketError{
			Code:    "HANDLER_ERROR",
			Message: "Simulated handler error",
			Details: "This is a test error",
		}
	}
	
	err = client.RegisterCommandHandler("get-book", errorHandler)
	if err != nil {
		t.Errorf("Failed to register error handler: %v", err)
	}
}

// TestAPISpecWithComplexArguments tests API specification with complex argument structures
// Matches Swift: testAPISpecWithComplexArguments()
func TestAPISpecWithComplexArguments(t *testing.T) {
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "task-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test complex arguments for create-task command
	complexArgs := map[string]interface{}{
		"title":       "Complex Task",
		"description": "This is a task with complex arguments",
		"priority":    "high",
		"due_date":    "2025-12-31T23:59:59Z",
	}
	
	ctx := context.Background()
	
	// This should fail with connection error (expected since no server running)
	// but the argument validation should pass
	_, err = client.SendCommand(ctx, "create-task", complexArgs, 1*time.Second, nil)
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
	testSocketPath := "/tmp/gounixsocketapi-client-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	spec := createComplexAPISpec()
	client, err := gounixsocketapi.NewUnixSockAPIClient(testSocketPath, "library-management", spec)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// Test with empty ID (should fail validation)
	invalidArgs := map[string]interface{}{
		"id": "",
	}
	
	_, err = client.SendCommand(ctx, "get-book", invalidArgs, 1*time.Second, nil)
	if err == nil {
		t.Error("Expected validation error for empty ID")
	}
	
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Expected validation error, got: %v", err)
	}
	
	// Test with invalid ID pattern (should fail validation if pattern is enforced)
	invalidPatternArgs := map[string]interface{}{
		"id": "invalid id with spaces",
	}
	
	_, err = client.SendCommand(ctx, "get-book", invalidPatternArgs, 1*time.Second, nil)
	if err == nil {
		t.Error("Expected validation error for invalid ID pattern")
	}
}

// Helper function to create a complex API specification for testing
// Matches Swift test helper patterns
func createComplexAPISpec() *gounixsocketapi.APISpecification {
	return &gounixsocketapi.APISpecification{
		Version:     "1.0.0",
		Name:        "Complex Test API",
		Description: "Complex API specification for testing",
		Channels: map[string]*gounixsocketapi.ChannelSpec{
			"library-management": {
				Name:        "Library Management",
				Description: "Library management operations",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"get-book": {
						Name:        "Get Book",
						Description: "Retrieve a book by ID",
						Arguments: map[string]*gounixsocketapi.ArgumentSpec{
							"id": {
								Name:        "Book ID",
								Type:        "string",
								Description: "Unique book identifier",
								Required:    true,
								Pattern:     "^[a-zA-Z0-9-]+$",
								MinLength:   &[]int{1}[0],
								MaxLength:   &[]int{50}[0],
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Book information",
						},
						ErrorCodes: []string{"BOOK_NOT_FOUND", "INVALID_ID"},
					},
					"add-book": {
						Name:        "Add Book",
						Description: "Add a new book",
						Arguments: map[string]*gounixsocketapi.ArgumentSpec{
							"title": {
								Name:        "Title",
								Type:        "string",
								Description: "Book title",
								Required:    true,
								MinLength:   &[]int{1}[0],
								MaxLength:   &[]int{200}[0],
							},
							"author": {
								Name:        "Author",
								Type:        "string",
								Description: "Book author",
								Required:    true,
								MinLength:   &[]int{1}[0],
								MaxLength:   &[]int{100}[0],
							},
							"pages": {
								Name:        "Pages",
								Type:        "integer",
								Description: "Number of pages",
								Required:    false,
								Minimum:     &[]float64{1}[0],
								Maximum:     &[]float64{10000}[0],
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Created book information",
						},
						ErrorCodes: []string{"VALIDATION_ERROR"},
					},
				},
			},
			"task-management": {
				Name:        "Task Management",
				Description: "Task management operations",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"create-task": {
						Name:        "Create Task",
						Description: "Create a new task",
						Arguments: map[string]*gounixsocketapi.ArgumentSpec{
							"title": {
								Name:        "Title",
								Type:        "string",
								Description: "Task title",
								Required:    true,
								MinLength:   &[]int{1}[0],
								MaxLength:   &[]int{100}[0],
							},
							"description": {
								Name:        "Description",
								Type:        "string",
								Description: "Task description",
								Required:    false,
								MaxLength:   &[]int{1000}[0],
							},
							"priority": {
								Name:        "Priority",
								Type:        "string",
								Description: "Task priority",
								Required:    false,
								Enum:        []string{"low", "medium", "high", "urgent"},
							},
							"due_date": {
								Name:        "Due Date",
								Type:        "string",
								Description: "Due date in ISO format",
								Required:    false,
								Pattern:     "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$",
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Created task information",
						},
						ErrorCodes: []string{"VALIDATION_ERROR"},
					},
				},
			},
		},
		Models: map[string]*gounixsocketapi.ModelDefinition{
			"Book": {
				Name:        "Book",
				Type:        "object",
				Description: "Book model",
				Properties: map[string]*gounixsocketapi.ArgumentSpec{
					"id": {
						Name:        "ID",
						Type:        "string",
						Description: "Book ID",
					},
					"title": {
						Name:        "Title",
						Type:        "string",
						Description: "Book title",
					},
				},
				Required: []string{"id", "title"},
			},
		},
	}
}