// Package gounixsocketapi provides a Go implementation of Unix socket API communication
// with 100% feature and protocol compatibility with SwiftUnixSocketAPI and RustUnixSocketAPI
//
// This package implements the complete three-layer architecture:
// - Core Layer: Low-level Unix socket communication with security validation
// - Protocol Layer: High-level API client with command handling and timeout management
// - Specification Layer: API specification parsing and validation engine
//
// Key Features:
// - Stateless communication with UUID tracking
// - 25+ comprehensive security mechanisms
// - Cross-language protocol compatibility
// - Connection pooling and resource management
// - Bilateral timeout management
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
	"github.com/user/GoUnixSocketAPI/pkg/core"
	"github.com/user/GoUnixSocketAPI/pkg/models"
	"github.com/user/GoUnixSocketAPI/pkg/protocol"
	"github.com/user/GoUnixSocketAPI/pkg/specification"
)

// Version represents the library version
const Version = "1.0.0"

// Re-export main types for convenient access

// Core layer types
type (
	UnixSocketClient         = core.UnixSocketClient
	UnixSocketClientConfig   = core.UnixSocketClientConfig
	ConnectionPool           = core.ConnectionPool
	ConnectionPoolConfig     = core.ConnectionPoolConfig
	SecurityValidator        = core.SecurityValidator
	MessageFraming          = core.MessageFraming
)

// Protocol layer types
type (
	UnixSockAPIClient       = protocol.UnixSockAPIClient
	UnixSockAPIClientConfig = protocol.UnixSockAPIClientConfig
	TimeoutManager          = protocol.TimeoutManager
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

// NewUnixSocketClient creates a new Unix socket client with default configuration
func NewUnixSocketClient(socketPath string) (*UnixSocketClient, error) {
	return core.NewUnixSocketClient(socketPath)
}

// NewUnixSocketClientWithConfig creates a new Unix socket client with custom configuration
func NewUnixSocketClientWithConfig(socketPath string, config UnixSocketClientConfig) (*UnixSocketClient, error) {
	return core.NewUnixSocketClient(socketPath, config)
}

// NewConnectionPool creates a new connection pool with default configuration
func NewConnectionPool(socketPath string) (*ConnectionPool, error) {
	return core.NewConnectionPool(socketPath)
}

// NewConnectionPoolWithConfig creates a new connection pool with custom configuration
func NewConnectionPoolWithConfig(socketPath string, config ConnectionPoolConfig) (*ConnectionPool, error) {
	return core.NewConnectionPool(socketPath, config)
}

// NewUnixSockAPIClient creates a new Unix socket API client with default configuration
func NewUnixSockAPIClient(socketPath, channelID string, apiSpec *APISpecification) (*UnixSockAPIClient, error) {
	return protocol.NewUnixSockAPIClient(socketPath, channelID, apiSpec)
}

// NewUnixSockAPIClientWithConfig creates a new Unix socket API client with custom configuration
func NewUnixSockAPIClientWithConfig(socketPath, channelID string, apiSpec *APISpecification, config UnixSockAPIClientConfig) (*UnixSockAPIClient, error) {
	return protocol.NewUnixSockAPIClient(socketPath, channelID, apiSpec, config)
}

// NewAPISpecificationParser creates a new API specification parser
func NewAPISpecificationParser() *APISpecificationParser {
	return specification.NewAPISpecificationParser()
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return core.NewSecurityValidator()
}

// NewMessageFraming creates a new message framing handler
func NewMessageFraming(maxMessageSize int) *MessageFraming {
	return core.NewMessageFraming(maxMessageSize)
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

// DefaultUnixSocketClientConfig returns the default Unix socket client configuration
func DefaultUnixSocketClientConfig() UnixSocketClientConfig {
	return core.DefaultUnixSocketClientConfig()
}

// DefaultConnectionPoolConfig returns the default connection pool configuration
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return core.DefaultConnectionPoolConfig()
}

// DefaultUnixSockAPIClientConfig returns the default Unix socket API client configuration
func DefaultUnixSockAPIClientConfig() UnixSockAPIClientConfig {
	return protocol.DefaultUnixSockAPIClientConfig()
}

// Library information

// GetVersion returns the library version
func GetVersion() string {
	return Version
}

// GetSupportedFeatures returns a list of supported features
func GetSupportedFeatures() []string {
	return []string{
		"Stateless Communication",
		"UUID Command Tracking",
		"Bilateral Timeout Management",
		"Connection Pooling",
		"25+ Security Mechanisms",
		"API Specification Engine",
		"JSON/YAML Support",
		"Cross-Language Compatibility",
		"Message Framing Protocol",
		"Channel Isolation",
		"Resource Monitoring",
		"Concurrent Operations",
		"Graceful Shutdown",
		"Error Recovery",
		"Path Validation",
	}
}