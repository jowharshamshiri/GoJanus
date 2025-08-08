package tests

import (
	"os"
	"path/filepath"
	"strings" 
	"testing"

	gojanus "GoJanus/pkg/manifest"
)

// TestParseJSONManifest tests parsing a valid JSON Manifest
// Matches Swift: testParseJSONManifest()
func TestParseJSONManifest(t *testing.T) {
	jsonData := `{
		"version": "1.0.0",
		"name": "Test API",
		"description": "Test Manifest",
		"channels": {
			"test-channel": {
				"name": "Test Channel",
				"description": "Test channel",
				"requests": {
					"test-request": {
						"name": "Test Request",
						"description": "Test request",
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
	
	manifest, err := gojanus.ParseJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse JSON manifest: %v", err)
	}
	
	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Version)
	}
	
	if manifest.Name != "Test API" {
		t.Errorf("Expected name 'Test API', got '%s'", manifest.Name)
	}
	
	if len(manifest.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(manifest.Channels))
	}
	
	channel, exists := manifest.Channels["test-channel"]
	if !exists {
		t.Fatal("Expected 'test-channel' to exist")
	}
	
	if len(channel.Requests) != 1 {
		t.Errorf("Expected 1 request, got %d", len(channel.Requests))
	}
	
	request, exists := channel.Requests["test-request"]
	if !exists {
		t.Fatal("Expected 'test-request' to exist")
	}
	
	if len(request.Args) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(request.Args))
	}
	
	arg, exists := request.Args["arg1"]
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

// TestParseYAMLManifest tests parsing a valid YAML Manifest
// Matches Swift: testParseYAMLManifest()
func TestParseYAMLManifest(t *testing.T) {
	yamlData := `
version: "1.0.0"
name: "Test API"
description: "Test Manifest"
channels:
  test-channel:
    name: "Test Channel"
    description: "Test channel"
    requests:
      test-request:
        name: "Test Request"
        description: "Test request"
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
	
	manifest, err := gojanus.ParseYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to parse YAML manifest: %v", err)
	}
	
	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Version)
	}
	
	if manifest.Name != "Test API" {
		t.Errorf("Expected name 'Test API', got '%s'", manifest.Name)
	}
	
	if len(manifest.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(manifest.Channels))
	}
	
	channel, exists := manifest.Channels["test-channel"]
	if !exists {
		t.Fatal("Expected 'test-channel' to exist")
	}
	
	request, exists := channel.Requests["test-request"]
	if !exists {
		t.Fatal("Expected 'test-request' to exist")
	}
	
	arg, exists := request.Args["arg1"]
	if !exists {
		t.Fatal("Expected 'arg1' to exist")
	}
	
	if arg.Type != "string" {
		t.Errorf("Expected argument type 'string', got '%s'", arg.Type)
	}
}

// TestValidateValidManifest tests validation of a valid manifest
// Matches Swift: testValidateValidManifest()
func TestValidateValidManifest(t *testing.T) {
	manifest := createValidManifest()
	
	err := gojanus.Validate(manifest)
	if err != nil {
		t.Errorf("Valid manifest should not produce validation error: %v", err)
	}
}

// TestValidateManifestWithEmptyVersion tests validation failure for empty version
// Matches Swift: testValidateManifestWithEmptyVersion()
func TestValidateManifestWithEmptyVersion(t *testing.T) {
	manifest := createValidManifest()
	manifest.Version = ""
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for empty version")
	}
	
	if !strings.Contains(err.Error(), "version is required") {
		t.Errorf("Expected version-related error, got: %v", err)
	}
}

// TestValidateManifestWithNoChannels tests validation failure for no channels
// Matches Swift: testValidateManifestWithNoChannels()
func TestValidateManifestWithNoChannels(t *testing.T) {
	manifest := createValidManifest()
	manifest.Channels = make(map[string]*gojanus.ChannelManifest)
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for no channels")
	}
	
	if !strings.Contains(err.Error(), "at least one channel") {
		t.Errorf("Expected channels-related error, got: %v", err)
	}
}

// TestValidateManifestWithEmptyChannelId tests validation failure for empty channel ID
// Matches Swift: testValidateManifestWithEmptyChannelId()
func TestValidateManifestWithEmptyChannelId(t *testing.T) {
	manifest := createValidManifest()
	
	// Add channel with empty ID
	channelManifest := manifest.Channels["test-channel"]
	delete(manifest.Channels, "test-channel")
	manifest.Channels[""] = channelManifest
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for empty channel ID")
	}
	
	if !strings.Contains(err.Error(), "channel ID cannot be empty") {
		t.Errorf("Expected channel ID error, got: %v", err)
	}
}

// TestValidateManifestWithNoRequests tests validation failure for no requests
// Matches Swift: testValidateManifestWithNoRequests()
func TestValidateManifestWithNoRequests(t *testing.T) {
	manifest := createValidManifest()
	manifest.Channels["test-channel"].Requests = make(map[string]*gojanus.RequestManifest)
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for no requests")
	}
	
	if !strings.Contains(err.Error(), "at least one request") {
		t.Errorf("Expected requests-related error, got: %v", err)
	}
}

// TestValidateManifestWithEmptyRequestName tests validation failure for empty request name
// Matches Swift: testValidateManifestWithEmptyRequestName()
func TestValidateManifestWithEmptyRequestName(t *testing.T) {
	manifest := createValidManifest()
	
	// Add request with empty name
	requestManifest := manifest.Channels["test-channel"].Requests["test-request"]
	delete(manifest.Channels["test-channel"].Requests, "test-request")
	manifest.Channels["test-channel"].Requests[""] = requestManifest
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for empty request name")
	}
	
	if !strings.Contains(err.Error(), "request name cannot be empty") {
		t.Errorf("Expected request name error, got: %v", err)
	}
}

// TestValidateManifestWithInvalidValidation tests validation failure for invalid argument manifest
// Matches Swift: testValidateManifestWithInvalidValidation()
func TestValidateManifestWithInvalidValidation(t *testing.T) {
	manifest := createValidManifest()
	
	// Add invalid argument manifest (empty type)
	manifest.Channels["test-channel"].Requests["test-request"].Args["invalid_arg"] = &gojanus.ArgumentManifest{
		Name:        "Invalid Argument",
		Type:        "", // Empty type should cause validation error
		Description: "Invalid argument",
		Required:    true,
	}
	
	err := gojanus.Validate(manifest)
	if err == nil {
		t.Error("Expected validation error for invalid argument manifest")
	}
	
	if !strings.Contains(err.Error(), "type is required") {
		t.Errorf("Expected type-related error, got: %v", err)
	}
}

// TestValidateManifestWithInvalidRegexPattern tests validation failure for invalid regex
// Matches Swift: testValidateManifestWithInvalidRegexPattern()
func TestValidateManifestWithInvalidRegexPattern(t *testing.T) {
	manifest := createValidManifest()
	
	// Add argument with invalid regex pattern
	manifest.Channels["test-channel"].Requests["test-request"].Args["regex_arg"] = &gojanus.ArgumentManifest{
		Name:        "Regex Argument",
		Type:        "string",
		Description: "Argument with invalid regex",
		Required:    false,
		Pattern:     "[invalid-regex(", // Invalid regex pattern
	}
	
	err := gojanus.Validate(manifest)
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
				"requests": {
					"test-request": {
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
				"requests": {
					"file-request": {
						"name": "File Request",
						"description": "Test request from file"
					}
				}
			}
		}
	}`
	
	err = os.WriteFile(jsonFile, []byte(jsonData), 0644)
	if err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}
	
	manifest, err := gojanus.ParseFromFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to parse JSON file: %v", err)
	}
	
	if manifest.Name != "File Test API" {
		t.Errorf("Expected name 'File Test API', got '%s'", manifest.Name)
	}
	
	if len(manifest.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(manifest.Channels))
	}
	
	channel, exists := manifest.Channels["file-channel"]
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
    requests:
      yaml-request:
        name: "YAML Request"
        description: "Test request from YAML file"
`
	
	err = os.WriteFile(yamlFile, []byte(yamlData), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}
	
	manifest, err := gojanus.ParseFromFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to parse YAML file: %v", err)
	}
	
	if manifest.Name != "YAML Test API" {
		t.Errorf("Expected name 'YAML Test API', got '%s'", manifest.Name)
	}
	
	if len(manifest.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(manifest.Channels))
	}
	
	channel, exists := manifest.Channels["yaml-channel"]
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
	
	// Create first manifest file (base)
	baseFile := filepath.Join(tempDir, "base.json")
	baseData := `{
		"version": "1.0.0",
		"name": "Multi-File Test API",
		"description": "Base Manifest",
		"channels": {
			"base-channel": {
				"name": "Base Channel",
				"description": "Base channel",
				"requests": {
					"base-request": {
						"name": "Base Request",
						"description": "Base request"
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
	
	// Create second manifest file (additional)
	additionalFile := filepath.Join(tempDir, "additional.json")
	additionalData := `{
		"version": "1.0.0",
		"name": "Additional Manifest",
		"description": "Additional Manifest",
		"channels": {
			"additional-channel": {
				"name": "Additional Channel",
				"description": "Additional channel",
				"requests": {
					"additional-request": {
						"name": "Additional Request",
						"description": "Additional request"
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
	manifest, err := parser.ParseMultipleFiles([]string{baseFile, additionalFile})
	if err != nil {
		t.Fatalf("Failed to parse multiple files: %v", err)
	}
	
	// Verify merged result
	if manifest.Name != "Multi-File Test API" {
		t.Errorf("Expected base name 'Multi-File Test API', got '%s'", manifest.Name)
	}
	
	if len(manifest.Channels) != 2 {
		t.Errorf("Expected 2 channels after merge, got %d", len(manifest.Channels))
	}
	
	// Verify base channel exists
	if _, exists := manifest.Channels["base-channel"]; !exists {
		t.Error("Expected 'base-channel' to exist after merge")
	}
	
	// Verify additional channel exists
	if _, exists := manifest.Channels["additional-channel"]; !exists {
		t.Error("Expected 'additional-channel' to exist after merge")
	}
	
	if len(manifest.Models) != 2 {
		t.Errorf("Expected 2 models after merge, got %d", len(manifest.Models))
	}
	
	// Verify base model exists
	if _, exists := manifest.Models["BaseModel"]; !exists {
		t.Error("Expected 'BaseModel' to exist after merge")
	}
	
	// Verify additional model exists
	if _, exists := manifest.Models["AdditionalModel"]; !exists {
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
				"requests": {
					"test-request": {
						"name": "Test Request"
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
		"name": "Conflict Manifest",
		"channels": {
			"conflict-channel": {
				"name": "Conflicting Channel",
				"requests": {
					"different-request": {
						"name": "Different Request"
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

// TestManifestMerging tests direct manifest merging functionality
func TestManifestMerging(t *testing.T) {
	// Create base manifest
	baseManifest := &gojanus.Manifest{
		Version: "1.0.0",
		Name:    "Base API",
		Channels: map[string]*gojanus.ChannelManifest{
			"base-channel": {
				Name: "Base Channel",
				Requests: map[string]*gojanus.RequestManifest{
					"base-request": {
						Name: "Base Request",
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
	
	// Create additional manifest
	additionalManifest := &gojanus.Manifest{
		Version: "1.0.0",
		Name:    "Additional API",
		Channels: map[string]*gojanus.ChannelManifest{
			"additional-channel": {
				Name: "Additional Channel",
				Requests: map[string]*gojanus.RequestManifest{
					"additional-request": {
						Name: "Additional Request",
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
	
	// Merge manifests using parser method
	parser := gojanus.NewManifestParser()
	err := parser.MergeManifests(baseManifest, additionalManifest)
	if err != nil {
		t.Fatalf("Failed to merge manifests: %v", err)
	}
	
	// Verify merge results
	if len(baseManifest.Channels) != 2 {
		t.Errorf("Expected 2 channels after merge, got %d", len(baseManifest.Channels))
	}
	
	if len(baseManifest.Models) != 2 {
		t.Errorf("Expected 2 models after merge, got %d", len(baseManifest.Models))
	}
	
	// Verify both channels exist
	if _, exists := baseManifest.Channels["base-channel"]; !exists {
		t.Error("Expected 'base-channel' to exist after merge")
	}
	
	if _, exists := baseManifest.Channels["additional-channel"]; !exists {
		t.Error("Expected 'additional-channel' to exist after merge")
	}
	
	// Verify both models exist
	if _, exists := baseManifest.Models["BaseModel"]; !exists {
		t.Error("Expected 'BaseModel' to exist after merge")
	}
	
	if _, exists := baseManifest.Models["AdditionalModel"]; !exists {
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
				"requests": {
					"static-request": {
						"name": "Static Request"
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
    requests:
      static-yaml-request:
        name: "Static YAML Request"
`
	
	// Test static ParseJSON
	manifest1, err := gojanus.ParseJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Static ParseJSON failed: %v", err)
	}
	
	if manifest1.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", manifest1.Name)
	}
	
	// Test static ParseJSONString  
	manifest2, err := gojanus.ParseJSONString(jsonData)
	if err != nil {
		t.Fatalf("Static ParseJSONString failed: %v", err)
	}
	
	if manifest2.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", manifest2.Name)
	}
	
	// Test static ParseYAML
	manifest3, err := gojanus.ParseYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Static ParseYAML failed: %v", err)
	}
	
	if manifest3.Name != "Static YAML API" {
		t.Errorf("Expected name 'Static YAML API', got '%s'", manifest3.Name)
	}
	
	// Test static ParseYAMLString
	manifest4, err := gojanus.ParseYAMLString(yamlData)
	if err != nil {
		t.Fatalf("Static ParseYAMLString failed: %v", err)
	}
	
	if manifest4.Name != "Static YAML API" {
		t.Errorf("Expected name 'Static YAML API', got '%s'", manifest4.Name)
	}
	
	// Test static Validate
	err = gojanus.Validate(manifest1)
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
	manifest5, err := gojanus.ParseFromFile(staticFile)
	if err != nil {
		t.Fatalf("Static ParseFromFile failed: %v", err)
	}
	
	if manifest5.Name != "Static Test API" {
		t.Errorf("Expected name 'Static Test API', got '%s'", manifest5.Name)
	}
}

// TestJSONYAMLSerialization tests serialization of manifests back to JSON/YAML
func TestJSONYAMLSerialization(t *testing.T) {
	// Create test manifest
	manifest := createValidManifest()
	
	parser := gojanus.NewManifestParser()
	
	// Test JSON serialization
	jsonBytes, err := parser.SerializeToJSON(manifest)
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}
	
	if len(jsonBytes) == 0 {
		t.Error("JSON serialization produced empty result")
	}
	
	// Verify JSON is valid by parsing it back
	reparsedManifest, err := gojanus.ParseJSON(jsonBytes)
	if err != nil {
		t.Fatalf("Failed to reparse serialized JSON: %v", err)
	}
	
	if reparsedManifest.Name != manifest.Name {
		t.Errorf("Expected name '%s', got '%s' after JSON round-trip", manifest.Name, reparsedManifest.Name)
	}
	
	// Test YAML serialization
	yamlBytes, err := parser.SerializeToYAML(manifest)
	if err != nil {
		t.Fatalf("Failed to serialize to YAML: %v", err)
	}
	
	if len(yamlBytes) == 0 {
		t.Error("YAML serialization produced empty result")
	}
	
	// Verify YAML is valid by parsing it back
	reparsedYAMLManifest, err := gojanus.ParseYAML(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to reparse serialized YAML: %v", err)
	}
	
	if reparsedYAMLManifest.Name != manifest.Name {
		t.Errorf("Expected name '%s', got '%s' after YAML round-trip", manifest.Name, reparsedYAMLManifest.Name)
	}
}

// Helper function to create a valid Manifest
// Matches Swift test helper patterns
func createValidManifest() *gojanus.Manifest {
	return &gojanus.Manifest{
		Version:     "1.0.0",
		Name:        "Valid Test API",
		Description: "Valid test Manifest",
		Channels: map[string]*gojanus.ChannelManifest{
			"test-channel": {
				Name:        "Test Channel",
				Description: "Test channel description",
				Requests: map[string]*gojanus.RequestManifest{
					"test-request": {
						Name:        "Test Request",
						Description: "Test request description",
						Args: map[string]*gojanus.ArgumentManifest{
							"test_arg": {
								Name:        "Test Argument",
								Type:        "string",
								Description: "Test argument description",
								Required:    true,
							},
						},
						Response: &gojanus.ResponseManifest{
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
				Properties: map[string]*gojanus.ArgumentManifest{
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