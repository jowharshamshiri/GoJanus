package tests

import (
	"os"
	"path/filepath"
	"strings" 
	"testing"

	gojanus "github.com/jowharshamshiri/GoJanus/pkg/specification"
)

// TestParseJSONSpecification tests parsing a valid JSON Manifest
// Matches Swift: testParseJSONSpecification()
func TestParseJSONSpecification(t *testing.T) {
	jsonData := `{
		"version": "1.0.0",
		"name": "Test API",
		"description": "Test Manifest",
		"channels": {
			"test-channel": {
				"name": "Test Channel",
				"description": "Test channel",
				"commands": {
					"test-command": {
						"name": "Test Command",
						"description": "Test command",
						"args": {
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
	
	spec, err := gojanus.ParseJSON([]byte(jsonData))
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

// TestParseYAMLSpecification tests parsing a valid YAML Manifest
// Matches Swift: testParseYAMLSpecification()
func TestParseYAMLSpecification(t *testing.T) {
	yamlData := `
version: "1.0.0"
name: "Test API"
description: "Test Manifest"
channels:
  test-channel:
    name: "Test Channel"
    description: "Test channel"
    commands:
      test-command:
        name: "Test Command"
        description: "Test command"
        args:
          arg1:
            name: "Argument 1"
            type: "string"
            description: "Test argument"
            required: true
        response:
          type: "object"
          description: "Test response"
`
	
	spec, err := gojanus.ParseYAML([]byte(yamlData))
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
	spec := createValidManifest()
	
	err := gojanus.Validate(spec)
	if err != nil {
		t.Errorf("Valid specification should not produce validation error: %v", err)
	}
}

// TestValidateSpecificationWithEmptyVersion tests validation failure for empty version
// Matches Swift: testValidateSpecificationWithEmptyVersion()
func TestValidateSpecificationWithEmptyVersion(t *testing.T) {
	spec := createValidManifest()
	spec.Version = ""
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	spec.Channels = make(map[string]*gojanus.ChannelSpec)
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	
	// Add channel with empty ID
	channelSpec := spec.Channels["test-channel"]
	delete(spec.Channels, "test-channel")
	spec.Channels[""] = channelSpec
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	spec.Channels["test-channel"].Commands = make(map[string]*gojanus.CommandSpec)
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	
	// Add command with empty name
	commandSpec := spec.Channels["test-channel"].Commands["test-command"]
	delete(spec.Channels["test-channel"].Commands, "test-command")
	spec.Channels["test-channel"].Commands[""] = commandSpec
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	
	// Add invalid argument specification (empty type)
	spec.Channels["test-channel"].Commands["test-command"].Args["invalid_arg"] = &gojanus.ArgumentSpec{
		Name:        "Invalid Argument",
		Type:        "", // Empty type should cause validation error
		Description: "Invalid argument",
		Required:    true,
	}
	
	err := gojanus.Validate(spec)
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
	spec := createValidManifest()
	
	// Add argument with invalid regex pattern
	spec.Channels["test-channel"].Commands["test-command"].Args["regex_arg"] = &gojanus.ArgumentSpec{
		Name:        "Regex Argument",
		Type:        "string",
		Description: "Argument with invalid regex",
		Required:    false,
		Pattern:     "[invalid-regex(", // Invalid regex pattern
	}
	
	err := gojanus.Validate(spec)
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
	
	_, err := gojanus.ParseJSON([]byte(invalidJSON))
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
	tempDir, err := os.MkdirTemp("", "gojanus-test")
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
	_, err = gojanus.ParseFromFile(unsupportedFile)
	if err == nil {
		t.Error("Expected parsing error for unsupported file format")
	}
}

// TestParseFromJSONFile tests parsing from a JSON file
// Matches Swift: testParseFromJSONFile()
func TestParseFromJSONFile(t *testing.T) {
	// Create temporary JSON file
	tempDir, err := os.MkdirTemp("", "gojanus-test")
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
	
	spec, err := gojanus.ParseFromFile(jsonFile)
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
	tempDir, err := os.MkdirTemp("", "gojanus-test")
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
	
	spec, err := gojanus.ParseFromFile(yamlFile)
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

// TestParseMultipleFiles tests parsing and merging multiple Manifest files
func TestParseMultipleFiles(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gojanus-multifile-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create first specification file (base)
	baseFile := filepath.Join(tempDir, "base.json")
	baseData := `{
		"version": "1.0.0",
		"name": "Multi-File Test API",
		"description": "Base Manifest",
		"channels": {
			"base-channel": {
				"name": "Base Channel",
				"description": "Base channel",
				"commands": {
					"base-command": {
						"name": "Base Command",
						"description": "Base command"
					}
				}
			}
		},
		"models": {
			"BaseModel": {
				"name": "Base Model",
				"type": "object",
				"description": "Base model",
				"properties": {
					"id": {
						"name": "ID",
						"type": "string",
						"description": "Model ID",
						"required": true
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(baseFile, []byte(baseData), 0644)
	if err != nil {
		t.Fatalf("Failed to write base file: %v", err)
	}
	
	// Create second specification file (additional)
	additionalFile := filepath.Join(tempDir, "additional.json")
	additionalData := `{
		"version": "1.0.0",
		"name": "Additional Spec",
		"description": "Additional Manifest",
		"channels": {
			"additional-channel": {
				"name": "Additional Channel",
				"description": "Additional channel",
				"commands": {
					"additional-command": {
						"name": "Additional Command",
						"description": "Additional command"
					}
				}
			}
		},
		"models": {
			"AdditionalModel": {
				"name": "Additional Model",
				"type": "object",
				"description": "Additional model",
				"properties": {
					"value": {
						"name": "Value",
						"type": "string",
						"description": "Model value",
						"required": true
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(additionalFile, []byte(additionalData), 0644)
	if err != nil {
		t.Fatalf("Failed to write additional file: %v", err)
	}
	
	// Parse multiple files using parser instance method
	parser := gojanus.NewManifestParser()
	spec, err := parser.ParseMultipleFiles([]string{baseFile, additionalFile})
	if err != nil {
		t.Fatalf("Failed to parse multiple files: %v", err)
	}
	
	// Verify merged result
	if spec.Name != "Multi-File Test API" {
		t.Errorf("Expected base name 'Multi-File Test API', got '%s'", spec.Name)
	}
	
	if len(spec.Channels) != 2 {
		t.Errorf("Expected 2 channels after merge, got %d", len(spec.Channels))
	}
	
	// Verify base channel exists
	if _, exists := spec.Channels["base-channel"]; !exists {
		t.Error("Expected 'base-channel' to exist after merge")
	}
	
	// Verify additional channel exists
	if _, exists := spec.Channels["additional-channel"]; !exists {
		t.Error("Expected 'additional-channel' to exist after merge")
	}
	
	if len(spec.Models) != 2 {
		t.Errorf("Expected 2 models after merge, got %d", len(spec.Models))
	}
	
	// Verify base model exists
	if _, exists := spec.Models["BaseModel"]; !exists {
		t.Error("Expected 'BaseModel' to exist after merge")
	}
	
	// Verify additional model exists
	if _, exists := spec.Models["AdditionalModel"]; !exists {
		t.Error("Expected 'AdditionalModel' to exist after merge")
	}
}

// TestParseMultipleFilesWithConflict tests that merging fails when channels conflict
func TestParseMultipleFilesWithConflict(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "gojanus-conflict-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create first file
	baseFile := filepath.Join(tempDir, "base.json")
	baseData := `{
		"version": "1.0.0",
		"name": "Conflict Test API",
		"channels": {
			"conflict-channel": {
				"name": "Base Channel",
				"commands": {
					"test-command": {
						"name": "Test Command"
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(baseFile, []byte(baseData), 0644)
	if err != nil {
		t.Fatalf("Failed to write base file: %v", err)
	}
	
	// Create second file with conflicting channel
	conflictFile := filepath.Join(tempDir, "conflict.json")
	conflictData := `{
		"version": "1.0.0",
		"name": "Conflict Spec",
		"channels": {
			"conflict-channel": {
				"name": "Conflicting Channel",
				"commands": {
					"different-command": {
						"name": "Different Command"
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(conflictFile, []byte(conflictData), 0644)
	if err != nil {
		t.Fatalf("Failed to write conflict file: %v", err)
	}
	
	// Attempt to parse - should fail due to conflict
	parser := gojanus.NewManifestParser()
	_, err = parser.ParseMultipleFiles([]string{baseFile, conflictFile})
	if err == nil {
		t.Error("Expected error for conflicting channel names")
	}
	
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
}

// TestSpecificationMerging tests direct specification merging functionality
func TestSpecificationMerging(t *testing.T) {
	// Create base specification
	baseSpec := &gojanus.Manifest{
		Version: "1.0.0",
		Name:    "Base API",
		Channels: map[string]*gojanus.ChannelSpec{
			"base-channel": {
				Name: "Base Channel",
				Commands: map[string]*gojanus.CommandSpec{
					"base-command": {
						Name: "Base Command",
					},
				},
			},
		},
		Models: map[string]*gojanus.ModelDefinition{
			"BaseModel": {
				Name: "Base Model",
				Type: "object",
			},
		},
	}
	
	// Create additional specification
	additionalSpec := &gojanus.Manifest{
		Version: "1.0.0",
		Name:    "Additional API",
		Channels: map[string]*gojanus.ChannelSpec{
			"additional-channel": {
				Name: "Additional Channel",
				Commands: map[string]*gojanus.CommandSpec{
					"additional-command": {
						Name: "Additional Command",
					},
				},
			},
		},
		Models: map[string]*gojanus.ModelDefinition{
			"AdditionalModel": {
				Name: "Additional Model",
				Type: "object",
			},
		},
	}
	
	// Merge specifications using parser method
	parser := gojanus.NewManifestParser()
	err := parser.MergeSpecifications(baseSpec, additionalSpec)
	if err != nil {
		t.Fatalf("Failed to merge specifications: %v", err)
	}
	
	// Verify merge results
	if len(baseSpec.Channels) != 2 {
		t.Errorf("Expected 2 channels after merge, got %d", len(baseSpec.Channels))
	}
	
	if len(baseSpec.Models) != 2 {
		t.Errorf("Expected 2 models after merge, got %d", len(baseSpec.Models))
	}
	
	// Verify both channels exist
	if _, exists := baseSpec.Channels["base-channel"]; !exists {
		t.Error("Expected 'base-channel' to exist after merge")
	}
	
	if _, exists := baseSpec.Channels["additional-channel"]; !exists {
		t.Error("Expected 'additional-channel' to exist after merge")
	}
	
	// Verify both models exist
	if _, exists := baseSpec.Models["BaseModel"]; !exists {
		t.Error("Expected 'BaseModel' to exist after merge")
	}
	
	if _, exists := baseSpec.Models["AdditionalModel"]; !exists {
		t.Error("Expected 'AdditionalModel' to exist after merge")
	}
}

// TestStaticInterfaceMethods tests all static methods match instance method behavior
func TestStaticInterfaceMethods(t *testing.T) {
	// Test data
	jsonData := `{
		"version": "1.0.0",
		"name": "Static Test API",
		"channels": {
			"static-channel": {
				"name": "Static Channel",
				"commands": {
					"static-command": {
						"name": "Static Command"
					}
				}
			}
		}
	}`
	
	yamlData := `
version: "1.0.0"
name: "Static YAML API"
channels:
  static-yaml-channel:
    name: "Static YAML Channel"
    commands:
      static-yaml-command:
        name: "Static YAML Command"
`
	
	// Test static ParseJSON
	spec1, err := gojanus.ParseJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Static ParseJSON failed: %v", err)
	}
	
	if spec1.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", spec1.Name)
	}
	
	// Test static ParseJSONString  
	spec2, err := gojanus.ParseJSONString(jsonData)
	if err != nil {
		t.Fatalf("Static ParseJSONString failed: %v", err)
	}
	
	if spec2.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", spec2.Name)
	}
	
	// Test static ParseYAML
	spec3, err := gojanus.ParseYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Static ParseYAML failed: %v", err)
	}
	
	if spec3.Name != "Static YAML API" {
		t.Errorf("Expected name 'Static YAML API', got '%s'", spec3.Name)
	}
	
	// Test static ParseYAMLString
	spec4, err := gojanus.ParseYAMLString(yamlData)
	if err != nil {
		t.Fatalf("Static ParseYAMLString failed: %v", err)
	}
	
	if spec4.Name != "Static YAML API" {
		t.Errorf("Expected name 'Static YAML API', got '%s'", spec4.Name)
	}
	
	// Test static Validate
	err = gojanus.Validate(spec1)
	if err != nil {
		t.Errorf("Static Validate failed: %v", err)
	}
	
	// Create temporary file for static ParseFromFile test
	tempDir, err := os.MkdirTemp("", "gojanus-static-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	staticFile := filepath.Join(tempDir, "static.json")
	err = os.WriteFile(staticFile, []byte(jsonData), 0644)
	if err != nil {
		t.Fatalf("Failed to write static file: %v", err)
	}
	
	// Test static ParseFromFile
	spec5, err := gojanus.ParseFromFile(staticFile)
	if err != nil {
		t.Fatalf("Static ParseFromFile failed: %v", err)
	}
	
	if spec5.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", spec5.Name)
	}
}

// TestJSONYAMLSerialization tests serialization of specifications back to JSON/YAML
func TestJSONYAMLSerialization(t *testing.T) {
	// Create test specification
	spec := createValidManifest()
	
	parser := gojanus.NewManifestParser()
	
	// Test JSON serialization
	jsonBytes, err := parser.SerializeToJSON(spec)
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}
	
	if len(jsonBytes) == 0 {
		t.Error("JSON serialization produced empty result")
	}
	
	// Verify JSON is valid by parsing it back
	reparsedSpec, err := gojanus.ParseJSON(jsonBytes)
	if err != nil {
		t.Fatalf("Failed to reparse serialized JSON: %v", err)
	}
	
	if reparsedSpec.Name != spec.Name {
		t.Errorf("Expected name '%s', got '%s' after JSON round-trip", spec.Name, reparsedSpec.Name)
	}
	
	// Test YAML serialization
	yamlBytes, err := parser.SerializeToYAML(spec)
	if err != nil {
		t.Fatalf("Failed to serialize to YAML: %v", err)
	}
	
	if len(yamlBytes) == 0 {
		t.Error("YAML serialization produced empty result")
	}
	
	// Verify YAML is valid by parsing it back
	reparsedYAMLSpec, err := gojanus.ParseYAML(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to reparse serialized YAML: %v", err)
	}
	
	if reparsedYAMLSpec.Name != spec.Name {
		t.Errorf("Expected name '%s', got '%s' after YAML round-trip", spec.Name, reparsedYAMLSpec.Name)
	}
}

// Helper function to create a valid Manifest
// Matches Swift test helper patterns
func createValidManifest() *gojanus.Manifest {
	return &gojanus.Manifest{
		Version:     "1.0.0",
		Name:        "Valid Test API",
		Description: "Valid test Manifest",
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
		Models: map[string]*gojanus.ModelDefinition{
			"TestModel": {
				Name:        "Test Model",
				Type:        "object",
				Description: "Test model description",
				Properties: map[string]*gojanus.ArgumentSpec{
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