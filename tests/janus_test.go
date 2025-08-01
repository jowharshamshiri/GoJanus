package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/user/GoJanus"
)

// TestAPISpecificationCreation tests basic API specification model creation
// Matches Swift: testAPISpecificationCreation()
func TestAPISpecificationCreation(t *testing.T) {
	spec := createTestAPISpec()
	
	if spec.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", spec.Version)
	}
	
	if spec.Name != "Test API" {
		t.Errorf("Expected name 'Test API', got '%s'", spec.Name)
	}
	
	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}
	
	channel, exists := spec.Channels["test-channel"]
	if !exists {
		t.Error("Expected 'test-channel' to exist")
	}
	
	if len(channel.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(channel.Commands))
	}
	
	_, exists = channel.Commands["test-command"]
	if !exists {
		t.Error("Expected 'test-command' to exist")
	}
}

// TestAPISpecificationJSONSerialization tests JSON serialization of API specifications
// Matches Swift: testAPISpecificationJSONSerialization()
func TestAPISpecificationJSONSerialization(t *testing.T) {
	spec := createTestAPISpec()
	
	// Serialize to JSON
	jsonData, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("Failed to marshal spec to JSON: %v", err)
	}
	
	// Deserialize back
	var deserializedSpec gojanus.APISpecification
	err = json.Unmarshal(jsonData, &deserializedSpec)
	if err != nil {
		t.Fatalf("Failed to unmarshal spec from JSON: %v", err)
	}
	
	// Verify integrity
	if deserializedSpec.Version != spec.Version {
		t.Errorf("Version mismatch after serialization: expected '%s', got '%s'", spec.Version, deserializedSpec.Version)
	}
	
	if deserializedSpec.Name != spec.Name {
		t.Errorf("Name mismatch after serialization: expected '%s', got '%s'", spec.Name, deserializedSpec.Name)
	}
	
	if len(deserializedSpec.Channels) != len(spec.Channels) {
		t.Errorf("Channel count mismatch after serialization: expected %d, got %d", len(spec.Channels), len(deserializedSpec.Channels))
	}
}

// TestSocketCommandSerialization tests socket command JSON serialization
// Matches Swift: testSocketCommandSerialization()
func TestSocketCommandSerialization(t *testing.T) {
	args := map[string]interface{}{
		"test_string": "value",
		"test_int":    42,
		"test_bool":   true,
	}
	timeout := 30.0
	
	command := gojanus.NewSocketCommand("test-channel", "test-command", args, &timeout)
	
	// Serialize to JSON
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command to JSON: %v", err)
	}
	
	// Deserialize back
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command from JSON: %v", err)
	}
	
	// Verify integrity
	if deserializedCommand.ChannelID != command.ChannelID {
		t.Errorf("ChannelID mismatch: expected '%s', got '%s'", command.ChannelID, deserializedCommand.ChannelID)
	}
	
	if deserializedCommand.Command != command.Command {
		t.Errorf("Command mismatch: expected '%s', got '%s'", command.Command, deserializedCommand.Command)
	}
	
	if *deserializedCommand.Timeout != *command.Timeout {
		t.Errorf("Timeout mismatch: expected %f, got %f", *command.Timeout, *deserializedCommand.Timeout)
	}
	
	// Verify arguments
	if len(deserializedCommand.Args) != len(command.Args) {
		t.Errorf("Args count mismatch: expected %d, got %d", len(command.Args), len(deserializedCommand.Args))
	}
	
	if deserializedCommand.Args["test_string"] != args["test_string"] {
		t.Errorf("String arg mismatch: expected '%v', got '%v'", args["test_string"], deserializedCommand.Args["test_string"])
	}
	
	// JSON numbers are float64
	if deserializedCommand.Args["test_int"].(float64) != 42.0 {
		t.Errorf("Int arg mismatch: expected 42, got %v", deserializedCommand.Args["test_int"])
	}
	
	if deserializedCommand.Args["test_bool"] != args["test_bool"] {
		t.Errorf("Bool arg mismatch: expected '%v', got '%v'", args["test_bool"], deserializedCommand.Args["test_bool"])
	}
}

// TestSocketResponseSerialization tests socket response JSON serialization
// Matches Swift: testSocketResponseSerialization()
func TestSocketResponseSerialization(t *testing.T) {
	result := map[string]interface{}{
		"message": "success",
		"code":    200,
		"data":    []interface{}{"item1", "item2"},
	}
	
	response := gojanus.NewSuccessResponse("test-command-id", "test-channel", result)
	
	// Serialize to JSON
	jsonData, err := response.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize response to JSON: %v", err)
	}
	
	// Deserialize back
	var deserializedResponse gojanus.SocketResponse
	err = deserializedResponse.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize response from JSON: %v", err)
	}
	
	// Verify integrity
	if deserializedResponse.CommandID != response.CommandID {
		t.Errorf("CommandID mismatch: expected '%s', got '%s'", response.CommandID, deserializedResponse.CommandID)
	}
	
	if deserializedResponse.ChannelID != response.ChannelID {
		t.Errorf("ChannelID mismatch: expected '%s', got '%s'", response.ChannelID, deserializedResponse.ChannelID)
	}
	
	if deserializedResponse.Success != response.Success {
		t.Errorf("Success mismatch: expected %t, got %t", response.Success, deserializedResponse.Success)
	}
	
	// Verify result - type assert to map for comparison
	responseResultMap, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("response.Result is not a map[string]interface{}")
	}
	
	deserializedResultMap, ok := deserializedResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("deserializedResponse.Result is not a map[string]interface{}")
	}
	
	if len(deserializedResultMap) != len(responseResultMap) {
		t.Errorf("Result count mismatch: expected %d, got %d", len(responseResultMap), len(deserializedResultMap))
	}
	
	if deserializedResultMap["message"] != result["message"] {
		t.Errorf("Message mismatch: expected '%v', got '%v'", result["message"], deserializedResultMap["message"])
	}
}

// TestAnyCodableStringValue tests string value handling in arguments
// Matches Swift: testAnyCodableStringValue()
func TestAnyCodableStringValue(t *testing.T) {
	args := map[string]interface{}{
		"string_value": "test string",
	}
	
	command := gojanus.NewSocketCommand("test-channel", "test-command", args, nil)
	
	// Serialize and deserialize
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command: %v", err)
	}
	
	stringValue, ok := deserializedCommand.Args["string_value"].(string)
	if !ok {
		t.Fatalf("Expected string value, got %T", deserializedCommand.Args["string_value"])
	}
	
	if stringValue != "test string" {
		t.Errorf("Expected 'test string', got '%s'", stringValue)
	}
}

// TestAnyCodableIntegerValue tests integer value handling in arguments
// Matches Swift: testAnyCodableIntegerValue()
func TestAnyCodableIntegerValue(t *testing.T) {
	args := map[string]interface{}{
		"int_value": 42,
	}
	
	command := gojanus.NewSocketCommand("test-channel", "test-command", args, nil)
	
	// Serialize and deserialize
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command: %v", err)
	}
	
	// JSON numbers are always float64 in Go
	floatValue, ok := deserializedCommand.Args["int_value"].(float64)
	if !ok {
		t.Fatalf("Expected float64 value (JSON number), got %T", deserializedCommand.Args["int_value"])
	}
	
	if floatValue != 42.0 {
		t.Errorf("Expected 42.0, got %f", floatValue)
	}
}

// TestAnyCodableBooleanValue tests boolean value handling in arguments
// Matches Swift: testAnyCodableBooleanValue()
func TestAnyCodableBooleanValue(t *testing.T) {
	args := map[string]interface{}{
		"bool_true":  true,
		"bool_false": false,
	}
	
	command := gojanus.NewSocketCommand("test-channel", "test-command", args, nil)
	
	// Serialize and deserialize
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command: %v", err)
	}
	
	trueValue, ok := deserializedCommand.Args["bool_true"].(bool)
	if !ok {
		t.Fatalf("Expected bool value for bool_true, got %T", deserializedCommand.Args["bool_true"])
	}
	
	if !trueValue {
		t.Error("Expected true, got false")
	}
	
	falseValue, ok := deserializedCommand.Args["bool_false"].(bool)
	if !ok {
		t.Fatalf("Expected bool value for bool_false, got %T", deserializedCommand.Args["bool_false"])
	}
	
	if falseValue {
		t.Error("Expected false, got true")
	}
}

// TestAnyCodableArrayValue tests array value handling in arguments
// Matches Swift: testAnyCodableArrayValue()
func TestAnyCodableArrayValue(t *testing.T) {
	args := map[string]interface{}{
		"array_value": []interface{}{"item1", 2, true, []interface{}{"nested", "array"}},
	}
	
	command := gojanus.NewSocketCommand("test-channel", "test-command", args, nil)
	
	// Serialize and deserialize
	jsonData, err := command.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize command: %v", err)
	}
	
	var deserializedCommand gojanus.SocketCommand
	err = deserializedCommand.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize command: %v", err)
	}
	
	arrayValue, ok := deserializedCommand.Args["array_value"].([]interface{})
	if !ok {
		t.Fatalf("Expected []interface{} value, got %T", deserializedCommand.Args["array_value"])
	}
	
	if len(arrayValue) != 4 {
		t.Errorf("Expected array length 4, got %d", len(arrayValue))
	}
	
	if arrayValue[0].(string) != "item1" {
		t.Errorf("Expected 'item1', got '%v'", arrayValue[0])
	}
	
	if arrayValue[1].(float64) != 2.0 { // JSON numbers are float64
		t.Errorf("Expected 2.0, got %v", arrayValue[1])
	}
	
	if arrayValue[2].(bool) != true {
		t.Errorf("Expected true, got %v", arrayValue[2])
	}
	
	nestedArray, ok := arrayValue[3].([]interface{})
	if !ok {
		t.Fatalf("Expected nested array, got %T", arrayValue[3])
	}
	
	if len(nestedArray) != 2 {
		t.Errorf("Expected nested array length 2, got %d", len(nestedArray))
	}
	
	if nestedArray[0].(string) != "nested" {
		t.Errorf("Expected 'nested', got '%v'", nestedArray[0])
	}
}

// TestJanusClientInitialization tests SOCK_DGRAM client creation
// Matches Swift: testJanusClientInitialization()
func TestJanusClientInitialization(t *testing.T) {
	testSocketPath := "/tmp/gojanus-dgram-test.sock"
	
	// Clean up before test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Create test API spec and client for SOCK_DGRAM
	spec := createTestAPISpec()
	client, err := gojanus.NewJanusClient(testSocketPath, "test-channel")
	if err != nil {
		t.Fatalf("Failed to create SOCK_DGRAM client: %v", err)
	}
	// Note: SOCK_DGRAM clients are stateless and don't need cleanup
	
	if client.SocketPathString() != testSocketPath {
		t.Errorf("Expected socket path '%s', got '%s'", testSocketPath, client.SocketPathString())
	}
	
	if client.ChannelIdentifier() != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", client.ChannelIdentifier())
	}
	
	if client.Specification() == nil {
		t.Error("Expected specification to be set")
	}
}

// Helper function to create a test API specification
// Matches Swift test helper patterns
func createTestAPISpec() *gojanus.APISpecification {
	return &gojanus.APISpecification{
		Version:     "1.0.0",
		Name:        "Test API",
		Description: "Test API specification",
		Channels: map[string]*gojanus.ChannelSpec{
			"test-channel": {
				Name:        "Test Channel",
				Description: "Test channel description",
				Commands: map[string]*gojanus.CommandSpec{
					"test-command": {
						Name:        "Test Command",
						Description: "Test command description",
						Args: map[string]*gojanus.ArgumentSpec{
							"test_arg": {
								Name:        "Test Argument",
								Type:        "string",
								Description: "Test argument description",
								Required:    true,
							},
						},
						Response: &gojanus.ResponseSpec{
							Type:        "object",
							Description: "Test response",
						},
						ErrorCodes: []string{"TEST_ERROR"},
					},
				},
			},
		},
	}
}