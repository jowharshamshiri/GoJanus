// Package gojanus provides a Go implementation of Janus communication
// with 100% feature and protocol compatibility with SwiftJanus and RustJanus
//
// This package implements SOCK_DGRAM connectionless Unix domain socket communication:
// - Core Layer: Low-level Unix datagram socket communication with security validation
// - Protocol Layer: High-level API client with datagram messaging and reply-to mechanism
// - Specification Layer: Manifest parsing and validation engine
//
// Key Features:
// - Connectionless SOCK_DGRAM communication
// - Reply-to mechanism for request-response patterns
// - 25+ comprehensive security mechanisms
// - Cross-language protocol compatibility
// - Ephemeral socket patterns
// - Manifest engine (JSON/YAML)
// - Enterprise-grade configuration options
//
// Example Usage:
//
//	// Parse Manifest
//	spec, err := specification.ParseFromFile("manifest.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create client
//	client, err := protocol.NewJanusClient(
//		"/tmp/my-service.sock",
//		"library-management",
//		spec,
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Send command
//	response, err := client.SendCommand(
//		context.Background(),
//		"get-book",
//		map[string]interface{}{"id": "123"},
//		30*time.Second,
//		nil,
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Response: %+v\n", response)
package gojanus

import (
	"github.com/jowharshamshiri/GoJanus/pkg/core"
	"github.com/jowharshamshiri/GoJanus/pkg/models"
	"github.com/jowharshamshiri/GoJanus/pkg/protocol"
	"github.com/jowharshamshiri/GoJanus/pkg/specification"
)

// Version represents the library version
const Version = "1.0.0"

// Re-export main types for convenient access

// Core layer types
type (
	CoreJanusClient   = core.JanusClient
	CoreClientConfig  = core.JanusClientConfig
	SecurityValidator = core.SecurityValidator
)

// Protocol layer types (main API)
type (
	JanusClient       = protocol.JanusClient
	JanusClientConfig = protocol.JanusClientConfig
	TimeoutManager    = protocol.TimeoutManager
)

// Model types
type (
	JanusCommand     = models.JanusCommand
	JanusResponse    = models.JanusResponse
	JSONRPCError      = models.JSONRPCError
	JSONRPCErrorCode  = models.JSONRPCErrorCode
	JSONRPCErrorData  = models.JSONRPCErrorData
	SocketMessage     = models.SocketMessage
	CommandHandler    = models.CommandHandler
	TimeoutHandler    = models.TimeoutHandler
)

// Specification types
type (
	Manifest       = specification.Manifest
	ManifestParser = specification.ManifestParser
	ChannelSpec            = specification.ChannelSpec
	CommandSpec            = specification.CommandSpec
	ArgumentSpec           = specification.ArgumentSpec
	ResponseSpec           = specification.ResponseSpec
	ModelDefinition        = specification.ModelDefinition
	ValidationError        = specification.ValidationError
)

// Convenience constructors

// NewCoreClient creates a new Unix datagram client with default configuration
func NewCoreClient(socketPath string) (*CoreJanusClient, error) {
	return core.NewJanusClient(socketPath)
}

// NewCoreClientWithConfig creates a new Unix datagram client with custom configuration
func NewCoreClientWithConfig(socketPath string, config CoreClientConfig) (*CoreJanusClient, error) {
	return core.NewJanusClient(socketPath, config)
}

// NewJanusClient creates a new Janus datagram client with default configuration
func NewJanusClient(socketPath, channelID string) (*JanusClient, error) {
	return protocol.New(socketPath, channelID)
}

// NewJanusClientWithConfig creates a new Janus datagram client with custom configuration
func NewJanusClientWithConfig(socketPath, channelID string, config JanusClientConfig) (*JanusClient, error) {
	return protocol.New(socketPath, channelID, config)
}

// NewManifestParser creates a new Manifest parser
func NewManifestParser() *ManifestParser {
	return specification.NewManifestParser()
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return core.NewSecurityValidator()
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager() *TimeoutManager {
	return protocol.NewTimeoutManager()
}

// Direct function exports (no legacy wrappers)

// ParseFromFile parses an Manifest from a file
var ParseFromFile = specification.ParseFromFile

// ParseJSON parses an Manifest from JSON data  
var ParseJSON = specification.ParseJSON

// ParseYAML parses an Manifest from YAML data
var ParseYAML = specification.ParseYAML

// Validate validates an Manifest
var Validate = specification.Validate

// NewJanusCommand creates a new socket command with generated UUID
func NewJanusCommand(channelID, command string, args map[string]interface{}, timeout *float64) *JanusCommand {
	return models.NewJanusCommand(channelID, command, args, timeout)
}

// NewSuccessResponse creates a successful response for a command
func NewSuccessResponse(commandID, channelID string, result map[string]interface{}) *JanusResponse {
	return models.NewSuccessResponse(commandID, channelID, result)
}

// NewErrorResponse creates an error response for a command
func NewErrorResponse(commandID, channelID string, err *JSONRPCError) *JanusResponse {
	return models.NewErrorResponse(commandID, channelID, err)
}

// NewJSONRPCError creates a new JSON-RPC error with the specified code
func NewJSONRPCError(code JSONRPCErrorCode, details string) *JSONRPCError {
	return models.NewJSONRPCError(code, details)
}

// JSON-RPC error code constants
const (
	// Standard JSON-RPC 2.0 error codes
	ParseError           JSONRPCErrorCode = models.ParseError
	InvalidRequest       JSONRPCErrorCode = models.InvalidRequest
	MethodNotFound       JSONRPCErrorCode = models.MethodNotFound
	InvalidParams        JSONRPCErrorCode = models.InvalidParams
	InternalError        JSONRPCErrorCode = models.InternalError

	// Implementation-defined server error codes (-32000 to -32099)
	ServerError             JSONRPCErrorCode = models.ServerError
	ServiceUnavailable      JSONRPCErrorCode = models.ServiceUnavailable
	AuthenticationFailed    JSONRPCErrorCode = models.AuthenticationFailed
	RateLimitExceeded      JSONRPCErrorCode = models.RateLimitExceeded
	ResourceNotFound       JSONRPCErrorCode = models.ResourceNotFound
	ValidationFailed       JSONRPCErrorCode = models.ValidationFailed
	HandlerTimeout         JSONRPCErrorCode = models.HandlerTimeout
	SocketTransportError  JSONRPCErrorCode = models.SocketTransportError
	ConfigurationError    JSONRPCErrorCode = models.ConfigurationError
	SecurityViolation     JSONRPCErrorCode = models.SecurityViolation
	ResourceLimitExceeded JSONRPCErrorCode = models.ResourceLimitExceeded
)

// Default configuration getters

// DefaultCoreClientConfig returns the default Unix datagram client configuration
func DefaultCoreClientConfig() CoreClientConfig {
	return core.DefaultJanusClientConfig()
}

// DefaultJanusClientConfig returns the default Janus datagram client configuration
func DefaultJanusClientConfig() JanusClientConfig {
	return protocol.DefaultJanusClientConfig()
}

// Library information

// GetVersion returns the library version
func GetVersion() string {
	return Version
}

// GetSupportedFeatures returns a list of supported features
func GetSupportedFeatures() []string {
	return []string{
		"Connectionless SOCK_DGRAM Communication",
		"Reply-To Response Mechanism",
		"UUID Command Tracking",
		"Ephemeral Socket Patterns",
		"25+ Security Mechanisms",
		"Manifest Engine",
		"JSON/YAML Support",
		"Cross-Language Compatibility",
		"Natural Message Boundaries",
		"Channel Isolation",
		"Resource Monitoring",
		"Concurrent Operations",
		"Graceful Shutdown",
		"Error Recovery",
		"Path Validation",
	}
}
