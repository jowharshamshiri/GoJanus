package specification

import (
	"strconv"
	"strings"
	"testing"
)

func TestResponseValidator(t *testing.T) {
	// Create test Manifest matching TypeScript test structure
	testManifest := &Manifest{
		Version:     "1.0.0",
		Name:        "Test API",
		Description: "Test Manifest for response validation",
		Channels: map[string]*ChannelSpec{
			"test": {
				Name:        "test",
				Description: "Test channel",
				Commands: map[string]*CommandSpec{
					"ping": {
						Name:        "ping",
						Description: "Basic ping command",
						Response: &ResponseSpec{
							Type:        "object",
							Description: "Ping response",
							Properties: map[string]*ArgumentSpec{
								"status": {
									Type:        "string",
									Required:    true,
									Description: "Status message",
								},
								"echo": {
									Type:        "string",
									Required:    true,
									Description: "Echo message",
								},
								"timestamp": {
									Type:        "number",
									Required:    true,
									Description: "Response timestamp",
								},
								"server_id": {
									Type:        "string",
									Required:    true,
									Description: "Server identifier",
								},
								"request_count": {
									Type:        "number",
									Required:    false,
									Description: "Request count",
								},
								"metadata": {
									Type:        "object",
									Required:    false,
									Description: "Optional metadata",
								},
							},
						},
					},
					"get_info": {
						Name:        "get_info",
						Description: "Get server information",
						Response: &ResponseSpec{
							Type:        "object",
							Description: "Server information",
							Properties: map[string]*ArgumentSpec{
								"implementation": {
									Type:        "string",
									Required:    true,
									Description: "Implementation language",
								},
								"version": {
									Type:        "string",
									Required:    true,
									Pattern:     `^\d+\.\d+\.\d+$`,
									Description: "Version string",
								},
								"protocol": {
									Type:        "string",
									Required:    true,
									Enum:        []string{"SOCK_DGRAM"},
									Description: "Protocol type",
								},
							},
						},
					},
					"range_test": {
						Name:        "range_test",
						Description: "Numeric range validation test",
						Response: &ResponseSpec{
							Type:        "object",
							Description: "Range test response",
							Properties: map[string]*ArgumentSpec{
								"score": {
									Type:        "number",
									Required:    true,
									Minimum:     ptrFloat64(0),
									Maximum:     ptrFloat64(100),
									Description: "Test score",
								},
								"grade": {
									Type:        "string",
									Required:    true,
									Enum:        []string{"A", "B", "C", "D", "F"},
									Description: "Letter grade",
								},
								"count": {
									Type:        "integer",
									Required:    true,
									Minimum:     ptrFloat64(1),
									Description: "Item count",
								},
							},
						},
					},
					"array_test": {
						Name:        "array_test",
						Description: "Array validation test",
						Response: &ResponseSpec{
							Type:        "object",
							Description: "Array test response",
							Properties: map[string]*ArgumentSpec{
								"items": {
									Type:        "array",
									Required:    true,
									Description: "Array of strings",
									Items: &ArgumentSpec{
										Type:        "string",
										MinLength:   ptrInt(1),
										MaxLength:   ptrInt(50),
										Description: "String item",
									},
								},
								"numbers": {
									Type:        "array",
									Required:    false,
									Description: "Array of numbers",
									Items: &ArgumentSpec{
										Type:        "number",
										Minimum:     ptrFloat64(0),
										Description: "Number item",
									},
								},
							},
						},
					},
				},
			},
		},
		Models: map[string]*ModelDefinition{
			"UserInfo": {
				Name:        "UserInfo",
				Type:        "object",
				Description: "User information model",
				Properties: map[string]*ArgumentSpec{
					"id": {
						Type:        "string",
						Required:    true,
						Description: "User ID",
					},
					"name": {
						Type:        "string",
						Required:    true,
						MinLength:   ptrInt(1),
						MaxLength:   ptrInt(100),
						Description: "User name",
					},
					"age": {
						Type:        "integer",
						Required:    false,
						Minimum:     ptrFloat64(0),
						Maximum:     ptrFloat64(150),
						Description: "User age",
					},
				},
			},
		},
	}

	validator := NewResponseValidator(testManifest)

	t.Run("Basic Response Validation", func(t *testing.T) {
		t.Run("should validate correct ping response", func(t *testing.T) {
			response := map[string]interface{}{
				"status":     "ok",
				"echo":       "test message",
				"timestamp":  1234567890.0,
				"server_id":  "server-001",
			}

			result := validator.ValidateCommandResponse(response, "test", "ping")

			if !result.Valid {
				t.Errorf("Expected valid response, got invalid with errors: %+v", result.Errors)
			}
			if len(result.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(result.Errors))
			}
			if result.FieldsValidated != 6 {
				t.Errorf("Expected 6 fields validated, got %d", result.FieldsValidated)
			}
			if result.ValidationTime <= 0 {
				t.Errorf("Expected positive validation time, got %f", result.ValidationTime)
			}
		})

		t.Run("should validate response with optional fields", func(t *testing.T) {
			response := map[string]interface{}{
				"status":        "ok",
				"echo":          "test message",
				"timestamp":     1234567890.0,
				"server_id":     "server-001",
				"request_count": 42.0,
				"metadata":      map[string]interface{}{"custom": "data"},
			}

			result := validator.ValidateCommandResponse(response, "test", "ping")

			if !result.Valid {
				t.Errorf("Expected valid response, got invalid with errors: %+v", result.Errors)
			}
			if len(result.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(result.Errors))
			}
		})

		t.Run("should fail validation for missing required fields", func(t *testing.T) {
			response := map[string]interface{}{
				"status": "ok",
				"echo":   "test message",
				// Missing timestamp and server_id
			}

			result := validator.ValidateCommandResponse(response, "test", "ping")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(result.Errors) != 2 {
				t.Errorf("Expected 2 errors, got %d", len(result.Errors))
			}

			// Check that we have errors for the missing fields
			hasTimestampError := false
			hasServerIdError := false
			for _, err := range result.Errors {
				if err.Field == "timestamp" {
					hasTimestampError = true
				}
				if err.Field == "server_id" {
					hasServerIdError = true
				}
			}
			if !hasTimestampError {
				t.Errorf("Expected timestamp error")
			}
			if !hasServerIdError {
				t.Errorf("Expected server_id error")
			}
		})

		t.Run("should fail validation for incorrect types", func(t *testing.T) {
			response := map[string]interface{}{
				"status":    123,         // Should be string
				"echo":      true,        // Should be string
				"timestamp": "1234567890", // Should be number
				"server_id": nil,         // Should be string, null not allowed for required field
			}

			result := validator.ValidateCommandResponse(response, "test", "ping")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(result.Errors) != 4 {
				t.Errorf("Expected 4 errors, got %d", len(result.Errors))
			}
		})
	})

	t.Run("Type-Specific Validation", func(t *testing.T) {
		t.Run("should validate string patterns", func(t *testing.T) {
			validResponse := map[string]interface{}{
				"implementation": "Go",
				"version":        "1.2.3",
				"protocol":       "SOCK_DGRAM",
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "get_info")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"implementation": "Go",
				"version":        "1.2", // Invalid pattern - should be x.y.z
				"protocol":       "SOCK_DGRAM",
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "get_info")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}

			hasPatternError := false
			for _, err := range invalidResult.Errors {
				if err.Field == "version" && containsString(err.Message, "pattern") {
					hasPatternError = true
				}
			}
			if !hasPatternError {
				t.Errorf("Expected pattern validation error for version field")
			}
		})

		t.Run("should validate enum values", func(t *testing.T) {
			validResponse := map[string]interface{}{
				"implementation": "Go",
				"version":        "1.0.0",
				"protocol":       "SOCK_DGRAM",
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "get_info")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"implementation": "Go",
				"version":        "1.0.0",
				"protocol":       "SOCK_STREAM", // Invalid enum value
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "get_info")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}

			hasEnumError := false
			for _, err := range invalidResult.Errors {
				if err.Field == "protocol" && containsString(err.Message, "enum") {
					hasEnumError = true
				}
			}
			if !hasEnumError {
				t.Errorf("Expected enum validation error for protocol field")
			}
		})

		t.Run("should validate numeric ranges", func(t *testing.T) {
			validResponse := map[string]interface{}{
				"score": 85.5,
				"grade": "B",
				"count": 10.0,
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "range_test")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"score": 150.0, // > maximum of 100
				"grade": "X",    // Invalid enum
				"count": 0.0,    // < minimum of 1
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "range_test")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(invalidResult.Errors) != 3 {
				t.Errorf("Expected 3 errors, got %d", len(invalidResult.Errors))
			}

			hasScoreError := false
			hasGradeError := false
			hasCountError := false
			for _, err := range invalidResult.Errors {
				if err.Field == "score" && containsString(err.Message, "too large") {
					hasScoreError = true
				}
				if err.Field == "grade" && containsString(err.Message, "enum") {
					hasGradeError = true
				}
				if err.Field == "count" && containsString(err.Message, "too small") {
					hasCountError = true
				}
			}
			if !hasScoreError {
				t.Errorf("Expected score error")
			}
			if !hasGradeError {
				t.Errorf("Expected grade error")
			}
			if !hasCountError {
				t.Errorf("Expected count error")
			}
		})

		t.Run("should validate integers vs numbers", func(t *testing.T) {
			validResponse := map[string]interface{}{
				"score": 85.5, // number is fine
				"grade": "B",
				"count": 10.0, // integer is fine (as float)
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "range_test")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"score": 85.0,
				"grade": "B",
				"count": 10.5, // Should be integer, not float
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "range_test")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}

			hasIntegerError := false
			for _, err := range invalidResult.Errors {
				if err.Field == "count" && containsString(err.Message, "integer") {
					hasIntegerError = true
				}
			}
			if !hasIntegerError {
				t.Errorf("Expected integer validation error for count field")
			}
		})

		t.Run("should validate arrays", func(t *testing.T) {
			validResponse := map[string]interface{}{
				"items":   []interface{}{"hello", "world"},
				"numbers": []interface{}{1.0, 2.0, 3.5},
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "array_test")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"items":   []interface{}{"", "x" + stringRepeat("x", 100)}, // Empty string and too long string
				"numbers": []interface{}{1.0, -5.0, "not a number"},        // Negative number and wrong type
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "array_test")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(invalidResult.Errors) == 0 {
				t.Errorf("Expected validation errors")
			}

			hasTooShortError := false
			hasTooLongError := false
			for _, err := range invalidResult.Errors {
				if containsString(err.Field, "[0]") && containsString(err.Message, "too short") {
					hasTooShortError = true
				}
				if containsString(err.Field, "[1]") && containsString(err.Message, "too long") {
					hasTooLongError = true
				}
			}
			if !hasTooShortError {
				t.Errorf("Expected too short error for array item [0]")
			}
			if !hasTooLongError {
				t.Errorf("Expected too long error for array item [1]")
			}
		})
	})

	t.Run("Error Handling", func(t *testing.T) {
		t.Run("should handle missing channel", func(t *testing.T) {
			response := map[string]interface{}{"status": "ok"}

			result := validator.ValidateCommandResponse(response, "nonexistent", "ping")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(result.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(result.Errors))
			}
			if result.Errors[0].Field != "channelId" {
				t.Errorf("Expected channelId field error, got %s", result.Errors[0].Field)
			}
			if !containsString(result.Errors[0].Message, "Channel 'nonexistent' not found") {
				t.Errorf("Expected channel not found message, got %s", result.Errors[0].Message)
			}
		})

		t.Run("should handle missing command", func(t *testing.T) {
			response := map[string]interface{}{"status": "ok"}

			result := validator.ValidateCommandResponse(response, "test", "nonexistent")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(result.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(result.Errors))
			}
			if result.Errors[0].Field != "command" {
				t.Errorf("Expected command field error, got %s", result.Errors[0].Field)
			}
			if !containsString(result.Errors[0].Message, "Command 'nonexistent' not found") {
				t.Errorf("Expected command not found message, got %s", result.Errors[0].Message)
			}
		})

		t.Run("should handle missing response specification", func(t *testing.T) {
			// Add command without response specification
			testManifest.Channels["test"].Commands["no_response"] = &CommandSpec{
				Name:        "no_response",
				Description: "Command without response spec",
				// No Response field
			}

			validator := NewResponseValidator(testManifest)
			response := map[string]interface{}{"status": "ok"}

			result := validator.ValidateCommandResponse(response, "test", "no_response")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}
			if len(result.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(result.Errors))
			}
			if result.Errors[0].Field != "response" {
				t.Errorf("Expected response field error, got %s", result.Errors[0].Field)
			}
			if !containsString(result.Errors[0].Message, "No response specification defined") {
				t.Errorf("Expected no response specification message, got %s", result.Errors[0].Message)
			}
		})
	})

	t.Run("Performance", func(t *testing.T) {
		t.Run("should complete validation within performance requirements", func(t *testing.T) {
			response := map[string]interface{}{
				"status":        "ok",
				"echo":          "test message",
				"timestamp":     1234567890.0,
				"server_id":     "server-001",
				"request_count": 42.0,
				"metadata":      map[string]interface{}{"custom": "data", "nested": map[string]interface{}{"deep": "value"}},
			}

			result := validator.ValidateCommandResponse(response, "test", "ping")

			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}
			if result.ValidationTime >= 2.0 {
				t.Errorf("Expected validation time < 2ms, got %f ms", result.ValidationTime)
			}
		})

		t.Run("should handle large responses efficiently", func(t *testing.T) {
			largeItems := make([]interface{}, 1000)
			largeNumbers := make([]interface{}, 1000)
			for i := 0; i < 1000; i++ {
				largeItems[i] = "item-" + stringFromInt(i)
				largeNumbers[i] = float64(i)
			}

			largeResponse := map[string]interface{}{
				"items":   largeItems,
				"numbers": largeNumbers,
			}

			result := validator.ValidateCommandResponse(largeResponse, "test", "array_test")

			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}
			if result.ValidationTime >= 10.0 {
				t.Errorf("Expected validation time < 10ms for large response, got %f ms", result.ValidationTime)
			}
		})
	})

	t.Run("Static Functions", func(t *testing.T) {
		t.Run("should create missing specification error", func(t *testing.T) {
			result := CreateMissingSpecificationError("test", "unknown")

			if result.Valid {
				t.Errorf("Expected invalid result")
			}
			if len(result.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(result.Errors))
			}
			if result.Errors[0].Field != "specification" {
				t.Errorf("Expected specification field, got %s", result.Errors[0].Field)
			}
			if !containsString(result.Errors[0].Message, "No response specification found") {
				t.Errorf("Expected no response specification message, got %s", result.Errors[0].Message)
			}
			if result.FieldsValidated != 0 {
				t.Errorf("Expected 0 fields validated, got %d", result.FieldsValidated)
			}
			if result.ValidationTime != 0 {
				t.Errorf("Expected 0 validation time, got %f", result.ValidationTime)
			}
		})

		t.Run("should create success result", func(t *testing.T) {
			result := CreateSuccessResult(5, 1.5)

			if !result.Valid {
				t.Errorf("Expected valid result")
			}
			if len(result.Errors) != 0 {
				t.Errorf("Expected 0 errors, got %d", len(result.Errors))
			}
			if result.FieldsValidated != 5 {
				t.Errorf("Expected 5 fields validated, got %d", result.FieldsValidated)
			}
			if result.ValidationTime != 1.5 {
				t.Errorf("Expected 1.5 validation time, got %f", result.ValidationTime)
			}
		})
	})

	t.Run("Model References", func(t *testing.T) {
		t.Run("should handle model references", func(t *testing.T) {
			// Add command that uses model reference
			testManifest.Channels["test"].Commands["user_info"] = &CommandSpec{
				Name:        "user_info",
				Description: "Get user information",
				Response: &ResponseSpec{
					Type:        "object",
					Description: "User info response",
					ModelRef:    "UserInfo",
				},
			}

			validator := NewResponseValidator(testManifest)

			validResponse := map[string]interface{}{
				"id":   "user123",
				"name": "John Doe",
				"age":  30.0,
			}

			result := validator.ValidateCommandResponse(validResponse, "test", "user_info")
			if !result.Valid {
				t.Errorf("Expected valid response, got errors: %+v", result.Errors)
			}

			invalidResponse := map[string]interface{}{
				"id":   "user123",
				"name": "",     // Too short
				"age":  200.0,  // Too old
			}

			invalidResult := validator.ValidateCommandResponse(invalidResponse, "test", "user_info")
			if invalidResult.Valid {
				t.Errorf("Expected invalid response")
			}

			hasNameError := false
			hasAgeError := false
			for _, err := range invalidResult.Errors {
				if err.Field == "name" {
					hasNameError = true
				}
				if err.Field == "age" {
					hasAgeError = true
				}
			}
			if !hasNameError {
				t.Errorf("Expected name validation error")
			}
			if !hasAgeError {
				t.Errorf("Expected age validation error")
			}
		})

		t.Run("should handle missing model reference", func(t *testing.T) {
			testManifest.Channels["test"].Commands["bad_ref"] = &CommandSpec{
				Name:        "bad_ref",
				Description: "Command with bad model reference",
				Response: &ResponseSpec{
					Type:        "object",
					Description: "Response with bad model ref",
					ModelRef:    "NonexistentModel",
				},
			}

			validator := NewResponseValidator(testManifest)

			response := map[string]interface{}{"data": "test"}
			result := validator.ValidateCommandResponse(response, "test", "bad_ref")

			if result.Valid {
				t.Errorf("Expected invalid response")
			}

			hasMissingModelError := false
			for _, err := range result.Errors {
				if containsString(err.Message, "Model reference 'NonexistentModel' not found") {
					hasMissingModelError = true
				}
			}
			if !hasMissingModelError {
				t.Errorf("Expected missing model reference error")
			}
		})
	})
}

// Helper functions for tests

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt(i int) *int {
	return &i
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(s) >= len(substr) && s[:len(substr)] == substr ||
		   strings.Contains(s, substr)
}

func stringRepeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func stringFromInt(i int) string {
	return strconv.Itoa(i)
}