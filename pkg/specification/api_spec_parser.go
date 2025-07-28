package specification

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// APISpecificationParser handles parsing of API specifications from JSON and YAML
// Matches Swift APISpecificationParser functionality exactly for cross-language compatibility
type APISpecificationParser struct{}

// NewAPISpecificationParser creates a new parser instance
func NewAPISpecificationParser() *APISpecificationParser {
	return &APISpecificationParser{}
}

// ParseJSON parses an API specification from JSON data
// Matches Swift: static func parseJSON(_ data: Data) throws -> APISpecification
func (parser *APISpecificationParser) ParseJSON(data []byte) (*APISpecification, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("JSON data cannot be empty")
	}
	
	var spec APISpecification
	
	// Use json.Decoder to handle large files and provide better error messages
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields() // Strict parsing like Swift
	
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to parse JSON API specification: %w", err)
	}
	
	// Validate the parsed specification
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("API specification validation failed: %w", err)
	}
	
	return &spec, nil
}

// ParseJSONString parses an API specification from a JSON string
// Matches Swift: static func parseJSON(_ jsonString: String) throws -> APISpecification
func (parser *APISpecificationParser) ParseJSONString(jsonString string) (*APISpecification, error) {
	if strings.TrimSpace(jsonString) == "" {
		return nil, fmt.Errorf("JSON string cannot be empty")
	}
	
	return parser.ParseJSON([]byte(jsonString))
}

// ParseYAML parses an API specification from YAML data
// Matches Swift: static func parseYAML(_ data: Data) throws -> APISpecification
func (parser *APISpecificationParser) ParseYAML(data []byte) (*APISpecification, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("YAML data cannot be empty")
	}
	
	var spec APISpecification
	
	// Parse YAML with strict mode
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Strict parsing like Swift
	
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML API specification: %w", err)
	}
	
	// Validate the parsed specification
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("API specification validation failed: %w", err)
	}
	
	return &spec, nil
}

// ParseYAMLString parses an API specification from a YAML string
// Matches Swift: static func parseYAML(_ yamlString: String) throws -> APISpecification
func (parser *APISpecificationParser) ParseYAMLString(yamlString string) (*APISpecification, error) {
	if strings.TrimSpace(yamlString) == "" {
		return nil, fmt.Errorf("YAML string cannot be empty")
	}
	
	return parser.ParseYAML([]byte(yamlString))
}

// ParseFromFile parses an API specification from a file
// Matches Swift: static func parseFromFile(at url: URL) throws -> APISpecification
func (parser *APISpecificationParser) ParseFromFile(filePath string) (*APISpecification, error) {
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
func (parser *APISpecificationParser) parseAutoDetect(data []byte) (*APISpecification, error) {
	trimmed := strings.TrimSpace(string(data))
	
	// Try JSON first (starts with '{')
	if strings.HasPrefix(trimmed, "{") {
		spec, err := parser.ParseJSON(data)
		if err == nil {
			return spec, nil
		}
	}
	
	// Try YAML
	spec, err := parser.ParseYAML(data)
	if err == nil {
		return spec, nil
	}
	
	// If both fail, return JSON error as it's more common
	return parser.ParseJSON(data)
}

// ValidateSpecification validates a parsed API specification
// Matches Swift: static func validate(_ spec: APISpecification) throws
func (parser *APISpecificationParser) ValidateSpecification(spec *APISpecification) error {
	if spec == nil {
		return fmt.Errorf("API specification cannot be nil")
	}
	
	return spec.Validate()
}

// SerializeToJSON serializes an API specification to JSON
// Useful for debugging and testing
func (parser *APISpecificationParser) SerializeToJSON(spec *APISpecification) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("API specification cannot be nil")
	}
	
	// Use indented JSON for readability
	return json.MarshalIndent(spec, "", "  ")
}

// SerializeToYAML serializes an API specification to YAML
// Useful for debugging and testing
func (parser *APISpecificationParser) SerializeToYAML(spec *APISpecification) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("API specification cannot be nil")
	}
	
	return yaml.Marshal(spec)
}

// ParseAndValidate is a convenience method that parses and validates in one step
// Provides a simple interface for common use cases
func (parser *APISpecificationParser) ParseAndValidate(data []byte, format string) (*APISpecification, error) {
	var spec *APISpecification
	var err error
	
	switch strings.ToLower(format) {
	case "json":
		spec, err = parser.ParseJSON(data)
	case "yaml", "yml":
		spec, err = parser.ParseYAML(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, yaml)", format)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Additional validation is already done in Parse methods
	return spec, nil
}

// GetSupportedFormats returns the list of supported file formats
// Useful for CLI tools and user interfaces
func (parser *APISpecificationParser) GetSupportedFormats() []string {
	return []string{"json", "yaml", "yml"}
}

// ParseMultipleFiles parses multiple specification files and merges them
// Useful for modular API specifications
func (parser *APISpecificationParser) ParseMultipleFiles(filePaths []string) (*APISpecification, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files provided")
	}
	
	// Parse first file as base
	baseSpec, err := parser.ParseFromFile(filePaths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse base file '%s': %w", filePaths[0], err)
	}
	
	// Merge additional files
	for i := 1; i < len(filePaths); i++ {
		additionalSpec, err := parser.ParseFromFile(filePaths[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse additional file '%s': %w", filePaths[i], err)
		}
		
		if err := parser.mergeSpecifications(baseSpec, additionalSpec); err != nil {
			return nil, fmt.Errorf("failed to merge file '%s': %w", filePaths[i], err)
		}
	}
	
	// Validate merged specification
	if err := baseSpec.Validate(); err != nil {
		return nil, fmt.Errorf("merged specification validation failed: %w", err)
	}
	
	return baseSpec, nil
}

// mergeSpecifications merges two API specifications
// The additional spec's channels and models are added to the base spec
func (parser *APISpecificationParser) mergeSpecifications(base, additional *APISpecification) error {
	// Merge channels
	for channelID, channel := range additional.Channels {
		if _, exists := base.Channels[channelID]; exists {
			return fmt.Errorf("channel '%s' already exists in base specification", channelID)
		}
		base.Channels[channelID] = channel
	}
	
	// Merge models
	if base.Models == nil {
		base.Models = make(map[string]*ModelDefinition)
	}
	
	for modelName, model := range additional.Models {
		if _, exists := base.Models[modelName]; exists {
			return fmt.Errorf("model '%s' already exists in base specification", modelName)
		}
		base.Models[modelName] = model
	}
	
	return nil
}

// Static methods for direct use (matching Swift static interface)

// ParseJSON is a static method for parsing JSON specifications
func ParseJSON(data []byte) (*APISpecification, error) {
	parser := NewAPISpecificationParser()
	return parser.ParseJSON(data)
}

// ParseJSONString is a static method for parsing JSON strings
func ParseJSONString(jsonString string) (*APISpecification, error) {
	parser := NewAPISpecificationParser()
	return parser.ParseJSONString(jsonString)
}

// ParseYAML is a static method for parsing YAML specifications
func ParseYAML(data []byte) (*APISpecification, error) {
	parser := NewAPISpecificationParser()
	return parser.ParseYAML(data)
}

// ParseYAMLString is a static method for parsing YAML strings
func ParseYAMLString(yamlString string) (*APISpecification, error) {
	parser := NewAPISpecificationParser()
	return parser.ParseYAMLString(yamlString)
}

// ParseFromFile is a static method for parsing files
func ParseFromFile(filePath string) (*APISpecification, error) {
	parser := NewAPISpecificationParser()
	return parser.ParseFromFile(filePath)
}

// Validate is a static method for validating specifications
func Validate(spec *APISpecification) error {
	parser := NewAPISpecificationParser()
	return parser.ValidateSpecification(spec)
}