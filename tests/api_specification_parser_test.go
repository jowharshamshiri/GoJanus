package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/GoUnixSockAPI"
)

// TestParseJSONSpecification tests parsing a valid JSON API specification
// Matches Swift: testParseJSONSpecification()
func TestParseJSONSpecification(t *testing.T) {
	jsonData := `{
		"version": "1.0.0",
		"name": "Test API",
		"description": "Test API specification",
		"channels": {
			"test-channel": {
				"name": "Test Channel",
				"description": "Test channel",
				"commands": {
					"test-command": {
						"name": "Test Command",
						"description": "Test command",
						"arguments": {
							"arg1": {
								"name": "Argument 1",
								"type": "string",
								"description": "Test argument",
								"required": true
							}
						},
						"response": {
							"type": "object",
							"description": "Test response"
						}
					}
				}
			}
		}
	}`
	
	spec, err := gounixsocketapi.ParseAPISpecFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse JSON specification: %v", err)
	}
	
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
		t.Fatal("Expected 'test-channel' to exist")
	}
	
	if len(channel.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(channel.Commands))
	}
	
	command, exists := channel.Commands["test-command"]
	if !exists {
		t.Fatal("Expected 'test-command' to exist")
	}
	
	if len(command.Args) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(command.Args))
	}
	
	arg, exists := command.Args["arg1"]
	if !exists {
		t.Fatal("Expected 'arg1' to exist")
	}
	
	if arg.Type != "string" {
		t.Errorf("Expected argument type 'string', got '%s'", arg.Type)
	}
	
	if !arg.Required {
		t.Error("Expected argument to be required")
	}
}

// TestParseYAMLSpecification tests parsing a valid YAML API specification
// Matches Swift: testParseYAMLSpecification()
func TestParseYAMLSpecification(t *testing.T) {
	yamlData := `
version: "1.0.0"
name: "Test API"
description: "Test API specification"
channels:
  test-channel:
    name: "Test Channel"
    description: "Test channel"
    commands:
      test-command:
        name: "Test Command"
        description: "Test command"
        arguments:
          arg1:
            name: "Argument 1"
            type: "string"
            description: "Test argument"
            required: true
        response:
          type: "object"
          description: "Test response"
`
	
	spec, err := gounixsocketapi.ParseAPISpecFromYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to parse YAML specification: %v", err)
	}
	
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
		t.Fatal("Expected 'test-channel' to exist")
	}
	
	command, exists := channel.Commands["test-command"]
	if !exists {
		t.Fatal("Expected 'test-command' to exist")
	}
	
	arg, exists := command.Args["arg1"]
	if !exists {
		t.Fatal("Expected 'arg1' to exist")
	}
	
	if arg.Type != "string" {
		t.Errorf("Expected argument type 'string', got '%s'", arg.Type)
	}
}

// TestValidateValidSpecification tests validation of a valid specification
// Matches Swift: testValidateValidSpecification()
func TestValidateValidSpecification(t *testing.T) {
	spec := createValidAPISpecification()
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err != nil {
		t.Errorf("Valid specification should not produce validation error: %v", err)
	}
}

// TestValidateSpecificationWithEmptyVersion tests validation failure for empty version
// Matches Swift: testValidateSpecificationWithEmptyVersion()
func TestValidateSpecificationWithEmptyVersion(t *testing.T) {
	spec := createValidAPISpecification()
	spec.Version = ""
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for empty version")
	}
	
	if !strings.Contains(err.Error(), "version is required") {
		t.Errorf("Expected version-related error, got: %v", err)
	}
}

// TestValidateSpecificationWithNoChannels tests validation failure for no channels
// Matches Swift: testValidateSpecificationWithNoChannels()
func TestValidateSpecificationWithNoChannels(t *testing.T) {
	spec := createValidAPISpecification()
	spec.Channels = make(map[string]*gounixsocketapi.ChannelSpec)
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for no channels")
	}
	
	if !strings.Contains(err.Error(), "at least one channel") {
		t.Errorf("Expected channels-related error, got: %v", err)
	}
}

// TestValidateSpecificationWithEmptyChannelId tests validation failure for empty channel ID
// Matches Swift: testValidateSpecificationWithEmptyChannelId()
func TestValidateSpecificationWithEmptyChannelId(t *testing.T) {
	spec := createValidAPISpecification()
	
	// Add channel with empty ID
	channelSpec := spec.Channels["test-channel"]
	delete(spec.Channels, "test-channel")
	spec.Channels[""] = channelSpec
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for empty channel ID")
	}
	
	if !strings.Contains(err.Error(), "channel ID cannot be empty") {
		t.Errorf("Expected channel ID error, got: %v", err)
	}
}

// TestValidateSpecificationWithNoCommands tests validation failure for no commands
// Matches Swift: testValidateSpecificationWithNoCommands()
func TestValidateSpecificationWithNoCommands(t *testing.T) {
	spec := createValidAPISpecification()
	spec.Channels["test-channel"].Commands = make(map[string]*gounixsocketapi.CommandSpec)
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for no commands")
	}
	
	if !strings.Contains(err.Error(), "at least one command") {
		t.Errorf("Expected commands-related error, got: %v", err)
	}
}

// TestValidateSpecificationWithEmptyCommandName tests validation failure for empty command name
// Matches Swift: testValidateSpecificationWithEmptyCommandName()
func TestValidateSpecificationWithEmptyCommandName(t *testing.T) {
	spec := createValidAPISpecification()
	
	// Add command with empty name
	commandSpec := spec.Channels["test-channel"].Commands["test-command"]
	delete(spec.Channels["test-channel"].Commands, "test-command")
	spec.Channels["test-channel"].Commands[""] = commandSpec
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for empty command name")
	}
	
	if !strings.Contains(err.Error(), "command name cannot be empty") {
		t.Errorf("Expected command name error, got: %v", err)
	}
}

// TestValidateSpecificationWithInvalidValidation tests validation failure for invalid argument spec
// Matches Swift: testValidateSpecificationWithInvalidValidation()
func TestValidateSpecificationWithInvalidValidation(t *testing.T) {
	spec := createValidAPISpecification()
	
	// Add invalid argument specification (empty type)
	spec.Channels["test-channel"].Commands["test-command"].Args["invalid_arg"] = &gounixsocketapi.ArgumentSpec{
		Name:        "Invalid Argument",
		Type:        "", // Empty type should cause validation error
		Description: "Invalid argument",
		Required:    true,
	}
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for invalid argument specification")
	}
	
	if !strings.Contains(err.Error(), "type is required") {
		t.Errorf("Expected type-related error, got: %v", err)
	}
}

// TestValidateSpecificationWithInvalidRegexPattern tests validation failure for invalid regex
// Matches Swift: testValidateSpecificationWithInvalidRegexPattern()
func TestValidateSpecificationWithInvalidRegexPattern(t *testing.T) {
	spec := createValidAPISpecification()
	
	// Add argument with invalid regex pattern
	spec.Channels["test-channel"].Commands["test-command"].Args["regex_arg"] = &gounixsocketapi.ArgumentSpec{
		Name:        "Regex Argument",
		Type:        "string",
		Description: "Argument with invalid regex",
		Required:    false,
		Pattern:     "[invalid-regex(", // Invalid regex pattern
	}
	
	err := gounixsocketapi.ValidateAPISpec(spec)
	if err == nil {
		t.Error("Expected validation error for invalid regex pattern")
	}
	
	if !strings.Contains(err.Error(), "invalid regex pattern") {
		t.Errorf("Expected regex pattern error, got: %v", err)
	}
}

// TestParseInvalidJSON tests parsing failure for invalid JSON
// Matches Swift: testParseInvalidJSON()
func TestParseInvalidJSON(t *testing.T) {
	invalidJSON := `{
		"version": "1.0.0",
		"name": "Test API",
		"channels": {
			"test-channel": {
				"commands": {
					"test-command": {
						// Invalid comment in JSON
					}
				}
			}
		}
	}` // Missing closing brace
	
	_, err := gounixsocketapi.ParseAPISpecFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected parsing error for invalid JSON")
	}
	
	if !strings.Contains(err.Error(), "failed to parse JSON") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

// TestParseUnsupportedFileFormat tests handling of unsupported file formats
// Matches Swift: testParseUnsupportedFileFormat() 
func TestParseUnsupportedFileFormat(t *testing.T) {
	// Create temporary file with unsupported extension
	tempDir, err := os.MkdirTemp("", "gounixsocketapi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	unsupportedFile := filepath.Join(tempDir, "test.xml")
	content := `<?xml version="1.0"?><root></root>`
	
	err = os.WriteFile(unsupportedFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Try to parse as JSON (should fail)
	_, err = gounixsocketapi.ParseAPISpecFromFile(unsupportedFile)
	if err == nil {
		t.Error("Expected parsing error for unsupported file format")
	}
}

// TestParseFromJSONFile tests parsing from a JSON file
// Matches Swift: testParseFromJSONFile()
func TestParseFromJSONFile(t *testing.T) {
	// Create temporary JSON file
	tempDir, err := os.MkdirTemp("", "gounixsocketapi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	jsonFile := filepath.Join(tempDir, "test.json")
	jsonData := `{
		"version": "1.0.0",
		"name": "File Test API",
		"description": "Test API from file",
		"channels": {
			"file-channel": {
				"name": "File Channel",
				"description": "Test channel from file",
				"commands": {
					"file-command": {
						"name": "File Command",
						"description": "Test command from file"
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(jsonFile, []byte(jsonData), 0644)
	if err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}
	
	spec, err := gounixsocketapi.ParseAPISpecFromFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to parse JSON file: %v", err)
	}
	
	if spec.Name != "File Test API" {
		t.Errorf("Expected name 'File Test API', got '%s'", spec.Name)
	}
	
	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}
	
	channel, exists := spec.Channels["file-channel"]
	if !exists {
		t.Fatal("Expected 'file-channel' to exist")
	}
	
	if channel.Name != "File Channel" {
		t.Errorf("Expected channel name 'File Channel', got '%s'", channel.Name)
	}
}

// TestParseFromYAMLFile tests parsing from a YAML file
// Matches Swift: testParseFromYAMLFile()
func TestParseFromYAMLFile(t *testing.T) {
	// Create temporary YAML file
	tempDir, err := os.MkdirTemp("", "gounixsocketapi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	yamlFile := filepath.Join(tempDir, "test.yaml")
	yamlData := `
version: "1.0.0"
name: "YAML Test API"
description: "Test API from YAML file"
channels:
  yaml-channel:
    name: "YAML Channel"
    description: "Test channel from YAML file"
    commands:
      yaml-command:
        name: "YAML Command"
        description: "Test command from YAML file"
`
	
	err = os.WriteFile(yamlFile, []byte(yamlData), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}
	
	spec, err := gounixsocketapi.ParseAPISpecFromFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to parse YAML file: %v", err)
	}
	
	if spec.Name != "YAML Test API" {
		t.Errorf("Expected name 'YAML Test API', got '%s'", spec.Name)
	}
	
	if len(spec.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(spec.Channels))
	}
	
	channel, exists := spec.Channels["yaml-channel"]
	if !exists {
		t.Fatal("Expected 'yaml-channel' to exist")
	}
	
	if channel.Name != "YAML Channel" {
		t.Errorf("Expected channel name 'YAML Channel', got '%s'", channel.Name)
	}
}

// Helper function to create a valid API specification
// Matches Swift test helper patterns
func createValidAPISpecification() *gounixsocketapi.APISpecification {
	return &gounixsocketapi.APISpecification{
		Version:     "1.0.0",
		Name:        "Valid Test API",
		Description: "Valid test API specification",
		Channels: map[string]*gounixsocketapi.ChannelSpec{
			"test-channel": {
				Name:        "Test Channel",
				Description: "Test channel description",
				Commands: map[string]*gounixsocketapi.CommandSpec{
					"test-command": {
						Name:        "Test Command",
						Description: "Test command description",
						Args: map[string]*gounixsocketapi.ArgumentSpec{
							"test_arg": {
								Name:        "Test Argument",
								Type:        "string",
								Description: "Test argument description",
								Required:    true,
							},
						},
						Response: &gounixsocketapi.ResponseSpec{
							Type:        "object",
							Description: "Test response",
						},
						ErrorCodes: []string{"TEST_ERROR"},
					},
				},
			},
		},
		Models: map[string]*gounixsocketapi.ModelDefinition{
			"TestModel": {
				Name:        "Test Model",
				Type:        "object",
				Description: "Test model description",
				Properties: map[string]*gounixsocketapi.ArgumentSpec{
					"id": {
						Name:        "ID",
						Type:        "string",
						Description: "Model ID",
						Required:    true,
					},
				},
				Required: []string{"id"},
			},
		},
	}
}