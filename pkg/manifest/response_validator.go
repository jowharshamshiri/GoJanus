package manifest

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

// ResponseValidator validates request handler responses against Manifest ResponseManifest models
// Matches TypeScript ResponseValidator functionality exactly for cross-language parity
type ResponseValidator struct {
	manifest *Manifest
}

// NewResponseValidator creates a new response validator with the given Manifest
func NewResponseValidator(manifest *Manifest) *ResponseValidator {
	return &ResponseValidator{
		manifest: manifest,
	}
}

// ValidateResponse validates a response against a ResponseManifest
func (rv *ResponseValidator) ValidateResponse(response map[string]interface{}, responseManifest *ResponseManifest) *ValidationResult {
	startTime := time.Now()
	var errors []*ValidationError
	
	// Validate the response value against the manifest
	rv.validateValue(response, responseManifest, "", &errors)
	
	fieldsValidated := rv.countValidatedFields(responseManifest)
	validationTime := float64(time.Since(startTime).Nanoseconds()) / 1e6 // Convert to milliseconds
	
	return &ValidationResult{
		Valid:           len(errors) == 0,
		Errors:          errors,
		ValidationTime:  validationTime,
		FieldsValidated: fieldsValidated,
	}
}

// ValidateRequestResponse validates a request response by looking up the request manifest
func (rv *ResponseValidator) ValidateRequestResponse(response map[string]interface{}, requestName string) *ValidationResult {
	startTime := time.Now()
	
	// Channels have been removed from the protocol
	// Return a validation result indicating that validation cannot be performed
	return &ValidationResult{
		Valid: false,
		Errors: []*ValidationError{{
			Field:    "protocol",
			Message:  "Channels have been removed from the protocol",
			Expected: "channel-free protocol",
			Actual:   "validation not available",
		}},
		ValidationTime:  float64(time.Since(startTime).Nanoseconds()) / 1e6,
		FieldsValidated: 0,
	}
}

// validateValue validates a value against an argument or response manifest
func (rv *ResponseValidator) validateValue(value interface{}, manifest interface{}, fieldPath string, errors *[]*ValidationError) {
	// Handle model references
	if responseManifest, ok := manifest.(*ResponseManifest); ok && responseManifest.ModelRef != "" {
		model := rv.resolveModelReference(responseManifest.ModelRef)
		if model == nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  fmt.Sprintf("Model reference '%s' not found", responseManifest.ModelRef),
				Expected: "valid model reference",
				Actual:   responseManifest.ModelRef,
			})
			return
		}
		rv.validateValue(value, model, fieldPath, errors)
		return
	}
	
	if argManifest, ok := manifest.(*ArgumentManifest); ok && argManifest.ModelRef != "" {
		model := rv.resolveModelReference(argManifest.ModelRef)
		if model == nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  fmt.Sprintf("Model reference '%s' not found", argManifest.ModelRef),
				Expected: "valid model reference",
				Actual:   argManifest.ModelRef,
			})
			return
		}
		rv.validateValue(value, model, fieldPath, errors)
		return
	}
	
	// Get the type string
	var typeStr string
	switch s := manifest.(type) {
	case *ResponseManifest:
		typeStr = s.Type
	case *ArgumentManifest:
		typeStr = s.Type
	case *ModelDefinition:
		typeStr = s.Type
	default:
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  "Unknown manifest type",
			Expected: "ResponseManifest, ArgumentManifest, or ModelDefinition",
			Actual:   reflect.TypeOf(manifest).String(),
		})
		return
	}
	
	// Validate type
	initialErrorCount := len(*errors)
	rv.validateType(value, typeStr, fieldPath, errors)
	
	if len(*errors) > initialErrorCount {
		return // Don't continue validation if type is wrong
	}
	
	// Type-manifestific validation
	switch typeStr {
	case "string":
		if strValue, ok := value.(string); ok {
			rv.validateString(strValue, manifest, fieldPath, errors)
		}
	case "number", "integer":
		if numValue, ok := rv.getNumericValue(value); ok {
			rv.validateNumber(numValue, typeStr, manifest, fieldPath, errors)
		}
	case "array":
		if arrayValue, ok := value.([]interface{}); ok {
			rv.validateArray(arrayValue, manifest, fieldPath, errors)
		}
	case "object":
		if objValue, ok := value.(map[string]interface{}); ok {
			rv.validateObject(objValue, manifest, fieldPath, errors)
		}
	case "boolean":
		// Boolean validation is covered by type validation
	}
	
	// Validate enum values (only available on ArgumentManifest)
	if argManifest, ok := manifest.(*ArgumentManifest); ok && len(argManifest.Enum) > 0 {
		rv.validateEnum(value, argManifest.Enum, fieldPath, errors)
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
func (rv *ResponseValidator) validateString(value string, manifest interface{}, fieldPath string, errors *[]*ValidationError) {
	argManifest, ok := manifest.(*ArgumentManifest)
	if !ok {
		return // Only ArgumentManifest has string constraints
	}
	
	// Length validation
	if argManifest.MinLength != nil && len(value) < *argManifest.MinLength {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("String is too short (%d < %d)", len(value), *argManifest.MinLength),
			Expected: fmt.Sprintf("minimum length %d", *argManifest.MinLength),
			Actual:   fmt.Sprintf("length %d", len(value)),
		})
	}
	
	if argManifest.MaxLength != nil && len(value) > *argManifest.MaxLength {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("String is too long (%d > %d)", len(value), *argManifest.MaxLength),
			Expected: fmt.Sprintf("maximum length %d", *argManifest.MaxLength),
			Actual:   fmt.Sprintf("length %d", len(value)),
		})
	}
	
	// Pattern validation
	if argManifest.Pattern != "" {
		regex, err := regexp.Compile(argManifest.Pattern)
		if err != nil {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  "Invalid regex pattern in manifest",
				Expected: "valid regex pattern",
				Actual:   argManifest.Pattern,
			})
			return
		}
		
		if !regex.MatchString(value) {
			*errors = append(*errors, &ValidationError{
				Field:    fieldPath,
				Message:  "String does not match required pattern",
				Expected: fmt.Sprintf("pattern %s", argManifest.Pattern),
				Actual:   value,
			})
		}
	}
}

// validateNumber validates numeric value constraints
func (rv *ResponseValidator) validateNumber(value float64, valueType string, manifest interface{}, fieldPath string, errors *[]*ValidationError) {
	argManifest, ok := manifest.(*ArgumentManifest)
	if !ok {
		return // Only ArgumentManifest has numeric constraints
	}
	
	// Range validation
	if argManifest.Minimum != nil && value < *argManifest.Minimum {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("Number is too small (%g < %g)", value, *argManifest.Minimum),
			Expected: fmt.Sprintf("minimum %g", *argManifest.Minimum),
			Actual:   value,
		})
	}
	
	if argManifest.Maximum != nil && value > *argManifest.Maximum {
		*errors = append(*errors, &ValidationError{
			Field:    fieldPath,
			Message:  fmt.Sprintf("Number is too large (%g > %g)", value, *argManifest.Maximum),
			Expected: fmt.Sprintf("maximum %g", *argManifest.Maximum),
			Actual:   value,
		})
	}
}

// validateArray validates array value and items
func (rv *ResponseValidator) validateArray(value []interface{}, manifest interface{}, fieldPath string, errors *[]*ValidationError) {
	var itemManifest *ArgumentManifest
	
	switch s := manifest.(type) {
	case *ResponseManifest:
		itemManifest = s.Items
	case *ArgumentManifest:
		itemManifest = s.Items
	}
	
	if itemManifest == nil {
		return // No item manifest, skip item validation
	}
	
	// Validate each array item
	for i, item := range value {
		itemFieldPath := fmt.Sprintf("%s[%d]", fieldPath, i)
		rv.validateValue(item, itemManifest, itemFieldPath, errors)
	}
}

// validateObject validates object properties
func (rv *ResponseValidator) validateObject(value map[string]interface{}, manifest interface{}, fieldPath string, errors *[]*ValidationError) {
	var properties map[string]*ArgumentManifest
	
	switch s := manifest.(type) {
	case *ResponseManifest:
		properties = s.Properties
	case *ArgumentManifest:
		properties = s.Properties
	case *ModelDefinition:
		properties = s.Properties
	}
	
	if properties == nil {
		return // No property manifest, skip property validation
	}
	
	// Validate each property
	for propName, propManifest := range properties {
		var propFieldPath string
		if fieldPath == "" {
			propFieldPath = propName
		} else {
			propFieldPath = fmt.Sprintf("%s.%s", fieldPath, propName)
		}
		
		propValue, exists := value[propName]
		
		// Check required fields
		if propManifest.Required && (!exists || propValue == nil) {
			*errors = append(*errors, &ValidationError{
				Field:    propFieldPath,
				Message:  "Required field is missing or null",
				Expected: fmt.Sprintf("non-null %s", propManifest.Type),
				Actual:   propValue,
			})
			continue
		}
		
		// Skip validation for optional missing fields
		if !exists && !propManifest.Required {
			continue
		}
		
		// Validate property value
		rv.validateValue(propValue, propManifest, propFieldPath, errors)
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
	if rv.manifest.Models == nil {
		return nil
	}
	
	return rv.manifest.Models[modelRef]
}

// countValidatedFields counts the number of fields that would be validated
func (rv *ResponseValidator) countValidatedFields(manifest interface{}) int {
	switch s := manifest.(type) {
	case *ResponseManifest:
		if s.Type == "object" && s.Properties != nil {
			return len(s.Properties)
		}
	case *ArgumentManifest:
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

// CreateMissingManifestError creates a validation error for missing response manifest
func CreateMissingManifestError(channelID, requestName string) *ValidationResult {
	return &ValidationResult{
		Valid: false,
		Errors: []*ValidationError{{
			Field:    "manifest",
			Message:  fmt.Sprintf("No response manifest found for request '%s' in channel '%s'", requestName, channelID),
			Expected: "response manifest",
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