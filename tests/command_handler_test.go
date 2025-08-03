package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jowharshamshiri/GoJanus/pkg/models"
	"github.com/jowharshamshiri/GoJanus/pkg/server"
)

// TestDirectValueHandlers tests all direct value response handlers
func TestDirectValueHandlers(t *testing.T) {
	tests := []struct {
		name     string
		handler  server.CommandHandler
		expected interface{}
	}{
		{
			name: "BoolHandler",
			handler: server.NewBoolHandler(func(cmd *models.JanusCommand) (bool, error) {
				return true, nil
			}),
			expected: true,
		},
		{
			name: "StringHandler",
			handler: server.NewStringHandler(func(cmd *models.JanusCommand) (string, error) {
				return "test response", nil
			}),
			expected: "test response",
		},
		{
			name: "IntHandler",
			handler: server.NewIntHandler(func(cmd *models.JanusCommand) (int, error) {
				return 42, nil
			}),
			expected: 42,
		},
		{
			name: "FloatHandler",
			handler: server.NewFloatHandler(func(cmd *models.JanusCommand) (float64, error) {
				return 3.14, nil
			}),
			expected: 3.14,
		},
		{
			name: "ArrayHandler",
			handler: server.NewArrayHandler(func(cmd *models.JanusCommand) ([]interface{}, error) {
				return []interface{}{"item1", "item2", 123}, nil
			}),
			expected: []interface{}{"item1", "item2", 123},
		},
		{
			name: "ObjectHandler",
			handler: server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
				return map[string]interface{}{
					"key1": "value1",
					"key2": 42,
					"key3": true,
				}, nil
			}),
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test command
			testCmd := &models.JanusCommand{
				ID:        "test-id",
				ChannelID: "test-channel",
				Command:   "test-command",
				Args:      map[string]interface{}{},
				ReplyTo:   stringPtr("/tmp/test-reply.sock"),
			}

			// Execute handler
			result := tt.handler.Handle(testCmd)

			// Verify result
			if result.Error != nil {
				t.Errorf("Handler returned error: %v", result.Error)
				return
			}

			// Compare result values
			if !compareValues(result.Value, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result.Value)
			}
		})
	}
}

// TestCustomHandlerTypes tests custom type handlers
func TestCustomHandlerTypes(t *testing.T) {
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	handler := server.NewCustomHandler(func(cmd *models.JanusCommand) (User, error) {
		return User{ID: 123, Name: "Test User"}, nil
	})

	testCmd := &models.JanusCommand{
		ID:        "test-id",
		ChannelID: "test-channel",
		Command:   "get-user",
		Args:      map[string]interface{}{},
		ReplyTo:   stringPtr("/tmp/test-reply.sock"),
	}

	result := handler.Handle(testCmd)

	if result.Error != nil {
		t.Errorf("Handler returned error: %v", result.Error)
		return
	}

	// Verify the result is a User struct
	user, ok := result.Value.(User)
	if !ok {
		t.Errorf("Expected User struct, got %T", result.Value)
		return
	}

	if user.ID != 123 || user.Name != "Test User" {
		t.Errorf("Expected User{ID: 123, Name: \"Test User\"}, got %+v", user)
	}
}

// TestAsyncHandlers tests asynchronous handler execution
func TestAsyncHandlers(t *testing.T) {
	tests := []struct {
		name     string
		handler  server.CommandHandler
		expected interface{}
	}{
		{
			name: "AsyncBoolHandler",
			handler: server.NewAsyncBoolHandler(func(cmd *models.JanusCommand) (bool, error) {
				time.Sleep(10 * time.Millisecond) // Simulate async work
				return true, nil
			}),
			expected: true,
		},
		{
			name: "AsyncStringHandler",
			handler: server.NewAsyncStringHandler(func(cmd *models.JanusCommand) (string, error) {
				time.Sleep(10 * time.Millisecond) // Simulate async work
				return "async response", nil
			}),
			expected: "async response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCmd := &models.JanusCommand{
				ID:        "test-id",
				ChannelID: "test-channel",
				Command:   "test-command",
				Args:      map[string]interface{}{},
				ReplyTo:   stringPtr("/tmp/test-reply.sock"),
			}

			start := time.Now()
			result := tt.handler.Handle(testCmd)
			duration := time.Since(start)

			// Verify it took some time (async execution)
			if duration < 10*time.Millisecond {
				t.Errorf("Expected async execution to take at least 10ms, took %v", duration)
			}

			if result.Error != nil {
				t.Errorf("Handler returned error: %v", result.Error)
				return
			}

			if !compareValues(result.Value, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result.Value)
			}
		})
	}
}

// TestHandlerErrorHandling tests error handling in handlers
func TestHandlerErrorHandling(t *testing.T) {
	// Test sync handler error
	syncHandler := server.NewStringHandler(func(cmd *models.JanusCommand) (string, error) {
		return "", fmt.Errorf("sync handler error")
	})

	testCmd := &models.JanusCommand{
		ID:        "test-id",
		ChannelID: "test-channel",
		Command:   "test-command",
		Args:      map[string]interface{}{},
		ReplyTo:   stringPtr("/tmp/test-reply.sock"),
	}

	result := syncHandler.Handle(testCmd)

	if result.Error == nil {
		t.Error("Expected error from sync handler")
		return
	}

	if result.Error.Message != "sync handler error" {
		t.Errorf("Expected error message 'sync handler error', got '%s'", result.Error.Message)
	}

	// Test async handler error
	asyncHandler := server.NewAsyncStringHandler(func(cmd *models.JanusCommand) (string, error) {
		return "", fmt.Errorf("async handler error")
	})

	result = asyncHandler.Handle(testCmd)

	if result.Error == nil {
		t.Error("Expected error from async handler")
		return
	}

	if result.Error.Message != "async handler error" {
		t.Errorf("Expected error message 'async handler error', got '%s'", result.Error.Message)
	}
}

// TestHandlerRegistry tests handler registration and retrieval
func TestHandlerRegistry(t *testing.T) {
	registry := server.NewHandlerRegistry()

	// Test registration
	handler := server.NewStringHandler(func(cmd *models.JanusCommand) (string, error) {
		return "test", nil
	})

	registry.RegisterHandler("test-command", handler)

	// Test retrieval
	retrievedHandler, exists := registry.GetHandler("test-command")
	if !exists {
		t.Error("Expected handler to exist")
		return
	}

	if retrievedHandler == nil {
		t.Error("Expected non-nil handler")
		return
	}

	// Test execution of retrieved handler
	testCmd := &models.JanusCommand{
		ID:        "test-id",
		ChannelID: "test-channel",
		Command:   "test-command",
		Args:      map[string]interface{}{},
		ReplyTo:   stringPtr("/tmp/test-reply.sock"),
	}

	result := retrievedHandler.Handle(testCmd)

	if result.Error != nil {
		t.Errorf("Handler returned error: %v", result.Error)
		return
	}

	if result.Value != "test" {
		t.Errorf("Expected 'test', got %v", result.Value)
	}

	// Test unregistration
	registry.UnregisterHandler("test-command")

	_, exists = registry.GetHandler("test-command")
	if exists {
		t.Error("Expected handler to not exist after unregistration")
	}
}

// TestJSONRPCErrorMapping tests JSON-RPC error code mapping
func TestJSONRPCErrorMapping(t *testing.T) {
	handler := server.NewStringHandler(func(cmd *models.JanusCommand) (string, error) {
		// Return a JSON-RPC error
		return "", &models.JSONRPCError{
			Code:    models.InvalidParams,
			Message: "Invalid parameters provided",
			Data:    &models.JSONRPCErrorData{Context: map[string]interface{}{"field": "missing_arg"}},
		}
	})

	testCmd := &models.JanusCommand{
		ID:        "test-id",
		ChannelID: "test-channel",
		Command:   "test-command",
		Args:      map[string]interface{}{},
		ReplyTo:   stringPtr("/tmp/test-reply.sock"),
	}

	result := handler.Handle(testCmd)

	if result.Error == nil {
		t.Error("Expected JSON-RPC error")
		return
	}

	if result.Error.Code != models.InvalidParams {
		t.Errorf("Expected error code %d, got %d", models.InvalidParams, result.Error.Code)
	}

	if result.Error.Message != "Invalid parameters provided" {
		t.Errorf("Expected error message 'Invalid parameters provided', got '%s'", result.Error.Message)
	}
}

// TestHandlerArguments tests handler access to command arguments
func TestHandlerArguments(t *testing.T) {
	handler := server.NewObjectHandler(func(cmd *models.JanusCommand) (map[string]interface{}, error) {
		// Extract arguments and return them
		name, nameOk := cmd.Args["name"].(string)
		age, ageOk := cmd.Args["age"].(float64)
		
		if !nameOk || !ageOk {
			return nil, fmt.Errorf("missing required arguments")
		}

		return map[string]interface{}{
			"processed_name": fmt.Sprintf("Hello, %s", name),
			"processed_age":  int(age) + 1,
			"original_command": cmd.Command,
		}, nil
	})

	testCmd := &models.JanusCommand{
		ID:        "test-id",
		ChannelID: "test-channel",
		Command:   "process-user",
		Args: map[string]interface{}{
			"name": "John",
			"age":  25.0,
		},
		ReplyTo: stringPtr("/tmp/test-reply.sock"),
	}

	result := handler.Handle(testCmd)

	if result.Error != nil {
		t.Errorf("Handler returned error: %v", result.Error)
		return
	}

	response, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Errorf("Expected map response, got %T", result.Value)
		return
	}

	expectedName := "Hello, John"
	if response["processed_name"] != expectedName {
		t.Errorf("Expected processed_name '%s', got '%v'", expectedName, response["processed_name"])
	}

	expectedAge := 26
	if response["processed_age"] != expectedAge {
		t.Errorf("Expected processed_age %d, got %v", expectedAge, response["processed_age"])
	}

	if response["original_command"] != "process-user" {
		t.Errorf("Expected original_command 'process-user', got '%v'", response["original_command"])
	}
}

// Helper function to compare values (handles different types)
func compareValues(a, b interface{}) bool {
	aBytes, err := json.Marshal(a)
	if err != nil {
		return false
	}
	
	bBytes, err := json.Marshal(b)
	if err != nil {
		return false
	}
	
	return string(aBytes) == string(bBytes)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}