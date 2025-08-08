package manifest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Manifest represents the complete Manifest
// Matches Swift Manifest structure exactly for cross-language compatibility
type Manifest struct {
	Version     string                        `json:"version"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Models      map[string]*ModelDefinition   `json:"models,omitempty"`
}


// RequestManifest represents a request manifest within a channel
// Matches Swift RequestManifest structure
type RequestManifest struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Args        map[string]*ArgumentManifest  `json:"args,omitempty"`
	Response    *ResponseManifest             `json:"response,omitempty"`
	ErrorCodes  []string                  `json:"errorCodes,omitempty"`
}

// ArgumentManifest represents an argument manifest for a request
// Matches Swift ArgumentManifest structure with ResponseValidator extensions
type ArgumentManifest struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Required    bool                     `json:"required"`
	Default     interface{}              `json:"default,omitempty"`
	Pattern     string                   `json:"pattern,omitempty"`
	MinLength   *int                     `json:"minLength,omitempty"`
	MaxLength   *int                     `json:"maxLength,omitempty"`
	Minimum     *float64                 `json:"minimum,omitempty"`
	Maximum     *float64                 `json:"maximum,omitempty"`
	Enum        []string                 `json:"enum,omitempty"`
	ModelRef    string                   `json:"modelRef,omitempty"`
	Items       *ArgumentManifest            `json:"items,omitempty"`       // For array types
	Properties  map[string]*ArgumentManifest `json:"properties,omitempty"` // For object types
}

// ResponseManifest represents a response manifest for a request
// Matches Swift ResponseManifest structure with ResponseValidator extensions
type ResponseManifest struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Properties  map[string]*ArgumentManifest `json:"properties,omitempty"`
	ModelRef    string                   `json:"modelRef,omitempty"`
	Items       *ArgumentManifest            `json:"items,omitempty"` // For array response types
}

// ModelDefinition represents a reusable data model
// Matches Swift ModelDefinition structure
type ModelDefinition struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Properties  map[string]*ArgumentManifest `json:"properties,omitempty"`
	Required    []string                 `json:"required,omitempty"`
}

// ValidationError represents a validation error with context
type ValidationError struct {
	Field    string      `json:"field"`
	Message  string      `json:"message"`
	Value    interface{} `json:"value,omitempty"`    // Legacy field for backward compatibility
	Expected string      `json:"expected,omitempty"` // Expected value/type for ResponseValidator
	Actual   interface{} `json:"actual,omitempty"`   // Actual value for ResponseValidator
	Context  string      `json:"context,omitempty"`  // Additional context for ResponseValidator
}

func (ve *ValidationError) Error() string {
	if ve.Expected != "" && ve.Actual != nil {
		return fmt.Sprintf("validation error for field '%s': %s (expected: %s, actual: %v)", 
			ve.Field, ve.Message, ve.Expected, ve.Actual)
	}
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", ve.Field, ve.Message, ve.Value)
}

// HasRequest checks if a request exists in the manifest
// Channels have been removed from the protocol
func (manifest *Manifest) HasRequest(requestName string) bool {
	// Since channels are removed, this always returns false for now
	// The server will handle request validation
	return false
}

// GetRequest retrieves a request manifest
// Channels have been removed from the protocol
func (manifest *Manifest) GetRequest(requestName string) (*RequestManifest, error) {
	// Since channels are removed, this always returns an error
	// The server will handle request validation
	return nil, fmt.Errorf("channels have been removed from protocol")
}

// ValidateRequestArgs validates request arguments against the manifest
// Matches Swift comprehensive argument validation
func (manifest *Manifest) ValidateRequestArgs(requestManifest *RequestManifest, args map[string]interface{}) error {
	if requestManifest.Args == nil {
		if len(args) > 0 {
			return &ValidationError{
				Field:   "arguments",
				Message: "request does not accept arguments",
				Value:   args,
			}
		}
		return nil
	}
	
	// Check required arguments
	for argName, argManifest := range requestManifest.Args {
		if argManifest.Required {
			if _, exists := args[argName]; !exists {
				return &ValidationError{
					Field:   argName,
					Message: "required argument missing",
					Value:   nil,
				}
			}
		}
	}
	
	// Validate provided arguments
	for argName, argValue := range args {
		argManifest, exists := requestManifest.Args[argName]
		if !exists {
			return &ValidationError{
				Field:   argName,
				Message: "unknown argument",
				Value:   argValue,
			}
		}
		
		if err := manifest.validateArgument(argName, argValue, argManifest); err != nil {
			return err
		}
	}
	
	return nil
}

// validateArgument validates a single argument against its manifest
// Implements all Swift validation rules
func (manifest *Manifest) validateArgument(name string, value interface{}, argManifest *ArgumentManifest) error {
	// Handle null values
	if value == nil {
		if argManifest.Required {
			return &ValidationError{
				Field:   name,
				Message: "required argument cannot be null",
				Value:   value,
			}
		}
		return nil
	}
	
	// Type validation
	if err := manifest.validateArgumentType(name, value, argManifest); err != nil {
		return err
	}
	
	// Pattern validation for strings
	if argManifest.Pattern != "" && argManifest.Type == "string" {
		if strValue, ok := value.(string); ok {
			matched, err := regexp.MatchString(argManifest.Pattern, strValue)
			if err != nil {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("invalid pattern regex: %s", err.Error()),
					Value:   value,
				}
			}
			if !matched {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("value does not match pattern: %s", argManifest.Pattern),
					Value:   value,
				}
			}
		}
	}
	
	// Length validation for strings
	if argManifest.Type == "string" {
		if strValue, ok := value.(string); ok {
			if argManifest.MinLength != nil && len(strValue) < *argManifest.MinLength {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("string length %d is less than minimum %d", len(strValue), *argManifest.MinLength),
					Value:   value,
				}
			}
			if argManifest.MaxLength != nil && len(strValue) > *argManifest.MaxLength {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("string length %d exceeds maximum %d", len(strValue), *argManifest.MaxLength),
					Value:   value,
				}
			}
		}
	}
	
	// Numeric range validation
	if argManifest.Type == "number" || argManifest.Type == "integer" {
		if numValue, err := manifest.getNumericValue(value); err == nil {
			if argManifest.Minimum != nil && numValue < *argManifest.Minimum {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("value %f is less than minimum %f", numValue, *argManifest.Minimum),
					Value:   value,
				}
			}
			if argManifest.Maximum != nil && numValue > *argManifest.Maximum {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("value %f exceeds maximum %f", numValue, *argManifest.Maximum),
					Value:   value,
				}
			}
		}
	}
	
	// Enum validation
	if len(argManifest.Enum) > 0 {
		strValue := fmt.Sprintf("%v", value)
		valid := false
		for _, enumValue := range argManifest.Enum {
			if strValue == enumValue {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{
				Field:   name,
				Message: fmt.Sprintf("value not in allowed enum values: %v", argManifest.Enum),
				Value:   value,
			}
		}
	}
	
	// Model reference validation
	if argManifest.ModelRef != "" {
		if err := manifest.validateModelReference(name, value, argManifest.ModelRef); err != nil {
			return err
		}
	}
	
	return nil
}

// validateArgumentType validates the type of an argument
// Matches Swift type validation exactly
func (manifest *Manifest) validateArgumentType(name string, value interface{}, argManifest *ArgumentManifest) error {
	switch argManifest.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   name,
				Message: "expected string type",
				Value:   value,
			}
		}
	case "number":
		if !manifest.isNumericType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected number type",
				Value:   value,
			}
		}
	case "integer":
		if !manifest.isIntegerType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected integer type",
				Value:   value,
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:   name,
				Message: "expected boolean type",
				Value:   value,
			}
		}
	case "array":
		if !manifest.isArrayType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected array type",
				Value:   value,
			}
		}
	case "object":
		if !manifest.isObjectType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected object type",
				Value:   value,
			}
		}
	default:
		return &ValidationError{
			Field:   name,
			Message: fmt.Sprintf("unknown type: %s", argManifest.Type),
			Value:   value,
		}
	}
	
	return nil
}

// validateModelReference validates a value against a model definition
// Matches Swift model reference validation
func (manifest *Manifest) validateModelReference(name string, value interface{}, modelRef string) error {
	model, exists := manifest.Models[modelRef]
	if !exists {
		return &ValidationError{
			Field:   name,
			Message: fmt.Sprintf("model reference '%s' not found", modelRef),
			Value:   value,
		}
	}
	
	// For object types, validate properties
	if model.Type == "object" {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			return &ValidationError{
				Field:   name,
				Message: "expected object for model reference",
				Value:   value,
			}
		}
		
		// Check required properties
		for _, requiredProp := range model.Required {
			if _, exists := valueMap[requiredProp]; !exists {
				return &ValidationError{
					Field:   fmt.Sprintf("%s.%s", name, requiredProp),
					Message: "required property missing in model",
					Value:   value,
				}
			}
		}
		
		// Validate properties
		for propName, propValue := range valueMap {
			propManifest, exists := model.Properties[propName]
			if !exists {
				return &ValidationError{
					Field:   fmt.Sprintf("%s.%s", name, propName),
					Message: "unknown property in model",
					Value:   propValue,
				}
			}
			
			if err := manifest.validateArgument(fmt.Sprintf("%s.%s", name, propName), propValue, propManifest); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Helper methods for type checking

func (manifest *Manifest) isNumericType(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case json.Number:
		return true
	default:
		// Try to parse as number from string
		if str, ok := value.(string); ok {
			_, err := strconv.ParseFloat(str, 64)
			return err == nil
		}
		return false
	}
}

func (manifest *Manifest) isIntegerType(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case json.Number:
		if num, ok := value.(json.Number); ok {
			_, err := num.Int64()
			return err == nil
		}
		return false
	default:
		// Try to parse as integer from string
		if str, ok := value.(string); ok {
			_, err := strconv.ParseInt(str, 10, 64)
			return err == nil
		}
		// Check if float is actually an integer
		if manifest.isNumericType(value) {
			if numVal, err := manifest.getNumericValue(value); err == nil {
				return numVal == float64(int64(numVal))
			}
		}
		return false
	}
}

func (manifest *Manifest) isArrayType(value interface{}) bool {
	switch value.(type) {
	case []interface{}:
		return true
	default:
		return false
	}
}

func (manifest *Manifest) isObjectType(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

func (manifest *Manifest) getNumericValue(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to numeric value")
	}
}

// Validate performs comprehensive validation of the Manifest
// Matches Swift manifest validation
func (manifest *Manifest) Validate() error {
	if manifest.Version == "" {
		return fmt.Errorf("Manifest version is required")
	}
	
	if manifest.Name == "" {
		return fmt.Errorf("Manifest name is required")
	}
	
	
	// Validate model definitions
	for modelName, model := range manifest.Models {
		if modelName == "" {
			return fmt.Errorf("model name cannot be empty")
		}
		
		if model.Name == "" {
			return fmt.Errorf("model '%s' name is required", modelName)
		}
		
		if model.Type == "" {
			return fmt.Errorf("model '%s' type is required", modelName)
		}
		
		// Validate model properties
		for propName, propManifest := range model.Properties {
			if err := manifest.validateArgumentManifest(fmt.Sprintf("model.%s.%s", modelName, propName), propManifest); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateArgumentManifest validates an argument manifest itself
func (manifest *Manifest) validateArgumentManifest(context string, argManifest *ArgumentManifest) error {
	if argManifest.Type == "" {
		return fmt.Errorf("argument type is required for '%s'", context)
	}
	
	validTypes := []string{"string", "number", "integer", "boolean", "array", "object"}
	validType := false
	for _, vt := range validTypes {
		if argManifest.Type == vt {
			validType = true
			break
		}
	}
	
	if !validType {
		return fmt.Errorf("invalid argument type '%s' for '%s', must be one of: %s", argManifest.Type, context, strings.Join(validTypes, ", "))
	}
	
	// Validate pattern if provided
	if argManifest.Pattern != "" {
		if _, err := regexp.Compile(argManifest.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern for '%s': %s", context, err.Error())
		}
	}
	
	// Validate numeric constraints
	if argManifest.Minimum != nil && argManifest.Maximum != nil {
		if *argManifest.Minimum > *argManifest.Maximum {
			return fmt.Errorf("minimum value cannot be greater than maximum value for '%s'", context)
		}
	}
	
	// Validate length constraints
	if argManifest.MinLength != nil && argManifest.MaxLength != nil {
		if *argManifest.MinLength > *argManifest.MaxLength {
			return fmt.Errorf("minimum length cannot be greater than maximum length for '%s'", context)
		}
	}
	
	return nil
}