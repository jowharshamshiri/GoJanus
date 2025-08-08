package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManifestParser handles parsing of Manifests from JSON and YAML
// Matches Swift ManifestParser functionality exactly for cross-language compatibility
type ManifestParser struct{}

// NewManifestParser creates a new parser instance
func NewManifestParser() *ManifestParser {
	return &ManifestParser{}
}

// ParseJSON parses an Manifest from JSON data
// Matches Swift: static func parseJSON(_ data: Data) throws -> Manifest
func (parser *ManifestParser) ParseJSON(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("JSON data cannot be empty")
	}
	
	var manifest Manifest
	
	// Use json.Decoder to handle large files and provide better error messages
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields() // Strict parsing like Swift
	
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse JSON Manifest: %w", err)
	}
	
	// Validate the parsed manifest
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("Manifest validation failed: %w", err)
	}
	
	return &manifest, nil
}

// ParseJSONString parses an Manifest from a JSON string
// Matches Swift: static func parseJSON(_ jsonString: String) throws -> Manifest
func (parser *ManifestParser) ParseJSONString(jsonString string) (*Manifest, error) {
	if strings.TrimSpace(jsonString) == "" {
		return nil, fmt.Errorf("JSON string cannot be empty")
	}
	
	return parser.ParseJSON([]byte(jsonString))
}

// ParseYAML parses an Manifest from YAML data
// Matches Swift: static func parseYAML(_ data: Data) throws -> Manifest
func (parser *ManifestParser) ParseYAML(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("YAML data cannot be empty")
	}
	
	var manifest Manifest
	
	// Parse YAML with strict mode
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Strict parsing like Swift
	
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse YAML Manifest: %w", err)
	}
	
	// Validate the parsed manifest
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("Manifest validation failed: %w", err)
	}
	
	return &manifest, nil
}

// ParseYAMLString parses an Manifest from a YAML string
// Matches Swift: static func parseYAML(_ yamlString: String) throws -> Manifest
func (parser *ManifestParser) ParseYAMLString(yamlString string) (*Manifest, error) {
	if strings.TrimSpace(yamlString) == "" {
		return nil, fmt.Errorf("YAML string cannot be empty")
	}
	
	return parser.ParseYAML([]byte(yamlString))
}

// ParseFromFile parses an Manifest from a file
// Matches Swift: static func parseFromFile(at url: URL) throws -> Manifest
func (parser *ManifestParser) ParseFromFile(filePath string) (*Manifest, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()
	
	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}
	
	// Determine format based on file extension
	if strings.HasSuffix(strings.ToLower(filePath), ".yaml") || strings.HasSuffix(strings.ToLower(filePath), ".yml") {
		return parser.ParseYAML(data)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".json") {
		return parser.ParseJSON(data)
	} else {
		// Try to auto-detect format based on content
		return parser.parseAutoDetect(data)
	}
}

// parseAutoDetect attempts to parse data by auto-detecting the format
func (parser *ManifestParser) parseAutoDetect(data []byte) (*Manifest, error) {
	trimmed := strings.TrimSpace(string(data))
	
	// Try JSON first (starts with '{')
	if strings.HasPrefix(trimmed, "{") {
		manifest, err := parser.ParseJSON(data)
		if err == nil {
			return manifest, nil
		}
	}
	
	// Try YAML
	manifest, err := parser.ParseYAML(data)
	if err == nil {
		return manifest, nil
	}
	
	// If both fail, return JSON error as it's more common
	return parser.ParseJSON(data)
}

// ValidateManifest validates a parsed Manifest
// Matches Swift: static func validate(_ manifest: Manifest) throws
func (parser *ManifestParser) ValidateManifest(manifest *Manifest) error {
	if manifest == nil {
		return fmt.Errorf("Manifest cannot be nil")
	}
	
	return manifest.Validate()
}

// SerializeToJSON serializes an Manifest to JSON
// Useful for debugging and testing
func (parser *ManifestParser) SerializeToJSON(manifest *Manifest) ([]byte, error) {
	if manifest == nil {
		return nil, fmt.Errorf("Manifest cannot be nil")
	}
	
	// Use indented JSON for readability
	return json.MarshalIndent(manifest, "", "  ")
}

// SerializeToYAML serializes an Manifest to YAML
// Useful for debugging and testing
func (parser *ManifestParser) SerializeToYAML(manifest *Manifest) ([]byte, error) {
	if manifest == nil {
		return nil, fmt.Errorf("Manifest cannot be nil")
	}
	
	return yaml.Marshal(manifest)
}

// ParseAndValidate is a convenience method that parses and validates in one step
// Provides a simple interface for common use cases
func (parser *ManifestParser) ParseAndValidate(data []byte, format string) (*Manifest, error) {
	var manifest *Manifest
	var err error
	
	switch strings.ToLower(format) {
	case "json":
		manifest, err = parser.ParseJSON(data)
	case "yaml", "yml":
		manifest, err = parser.ParseYAML(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, yaml)", format)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Additional validation is already done in Parse methods
	return manifest, nil
}

// GetSupportedFormats returns the list of supported file formats
// Useful for CLI tools and user interfaces
func (parser *ManifestParser) GetSupportedFormats() []string {
	return []string{"json", "yaml", "yml"}
}

// ParseMultipleFiles parses multiple manifest files and merges them
// Useful for modular Manifests
func (parser *ManifestParser) ParseMultipleFiles(filePaths []string) (*Manifest, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files provided")
	}
	
	// Parse first file as base
	baseManifest, err := parser.ParseFromFile(filePaths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse base file '%s': %w", filePaths[0], err)
	}
	
	// Merge additional files
	for i := 1; i < len(filePaths); i++ {
		additionalManifest, err := parser.ParseFromFile(filePaths[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse additional file '%s': %w", filePaths[i], err)
		}
		
		if err := parser.mergeManifests(baseManifest, additionalManifest); err != nil {
			return nil, fmt.Errorf("failed to merge file '%s': %w", filePaths[i], err)
		}
	}
	
	// Validate merged manifest
	if err := baseManifest.Validate(); err != nil {
		return nil, fmt.Errorf("merged manifest validation failed: %w", err)
	}
	
	return baseManifest, nil
}

// MergeManifests merges two Manifests (public method)
// The additional manifest's channels and models are added to the base manifest
func (parser *ManifestParser) MergeManifests(base, additional *Manifest) error {
	return parser.mergeManifests(base, additional)
}

// mergeManifests merges two Manifests (private implementation)
// Only models are merged now that channels have been removed from the protocol
func (parser *ManifestParser) mergeManifests(base, additional *Manifest) error {
	// Merge models
	if base.Models == nil {
		base.Models = make(map[string]*ModelDefinition)
	}
	
	for modelName, model := range additional.Models {
		if _, exists := base.Models[modelName]; exists {
			return fmt.Errorf("model '%s' already exists in base manifest", modelName)
		}
		base.Models[modelName] = model
	}
	
	return nil
}

// Static methods for direct use (matching Swift static interface)

// ParseJSON is a static method for parsing JSON manifests
func ParseJSON(data []byte) (*Manifest, error) {
	parser := NewManifestParser()
	return parser.ParseJSON(data)
}

// ParseJSONString is a static method for parsing JSON strings
func ParseJSONString(jsonString string) (*Manifest, error) {
	parser := NewManifestParser()
	return parser.ParseJSONString(jsonString)
}

// ParseYAML is a static method for parsing YAML manifests
func ParseYAML(data []byte) (*Manifest, error) {
	parser := NewManifestParser()
	return parser.ParseYAML(data)
}

// ParseYAMLString is a static method for parsing YAML strings
func ParseYAMLString(yamlString string) (*Manifest, error) {
	parser := NewManifestParser()
	return parser.ParseYAMLString(yamlString)
}

// ParseFromFile is a static method for parsing files
func ParseFromFile(filePath string) (*Manifest, error) {
	parser := NewManifestParser()
	return parser.ParseFromFile(filePath)
}

// Validate is a static method for validating manifests
func Validate(manifest *Manifest) error {
	parser := NewManifestParser()
	return parser.ValidateManifest(manifest)
}