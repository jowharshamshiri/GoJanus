// Package gounixsocketapi provides a Go implementation of Unix socket API communication
// with 100% feature and protocol compatibility with SwiftUnixSockAPI and RustUnixSockAPI
//
// This package implements SOCK_DGRAM connectionless Unix domain socket communication:
// - Core Layer: Low-level Unix datagram socket communication with security validation
// - Protocol Layer: High-level API client with datagram messaging and reply-to mechanism
// - Specification Layer: API specification parsing and validation engine
//
// Key Features:
// - Connectionless SOCK_DGRAM communication
// - Reply-to mechanism for request-response patterns
// - 25+ comprehensive security mechanisms
// - Cross-language protocol compatibility
// - Ephemeral socket patterns
// - API specification engine (JSON/YAML)
// - Enterprise-grade configuration options
//
// Example Usage:
//
//	// Parse API specification
//	spec, err := specification.ParseFromFile("api-spec.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create client
//	client, err := protocol.NewUnixSockAPIClient(
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
//
package gounixsocketapi

import (
	"github.com/user/GoUnixSockAPI/pkg/core"
	"github.com/user/GoUnixSockAPI/pkg/models"
	"github.com/user/GoUnixSockAPI/pkg/protocol"
	"github.com/user/GoUnixSockAPI/pkg/specification"
)

// Version represents the library version
const Version = "1.0.0"

// Re-export main types for convenient access

// Core layer types
type (
	UnixDatagramClient         = core.UnixDatagramClient
	UnixDatagramClientConfig   = core.UnixDatagramClientConfig
	SecurityValidator          = core.SecurityValidator
)

// Protocol layer types
type (
	UnixSockAPIDatagramClient       = protocol.UnixSockAPIDatagramClient
	UnixSockAPIDatagramClientConfig = protocol.UnixSockAPIDatagramClientConfig
	TimeoutManager                  = protocol.TimeoutManager
)

// Model types
type (
	SocketCommand    = models.SocketCommand
	SocketResponse   = models.SocketResponse
	SocketError      = models.SocketError
	SocketMessage    = models.SocketMessage
	CommandHandler   = models.CommandHandler
	TimeoutHandler   = models.TimeoutHandler
)

// Specification types
type (
	APISpecification       = specification.APISpecification
	APISpecificationParser = specification.APISpecificationParser
	ChannelSpec           = specification.ChannelSpec
	CommandSpec           = specification.CommandSpec
	ArgumentSpec          = specification.ArgumentSpec
	ResponseSpec          = specification.ResponseSpec
	ModelDefinition       = specification.ModelDefinition
	ValidationError       = specification.ValidationError
)

// Convenience constructors

// NewUnixDatagramClient creates a new Unix datagram client with default configuration
func NewUnixDatagramClient(socketPath string) (*UnixDatagramClient, error) {
	return core.NewUnixDatagramClient(socketPath)
}

// NewUnixDatagramClientWithConfig creates a new Unix datagram client with custom configuration
func NewUnixDatagramClientWithConfig(socketPath string, config UnixDatagramClientConfig) (*UnixDatagramClient, error) {
	return core.NewUnixDatagramClient(socketPath, config)
}

// NewUnixSockAPIDatagramClient creates a new Unix socket API datagram client with default configuration
func NewUnixSockAPIDatagramClient(socketPath, channelID string, apiSpec *APISpecification) (*UnixSockAPIDatagramClient, error) {
	return protocol.NewUnixSockAPIDatagramClient(socketPath, channelID, apiSpec)
}

// NewUnixSockAPIDatagramClientWithConfig creates a new Unix socket API datagram client with custom configuration
func NewUnixSockAPIDatagramClientWithConfig(socketPath, channelID string, apiSpec *APISpecification, config UnixSockAPIDatagramClientConfig) (*UnixSockAPIDatagramClient, error) {
	return protocol.NewUnixSockAPIDatagramClient(socketPath, channelID, apiSpec, config)
}

// NewAPISpecificationParser creates a new API specification parser
func NewAPISpecificationParser() *APISpecificationParser {
	return specification.NewAPISpecificationParser()
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return core.NewSecurityValidator()
}


// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager() *TimeoutManager {
	return protocol.NewTimeoutManager()
}

// Convenience functions for common operations

// ParseAPISpecFromFile parses an API specification from a file
func ParseAPISpecFromFile(filePath string) (*APISpecification, error) {
	return specification.ParseFromFile(filePath)
}

// ParseAPISpecFromJSON parses an API specification from JSON data
func ParseAPISpecFromJSON(data []byte) (*APISpecification, error) {
	return specification.ParseJSON(data)
}

// ParseAPISpecFromYAML parses an API specification from YAML data
func ParseAPISpecFromYAML(data []byte) (*APISpecification, error) {
	return specification.ParseYAML(data)
}

// ValidateAPISpec validates an API specification
func ValidateAPISpec(spec *APISpecification) error {
	return specification.Validate(spec)
}

// NewSocketCommand creates a new socket command with generated UUID
func NewSocketCommand(channelID, command string, args map[string]interface{}, timeout *float64) *SocketCommand {
	return models.NewSocketCommand(channelID, command, args, timeout)
}

// NewSuccessResponse creates a successful response for a command
func NewSuccessResponse(commandID, channelID string, result map[string]interface{}) *SocketResponse {
	return models.NewSuccessResponse(commandID, channelID, result)
}

// NewErrorResponse creates an error response for a command
func NewErrorResponse(commandID, channelID string, err *SocketError) *SocketResponse {
	return models.NewErrorResponse(commandID, channelID, err)
}

// Default configuration getters

// DefaultUnixDatagramClientConfig returns the default Unix datagram client configuration
func DefaultUnixDatagramClientConfig() UnixDatagramClientConfig {
	return core.DefaultUnixDatagramClientConfig()
}

// DefaultUnixSockAPIDatagramClientConfig returns the default Unix socket API datagram client configuration
func DefaultUnixSockAPIDatagramClientConfig() UnixSockAPIDatagramClientConfig {
	return protocol.DefaultUnixSockAPIDatagramClientConfig()
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
		"API Specification Engine",
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