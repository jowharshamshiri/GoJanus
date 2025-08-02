package specification

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
	Channels    map[string]*ChannelSpec       `json:"channels"`
	Models      map[string]*ModelDefinition   `json:"models,omitempty"`
}

// ChannelSpec represents a communication channel specification
// Matches Swift ChannelSpec structure
type ChannelSpec struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Commands    map[string]*CommandSpec   `json:"commands"`
}

// CommandSpec represents a command specification within a channel
// Matches Swift CommandSpec structure
type CommandSpec struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Args        map[string]*ArgumentSpec  `json:"args,omitempty"`
	Response    *ResponseSpec             `json:"response,omitempty"`
	ErrorCodes  []string                  `json:"errorCodes,omitempty"`
}

// ArgumentSpec represents an argument specification for a command
// Matches Swift ArgumentSpec structure with ResponseValidator extensions
type ArgumentSpec struct {
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
	Items       *ArgumentSpec            `json:"items,omitempty"`       // For array types
	Properties  map[string]*ArgumentSpec `json:"properties,omitempty"` // For object types
}

// ResponseSpec represents a response specification for a command
// Matches Swift ResponseSpec structure with ResponseValidator extensions
type ResponseSpec struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Properties  map[string]*ArgumentSpec `json:"properties,omitempty"`
	ModelRef    string                   `json:"modelRef,omitempty"`
	Items       *ArgumentSpec            `json:"items,omitempty"` // For array response types
}

// ModelDefinition represents a reusable data model
// Matches Swift ModelDefinition structure
type ModelDefinition struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Properties  map[string]*ArgumentSpec `json:"properties,omitempty"`
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

// HasCommand checks if a command exists in the specified channel
// Matches Swift Manifest query methods
func (spec *Manifest) HasCommand(channelID, commandName string) bool {
	if spec == nil || spec.Channels == nil {
		return false
	}
	
	channel, exists := spec.Channels[channelID]
	if !exists {
		return false
	}
	
	if channel.Commands == nil {
		return false
	}
	
	_, exists = channel.Commands[commandName]
	return exists
}

// GetCommand retrieves a command specification
// Matches Swift command retrieval functionality
func (spec *Manifest) GetCommand(channelID, commandName string) (*CommandSpec, error) {
	channel, exists := spec.Channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel '%s' not found in Manifest", channelID)
	}
	
	command, exists := channel.Commands[commandName]
	if !exists {
		return nil, fmt.Errorf("command '%s' not found in channel '%s'", commandName, channelID)
	}
	
	return command, nil
}

// ValidateCommandArgs validates command arguments against the specification
// Matches Swift comprehensive argument validation
func (spec *Manifest) ValidateCommandArgs(commandSpec *CommandSpec, args map[string]interface{}) error {
	if commandSpec.Args == nil {
		if len(args) > 0 {
			return &ValidationError{
				Field:   "arguments",
				Message: "command does not accept arguments",
				Value:   args,
			}
		}
		return nil
	}
	
	// Check required arguments
	for argName, argSpec := range commandSpec.Args {
		if argSpec.Required {
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
		argSpec, exists := commandSpec.Args[argName]
		if !exists {
			return &ValidationError{
				Field:   argName,
				Message: "unknown argument",
				Value:   argValue,
			}
		}
		
		if err := spec.validateArgument(argName, argValue, argSpec); err != nil {
			return err
		}
	}
	
	return nil
}

// validateArgument validates a single argument against its specification
// Implements all Swift validation rules
func (spec *Manifest) validateArgument(name string, value interface{}, argSpec *ArgumentSpec) error {
	// Handle null values
	if value == nil {
		if argSpec.Required {
			return &ValidationError{
				Field:   name,
				Message: "required argument cannot be null",
				Value:   value,
			}
		}
		return nil
	}
	
	// Type validation
	if err := spec.validateArgumentType(name, value, argSpec); err != nil {
		return err
	}
	
	// Pattern validation for strings
	if argSpec.Pattern != "" && argSpec.Type == "string" {
		if strValue, ok := value.(string); ok {
			matched, err := regexp.MatchString(argSpec.Pattern, strValue)
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
					Message: fmt.Sprintf("value does not match pattern: %s", argSpec.Pattern),
					Value:   value,
				}
			}
		}
	}
	
	// Length validation for strings
	if argSpec.Type == "string" {
		if strValue, ok := value.(string); ok {
			if argSpec.MinLength != nil && len(strValue) < *argSpec.MinLength {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("string length %d is less than minimum %d", len(strValue), *argSpec.MinLength),
					Value:   value,
				}
			}
			if argSpec.MaxLength != nil && len(strValue) > *argSpec.MaxLength {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("string length %d exceeds maximum %d", len(strValue), *argSpec.MaxLength),
					Value:   value,
				}
			}
		}
	}
	
	// Numeric range validation
	if argSpec.Type == "number" || argSpec.Type == "integer" {
		if numValue, err := spec.getNumericValue(value); err == nil {
			if argSpec.Minimum != nil && numValue < *argSpec.Minimum {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("value %f is less than minimum %f", numValue, *argSpec.Minimum),
					Value:   value,
				}
			}
			if argSpec.Maximum != nil && numValue > *argSpec.Maximum {
				return &ValidationError{
					Field:   name,
					Message: fmt.Sprintf("value %f exceeds maximum %f", numValue, *argSpec.Maximum),
					Value:   value,
				}
			}
		}
	}
	
	// Enum validation
	if len(argSpec.Enum) > 0 {
		strValue := fmt.Sprintf("%v", value)
		valid := false
		for _, enumValue := range argSpec.Enum {
			if strValue == enumValue {
				valid = true
				break
			}
		}
		if !valid {
			return &ValidationError{
				Field:   name,
				Message: fmt.Sprintf("value not in allowed enum values: %v", argSpec.Enum),
				Value:   value,
			}
		}
	}
	
	// Model reference validation
	if argSpec.ModelRef != "" {
		if err := spec.validateModelReference(name, value, argSpec.ModelRef); err != nil {
			return err
		}
	}
	
	return nil
}

// validateArgumentType validates the type of an argument
// Matches Swift type validation exactly
func (spec *Manifest) validateArgumentType(name string, value interface{}, argSpec *ArgumentSpec) error {
	switch argSpec.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   name,
				Message: "expected string type",
				Value:   value,
			}
		}
	case "number":
		if !spec.isNumericType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected number type",
				Value:   value,
			}
		}
	case "integer":
		if !spec.isIntegerType(value) {
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
		if !spec.isArrayType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected array type",
				Value:   value,
			}
		}
	case "object":
		if !spec.isObjectType(value) {
			return &ValidationError{
				Field:   name,
				Message: "expected object type",
				Value:   value,
			}
		}
	default:
		return &ValidationError{
			Field:   name,
			Message: fmt.Sprintf("unknown type: %s", argSpec.Type),
			Value:   value,
		}
	}
	
	return nil
}

// validateModelReference validates a value against a model definition
// Matches Swift model reference validation
func (spec *Manifest) validateModelReference(name string, value interface{}, modelRef string) error {
	model, exists := spec.Models[modelRef]
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
			propSpec, exists := model.Properties[propName]
			if !exists {
				return &ValidationError{
					Field:   fmt.Sprintf("%s.%s", name, propName),
					Message: "unknown property in model",
					Value:   propValue,
				}
			}
			
			if err := spec.validateArgument(fmt.Sprintf("%s.%s", name, propName), propValue, propSpec); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Helper methods for type checking

func (spec *Manifest) isNumericType(value interface{}) bool {
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

func (spec *Manifest) isIntegerType(value interface{}) bool {
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
		if spec.isNumericType(value) {
			if numVal, err := spec.getNumericValue(value); err == nil {
				return numVal == float64(int64(numVal))
			}
		}
		return false
	}
}

func (spec *Manifest) isArrayType(value interface{}) bool {
	switch value.(type) {
	case []interface{}:
		return true
	default:
		return false
	}
}

func (spec *Manifest) isObjectType(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

func (spec *Manifest) getNumericValue(value interface{}) (float64, error) {
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
// Matches Swift specification validation
func (spec *Manifest) Validate() error {
	if spec.Version == "" {
		return fmt.Errorf("Manifest version is required")
	}
	
	if spec.Name == "" {
		return fmt.Errorf("Manifest name is required")
	}
	
	if len(spec.Channels) == 0 {
		return fmt.Errorf("Manifest must have at least one channel")
	}
	
	// Validate channels
	for channelID, channel := range spec.Channels {
		if channelID == "" {
			return fmt.Errorf("channel ID cannot be empty")
		}
		
		if channel.Name == "" {
			return fmt.Errorf("channel '%s' name is required", channelID)
		}
		
		if len(channel.Commands) == 0 {
			return fmt.Errorf("channel '%s' must have at least one command", channelID)
		}
		
		// Validate commands
		for commandName, command := range channel.Commands {
			if commandName == "" {
				return fmt.Errorf("command name cannot be empty in channel '%s'", channelID)
			}
			
			if command.Name == "" {
				return fmt.Errorf("command '%s' name is required in channel '%s'", commandName, channelID)
			}
			
			// Validate argument specifications
			for argName, argSpec := range command.Args {
				if err := spec.validateArgumentSpec(fmt.Sprintf("%s.%s.%s", channelID, commandName, argName), argSpec); err != nil {
					return err
				}
			}
		}
	}
	
	// Validate model definitions
	for modelName, model := range spec.Models {
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
		for propName, propSpec := range model.Properties {
			if err := spec.validateArgumentSpec(fmt.Sprintf("model.%s.%s", modelName, propName), propSpec); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateArgumentSpec validates an argument specification itself
func (spec *Manifest) validateArgumentSpec(context string, argSpec *ArgumentSpec) error {
	if argSpec.Type == "" {
		return fmt.Errorf("argument type is required for '%s'", context)
	}
	
	validTypes := []string{"string", "number", "integer", "boolean", "array", "object"}
	validType := false
	for _, vt := range validTypes {
		if argSpec.Type == vt {
			validType = true
			break
		}
	}
	
	if !validType {
		return fmt.Errorf("invalid argument type '%s' for '%s', must be one of: %s", argSpec.Type, context, strings.Join(validTypes, ", "))
	}
	
	// Validate pattern if provided
	if argSpec.Pattern != "" {
		if _, err := regexp.Compile(argSpec.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern for '%s': %s", context, err.Error())
		}
	}
	
	// Validate numeric constraints
	if argSpec.Minimum != nil && argSpec.Maximum != nil {
		if *argSpec.Minimum > *argSpec.Maximum {
			return fmt.Errorf("minimum value cannot be greater than maximum value for '%s'", context)
		}
	}
	
	// Validate length constraints
	if argSpec.MinLength != nil && argSpec.MaxLength != nil {
		if *argSpec.MinLength > *argSpec.MaxLength {
			return fmt.Errorf("minimum length cannot be greater than maximum length for '%s'", context)
		}
	}
	
	return nil
}