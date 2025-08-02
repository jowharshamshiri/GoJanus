package models

import (
	"encoding/json"
	"fmt"
)

// JSONRPCErrorCode represents standard JSON-RPC 2.0 error codes
type JSONRPCErrorCode int

const (
	// Standard JSON-RPC 2.0 error codes
	ParseError           JSONRPCErrorCode = -32700
	InvalidRequest       JSONRPCErrorCode = -32600
	MethodNotFound       JSONRPCErrorCode = -32601
	InvalidParams        JSONRPCErrorCode = -32602
	InternalError        JSONRPCErrorCode = -32603

	// Implementation-defined server error codes (-32000 to -32099)
	ServerError             JSONRPCErrorCode = -32000
	ServiceUnavailable      JSONRPCErrorCode = -32001
	AuthenticationFailed    JSONRPCErrorCode = -32002
	RateLimitExceeded      JSONRPCErrorCode = -32003
	ResourceNotFound       JSONRPCErrorCode = -32004
	ValidationFailed       JSONRPCErrorCode = -32005
	HandlerTimeout         JSONRPCErrorCode = -32006
	SocketTransportError  JSONRPCErrorCode = -32007
	ConfigurationError    JSONRPCErrorCode = -32008
	SecurityViolation     JSONRPCErrorCode = -32009
	ResourceLimitExceeded JSONRPCErrorCode = -32010
	
	// Janus-specific protocol error codes (-32011 to -32020)
	MessageFramingError   JSONRPCErrorCode = -32011
	ResponseTrackingError JSONRPCErrorCode = -32012
	ManifestValidationError JSONRPCErrorCode = -32013
)

// String returns the string representation of the error code
func (code JSONRPCErrorCode) String() string {
	switch code {
	case ParseError:
		return "PARSE_ERROR"
	case InvalidRequest:
		return "INVALID_REQUEST"
	case MethodNotFound:
		return "METHOD_NOT_FOUND"
	case InvalidParams:
		return "INVALID_PARAMS"
	case InternalError:
		return "INTERNAL_ERROR"
	case ServerError:
		return "SERVER_ERROR"
	case ServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case AuthenticationFailed:
		return "AUTHENTICATION_FAILED"
	case RateLimitExceeded:
		return "RATE_LIMIT_EXCEEDED"
	case ResourceNotFound:
		return "RESOURCE_NOT_FOUND"
	case ValidationFailed:
		return "VALIDATION_FAILED"
	case HandlerTimeout:
		return "HANDLER_TIMEOUT"
	case SocketTransportError:
		return "SOCKET_ERROR"
	case ConfigurationError:
		return "CONFIGURATION_ERROR"
	case SecurityViolation:
		return "SECURITY_VIOLATION"
	case ResourceLimitExceeded:
		return "RESOURCE_LIMIT_EXCEEDED"
	case MessageFramingError:
		return "MESSAGE_FRAMING_ERROR"
	case ResponseTrackingError:
		return "RESPONSE_TRACKING_ERROR"
	case ManifestValidationError:
		return "MANIFEST_VALIDATION_ERROR"
	default:
		return fmt.Sprintf("UNKNOWN_ERROR_%d", int(code))
	}
}

// Message returns the standard human-readable message for the error code
func (code JSONRPCErrorCode) Message() string {
	switch code {
	case ParseError:
		return "Parse error"
	case InvalidRequest:
		return "Invalid Request"
	case MethodNotFound:
		return "Method not found"
	case InvalidParams:
		return "Invalid params"
	case InternalError:
		return "Internal error"
	case ServerError:
		return "Server error"
	case ServiceUnavailable:
		return "Service unavailable"
	case AuthenticationFailed:
		return "Authentication failed"
	case RateLimitExceeded:
		return "Rate limit exceeded"
	case ResourceNotFound:
		return "Resource not found"
	case ValidationFailed:
		return "Validation failed"
	case HandlerTimeout:
		return "Handler timeout"
	case SocketTransportError:
		return "Socket error"
	case ConfigurationError:
		return "Configuration error"
	case SecurityViolation:
		return "Security violation"
	case ResourceLimitExceeded:
		return "Resource limit exceeded"
	case MessageFramingError:
		return "Message framing error"
	case ResponseTrackingError:
		return "Response tracking error"
	case ManifestValidationError:
		return "Manifest validation error"
	default:
		return "Unknown error"
	}
}

// JSONRPCErrorData contains additional error context information
type JSONRPCErrorData struct {
	Details     string                 `json:"details,omitempty"`
	Field       string                 `json:"field,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 compliant error
type JSONRPCError struct {
	Code    JSONRPCErrorCode  `json:"code"`
	Message string            `json:"message"`
	Data    *JSONRPCErrorData `json:"data,omitempty"`
}


// Error implements the error interface
func (e *JSONRPCError) Error() string {
	if e.Data != nil && e.Data.Details != "" {
		return fmt.Sprintf("JSON-RPC Error %d: %s - %s", int(e.Code), e.Message, e.Data.Details)
	}
	return fmt.Sprintf("JSON-RPC Error %d: %s", int(e.Code), e.Message)
}

// NewJSONRPCError creates a new JSON-RPC error with the specified code
func NewJSONRPCError(code JSONRPCErrorCode, details string) *JSONRPCError {
	error := &JSONRPCError{
		Code:    code,
		Message: code.Message(),
	}
	
	if details != "" {
		error.Data = &JSONRPCErrorData{
			Details: details,
		}
	}
	
	return error
}

// NewJSONRPCErrorWithContext creates a new JSON-RPC error with additional context
func NewJSONRPCErrorWithContext(code JSONRPCErrorCode, details string, context map[string]interface{}) *JSONRPCError {
	error := &JSONRPCError{
		Code:    code,
		Message: code.Message(),
		Data: &JSONRPCErrorData{
			Details: details,
			Context: context,
		},
	}
	
	return error
}

// NewValidationError creates a validation-specific JSON-RPC error
func NewValidationError(field string, value interface{}, details string, constraints map[string]interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    ValidationFailed,
		Message: ValidationFailed.Message(),
		Data: &JSONRPCErrorData{
			Details:     details,
			Field:       field,
			Value:       value,
			Constraints: constraints,
		},
	}
}

// MarshalJSON implements custom JSON marshaling
func (e *JSONRPCError) MarshalJSON() ([]byte, error) {
	type Alias JSONRPCError
	return json.Marshal(&struct {
		Code int `json:"code"`
		*Alias
	}{
		Code:  int(e.Code),
		Alias: (*Alias)(e),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (e *JSONRPCError) UnmarshalJSON(data []byte) error {
	type Alias JSONRPCError
	aux := &struct {
		Code int `json:"code"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	e.Code = JSONRPCErrorCode(aux.Code)
	return nil
}

// Legacy error code mapping for backward compatibility
func MapLegacyErrorCode(legacyCode string) JSONRPCErrorCode {
	switch legacyCode {
	case "UNKNOWN_COMMAND":
		return MethodNotFound
	case "VALIDATION_ERROR":
		return ValidationFailed
	case "INVALID_ARGUMENTS":
		return InvalidParams
	case "HANDLER_ERROR":
		return InternalError
	case "HANDLER_TIMEOUT":
		return HandlerTimeout
	default:
		return ServerError
	}
}