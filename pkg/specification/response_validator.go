package specification

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)


// ValidationResult represents the result of response validation
type ValidationResult struct {
	Valid           bool               `json:"valid"`
	Errors          []*ValidationError `json:"errors"`
	ValidationTime  float64            `json:"validationTime"` // milliseconds
	FieldsValidated int                `json:"fieldsValidated"`
}

// ResponseValidator validates command handler responses against API specification ResponseSpec models
// Matches TypeScript ResponseValidator functionality exactly for cross-language parity
type ResponseValidator struct {
	specification *APISpecification
}

// NewResponseValidator creates a new response validator with the given API specification
func NewResponseValidator(spec *APISpecification) *ResponseValidator {
	return &ResponseValidator{
		specification: spec,
	}
}

// ValidateResponse validates a response against a ResponseSpec
func (rv *ResponseValidator) ValidateResponse(response map[string]interface{}, responseSpec *ResponseSpec) *ValidationResult {
	startTime := time.Now()
	var errors []*ValidationError
	
	// Validate the response value against the specification
	rv.validateValue(response, responseSpec, "", &errors)
	
	fieldsValidated := rv.countValidatedFields(responseSpec)
	validationTime := float64(time.Since(startTime).Nanoseconds()) / 1e6 // Convert to milliseconds
	
	return &ValidationResult{
		Valid:           len(errors) == 0,
		Errors:          errors,
		ValidationTime:  validationTime,
		FieldsValidated: fieldsValidated,
	}
}

// ValidateCommandResponse validates a command response by looking up the command specification
func (rv *ResponseValidator) ValidateCommandResponse(response map[string]interface{}, channelID, commandName string) *ValidationResult {
	startTime := time.Now()
	
	// Look up command specification
	channel, exists := rv.specification.Channels[channelID]
	if !exists {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:    "channelId",
				Message:  fmt.Sprintf("Channel '%s' not found in API specification", channelID),
				Expected: "valid channel ID",
				Actual:   channelID,
			}},
			ValidationTime:  float64(time.Since(startTime).Nanoseconds()) / 1e6,
			FieldsValidated: 0,
		}
	}
	
	command, exists := channel.Commands[commandName]
	if !exists {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:    "command",
				Message:  fmt.Sprintf("Command '%s' not found in channel '%s'", commandName, channelID),
				Expected: "valid command name",
				Actual:   commandName,
			}},
			ValidationTime:  float64(time.Since(startTime).Nanoseconds()) / 1e6,
			FieldsValidated: 0,
		}
	}
	
	if command.Response == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:    "response",
				Message:  fmt.Sprintf("No response specification defined for command '%s'", commandName),
				Expected: "response specification",
				Actual:   "undefined",
			}},
			ValidationTime:  float64(time.Since(startTime).Nanoseconds()) / 1e6,
			FieldsValidated: 0,
		}
	}
	
	return rv.ValidateResponse(response, command.Response)
}

// validateValue validates a value against an argument or response specification
func (rv *ResponseValidator) validateValue(value interface{}, spec interface{}, fieldPath string, errors *[]*ValidationError) {
	// Handle model references
	if responseSpec, ok := spec.(*ResponseSpec); ok && responseSpec.ModelRef != "" {
		model := rv.resolveModelReference(responseSpec.ModelRef)
		if model == nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  fmt.Sprintf("Model reference '%s' not found", responseSpec.ModelRef),
				Expected: "valid model reference",
				Actual:   responseSpec.ModelRef,
			})
			return
		}
		rv.validateValue(value, model, fieldPath, errors)
		return
	}
	
	if argSpec, ok := spec.(*ArgumentSpec); ok && argSpec.ModelRef != "" {
		model := rv.resolveModelReference(argSpec.ModelRef)
		if model == nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  fmt.Sprintf("Model reference '%s' not found", argSpec.ModelRef),
				Expected: "valid model reference",
				Actual:   argSpec.ModelRef,
			})
			return
		}
		rv.validateValue(value, model, fieldPath, errors)
		return
	}
	
	// Get the type string
	var typeStr string
	switch s := spec.(type) {
	case *ResponseSpec:
		typeStr = s.Type
	case *ArgumentSpec:
		typeStr = s.Type
	case *ModelDefinition:
		typeStr = s.Type
	default:
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  "Unknown specification type",
			Expected: "ResponseSpec, ArgumentSpec, or ModelDefinition",
			Actual:   reflect.TypeOf(spec).String(),
		})
		return
	}
	
	// Validate type
	initialErrorCount := len(*errors)
	rv.validateType(value, typeStr, fieldPath, errors)
	
	if len(*errors) > initialErrorCount {
		return // Don't continue validation if type is wrong
	}
	
	// Type-specific validation
	switch typeStr {
	case "string":
		if strValue, ok := value.(string); ok {
			rv.validateString(strValue, spec, fieldPath, errors)
		}
	case "number", "integer":
		if numValue, ok := rv.getNumericValue(value); ok {
			rv.validateNumber(numValue, typeStr, spec, fieldPath, errors)
		}
	case "array":
		if arrayValue, ok := value.([]interface{}); ok {
			rv.validateArray(arrayValue, spec, fieldPath, errors)
		}
	case "object":
		if objValue, ok := value.(map[string]interface{}); ok {
			rv.validateObject(objValue, spec, fieldPath, errors)
		}
	case "boolean":
		// Boolean validation is covered by type validation
	}
	
	// Validate enum values (only available on ArgumentSpec)
	if argSpec, ok := spec.(*ArgumentSpec); ok && len(argSpec.Enum) > 0 {
		rv.validateEnum(value, argSpec.Enum, fieldPath, errors)
	}
}

// validateType validates the type of a value
func (rv *ResponseValidator) validateType(value interface{}, expectedType, fieldPath string, errors *[]*ValidationError) {
	actualType := rv.getActualType(value)
	
	if expectedType == "integer" {
		if actualType != "number" || !rv.isInteger(value) {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  "Value is not an integer",
				Expected: "integer",
				Actual:   actualType,
			})
		}
	} else if actualType != expectedType {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  "Type mismatch",
			Expected: expectedType,
			Actual:   actualType,
		})
	}
}

// getActualType returns the actual type string of a value
func (rv *ResponseValidator) getActualType(value interface{}) string {
	if value == nil {
		return "null"
	}
	
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return reflect.TypeOf(value).String()
	}
}

// isInteger checks if a numeric value is an integer
func (rv *ResponseValidator) isInteger(value interface{}) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return v == float32(int64(v))
	case float64:
		return v == float64(int64(v))
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f == float64(int64(f))
		}
	}
	return false
}

// getNumericValue extracts a numeric value from various types
func (rv *ResponseValidator) getNumericValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// validateString validates string value constraints
func (rv *ResponseValidator) validateString(value string, spec interface{}, fieldPath string, errors *[]*ValidationError) {
	argSpec, ok := spec.(*ArgumentSpec)
	if !ok {
		return // Only ArgumentSpec has string constraints
	}
	
	// Length validation
	if argSpec.MinLength != nil && len(value) < *argSpec.MinLength {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("String is too short (%d < %d)", len(value), *argSpec.MinLength),
			Expected: fmt.Sprintf("minimum length %d", *argSpec.MinLength),
			Actual:   fmt.Sprintf("length %d", len(value)),
		})
	}
	
	if argSpec.MaxLength != nil && len(value) > *argSpec.MaxLength {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("String is too long (%d > %d)", len(value), *argSpec.MaxLength),
			Expected: fmt.Sprintf("maximum length %d", *argSpec.MaxLength),
			Actual:   fmt.Sprintf("length %d", len(value)),
		})
	}
	
	// Pattern validation
	if argSpec.Pattern != "" {
		regex, err := regexp.Compile(argSpec.Pattern)
		if err != nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  "Invalid regex pattern in specification",
				Expected: "valid regex pattern",
				Actual:   argSpec.Pattern,
			})
			return
		}
		
		if !regex.MatchString(value) {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  "String does not match required pattern",
				Expected: fmt.Sprintf("pattern %s", argSpec.Pattern),
				Actual:   value,
			})
		}
	}
}

// validateNumber validates numeric value constraints
func (rv *ResponseValidator) validateNumber(value float64, valueType string, spec interface{}, fieldPath string, errors *[]*ValidationError) {
	argSpec, ok := spec.(*ArgumentSpec)
	if !ok {
		return // Only ArgumentSpec has numeric constraints
	}
	
	// Range validation
	if argSpec.Minimum != nil && value < *argSpec.Minimum {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("Number is too small (%g < %g)", value, *argSpec.Minimum),
			Expected: fmt.Sprintf("minimum %g", *argSpec.Minimum),
			Actual:   value,
		})
	}
	
	if argSpec.Maximum != nil && value > *argSpec.Maximum {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("Number is too large (%g > %g)", value, *argSpec.Maximum),
			Expected: fmt.Sprintf("maximum %g", *argSpec.Maximum),
			Actual:   value,
		})
	}
}

// validateArray validates array value and items
func (rv *ResponseValidator) validateArray(value []interface{}, spec interface{}, fieldPath string, errors *[]*ValidationError) {
	var itemSpec *ArgumentSpec
	
	switch s := spec.(type) {
	case *ResponseSpec:
		itemSpec = s.Items
	case *ArgumentSpec:
		itemSpec = s.Items
	}
	
	if itemSpec == nil {
		return // No item specification, skip item validation
	}
	
	// Validate each array item
	for i, item := range value {
		itemFieldPath := fmt.Sprintf("%s[%d]", fieldPath, i)
		rv.validateValue(item, itemSpec, itemFieldPath, errors)
	}
}

// validateObject validates object properties
func (rv *ResponseValidator) validateObject(value map[string]interface{}, spec interface{}, fieldPath string, errors *[]*ValidationError) {
	var properties map[string]*ArgumentSpec
	
	switch s := spec.(type) {
	case *ResponseSpec:
		properties = s.Properties
	case *ArgumentSpec:
		properties = s.Properties
	case *ModelDefinition:
		properties = s.Properties
	}
	
	if properties == nil {
		return // No property specification, skip property validation
	}
	
	// Validate each property
	for propName, propSpec := range properties {
		var propFieldPath string
		if fieldPath == "" {
			propFieldPath = propName
		} else {
			propFieldPath = fmt.Sprintf("%s.%s", fieldPath, propName)
		}
		
		propValue, exists := value[propName]
		
		// Check required fields
		if propSpec.Required && (!exists || propValue == nil) {
			*errors = append(*errors, &ValidationError{
				Field:    propFieldPath,
				Message:  "Required field is missing or null",
				Expected: fmt.Sprintf("non-null %s", propSpec.Type),
				Actual:   propValue,
			})
			continue
		}
		
		// Skip validation for optional missing fields
		if !exists && !propSpec.Required {
			continue
		}
		
		// Validate property value
		rv.validateValue(propValue, propSpec, propFieldPath, errors)
	}
}

// validateEnum validates enum constraints
func (rv *ResponseValidator) validateEnum(value interface{}, enumValues []string, fieldPath string, errors *[]*ValidationError) {
	valueStr := fmt.Sprintf("%v", value)
	
	for _, enumValue := range enumValues {
		if valueStr == enumValue {
			return // Valid enum value found
		}
	}
	
	*errors = append(*errors, &ValidationError{
		Field:    fieldPath,
		Message:  "Value is not in allowed enum list",
		Expected: strings.Join(enumValues, ", "),
		Actual:   value,
	})
}

// resolveModelReference resolves a model reference to its definition
func (rv *ResponseValidator) resolveModelReference(modelRef string) *ModelDefinition {
	if rv.specification.Models == nil {
		return nil
	}
	
	return rv.specification.Models[modelRef]
}

// countValidatedFields counts the number of fields that would be validated
func (rv *ResponseValidator) countValidatedFields(spec interface{}) int {
	switch s := spec.(type) {
	case *ResponseSpec:
		if s.Type == "object" && s.Properties != nil {
			return len(s.Properties)
		}
	case *ArgumentSpec:
		if s.Type == "object" && s.Properties != nil {
			return len(s.Properties)
		}
	case *ModelDefinition:
		if s.Type == "object" && s.Properties != nil {
			return len(s.Properties)
		}
	}
	return 1
}

// CreateMissingSpecificationError creates a validation error for missing response specification
func CreateMissingSpecificationError(channelID, commandName string) *ValidationResult {
	return &ValidationResult{
		Valid: false,
		Errors: []*ValidationError{{
			Field:    "specification",
			Message:  fmt.Sprintf("No response specification found for command '%s' in channel '%s'", commandName, channelID),
			Expected: "response specification",
			Actual:   "undefined",
		}},
		ValidationTime:  0,
		FieldsValidated: 0,
	}
}

// CreateSuccessResult creates a validation result for successful validation
func CreateSuccessResult(fieldsValidated int, validationTime float64) *ValidationResult {
	return &ValidationResult{
		Valid:           true,
		Errors:          []*ValidationError{},
		ValidationTime:  validationTime,
		FieldsValidated: fieldsValidated,
	}
}