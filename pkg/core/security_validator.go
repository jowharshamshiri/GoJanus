package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// SecurityValidator implements all 25+ security mechanisms from Swift specification
// Provides defensive security for Unix socket communication
type SecurityValidator struct {
	maxSocketPathLength   int
	maxChannelNameLength  int
	maxCommandNameLength  int
	maxArgsDataSize       int
	allowedDirectories    []string
	commandNamePattern    *regexp.Regexp
	channelNamePattern    *regexp.Regexp
}

// NewSecurityValidator creates a new security validator with Swift-compatible defaults
func NewSecurityValidator() *SecurityValidator {
	// Command and channel names: alphanumeric + hyphen + underscore only
	// Matches Swift security validation exactly
	commandPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	channelPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	return &SecurityValidator{
		maxSocketPathLength:  108, // Unix socket path limit
		maxChannelNameLength: 256, // Matches Swift default
		maxCommandNameLength: 256, // Matches Swift default
		maxArgsDataSize:      5 * 1024 * 1024, // 5MB matches Swift
		allowedDirectories: []string{
			"/tmp/",
			"/var/run/",
			"/var/tmp/",
		},
		commandNamePattern: commandPattern,
		channelNamePattern: channelPattern,
	}
}

// ValidateSocketPath performs comprehensive socket path validation
// Implements all Swift path security mechanisms
func (sv *SecurityValidator) ValidateSocketPath(socketPath string) error {
	if socketPath == "" {
		return fmt.Errorf("socket path cannot be empty")
	}
	
	// Check path length limit (Unix socket limitation)
	if len(socketPath) > sv.maxSocketPathLength {
		return fmt.Errorf("socket path length %d exceeds maximum %d", len(socketPath), sv.maxSocketPathLength)
	}
	
	// Path traversal protection - matches Swift security
	if strings.Contains(socketPath, "../") {
		return fmt.Errorf("path traversal detected in socket path")
	}
	
	// Null byte injection prevention
	if strings.Contains(socketPath, "\x00") {
		return fmt.Errorf("null byte detected in socket path")
	}
	
	// Directory restriction enforcement - matches Swift allowed directories
	cleanPath := filepath.Clean(socketPath)
	allowed := false
	for _, allowedDir := range sv.allowedDirectories {
		if strings.HasPrefix(cleanPath, allowedDir) {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return fmt.Errorf("socket path must be in allowed directories: %v", sv.allowedDirectories)
	}
	
	// Additional character validation
	if !sv.isValidPathCharacters(socketPath) {
		return fmt.Errorf("socket path contains invalid characters")
	}
	
	return nil
}

// ValidateChannelID validates channel identifier for security
// Matches Swift channel validation rules exactly
func (sv *SecurityValidator) ValidateChannelID(channelID string) error {
	if channelID == "" {
		return fmt.Errorf("channel ID cannot be empty")
	}
	
	if len(channelID) > sv.maxChannelNameLength {
		return fmt.Errorf("channel ID length %d exceeds maximum %d", len(channelID), sv.maxChannelNameLength)
	}
	
	// Null byte injection prevention
	if strings.Contains(channelID, "\x00") {
		return fmt.Errorf("null byte detected in channel ID")
	}
	
	// Pattern validation: alphanumeric + hyphen + underscore only
	if !sv.channelNamePattern.MatchString(channelID) {
		return fmt.Errorf("channel ID contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}
	
	// UTF-8 validation
	if !utf8.ValidString(channelID) {
		return fmt.Errorf("channel ID contains invalid UTF-8 sequences")
	}
	
	return nil
}

// ValidateCommandName validates command name for security
// Matches Swift command validation rules exactly
func (sv *SecurityValidator) ValidateCommandName(commandName string) error {
	if commandName == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	
	if len(commandName) > sv.maxCommandNameLength {
		return fmt.Errorf("command name length %d exceeds maximum %d", len(commandName), sv.maxCommandNameLength)
	}
	
	// Null byte injection prevention
	if strings.Contains(commandName, "\x00") {
		return fmt.Errorf("null byte detected in command name")
	}
	
	// Pattern validation: alphanumeric + hyphen + underscore only
	if !sv.commandNamePattern.MatchString(commandName) {
		return fmt.Errorf("command name contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}
	
	// UTF-8 validation
	if !utf8.ValidString(commandName) {
		return fmt.Errorf("command name contains invalid UTF-8 sequences")
	}
	
	return nil
}

// ValidateMessageData validates message content for security
// Implements Swift message validation mechanisms
func (sv *SecurityValidator) ValidateMessageData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("message data cannot be empty")
	}
	
	// Check data size limit
	if len(data) > sv.maxArgsDataSize {
		return fmt.Errorf("message data size %d exceeds maximum %d", len(data), sv.maxArgsDataSize)
	}
	
	// Null byte detection
	for i, b := range data {
		if b == 0 {
			return fmt.Errorf("null byte detected at position %d", i)
		}
	}
	
	// UTF-8 validation
	if !utf8.Valid(data) {
		return fmt.Errorf("message data contains invalid UTF-8 sequences")
	}
	
	return nil
}

// ValidateJSONStructure performs basic JSON structure validation
// Matches Swift JSON validation requirements
func (sv *SecurityValidator) ValidateJSONStructure(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("JSON data cannot be empty")
	}
	
	// Basic JSON structure checks
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return fmt.Errorf("invalid JSON structure: must be an object")
	}
	
	// Check for balanced braces (basic validation)
	braceCount := 0
	inString := false
	escaped := false
	
	for _, r := range trimmed {
		if escaped {
			escaped = false
			continue
		}
		
		if r == '\\' {
			escaped = true
			continue
		}
		
		if r == '"' {
			inString = !inString
			continue
		}
		
		if !inString {
			if r == '{' {
				braceCount++
			} else if r == '}' {
				braceCount--
			}
		}
	}
	
	if braceCount != 0 {
		return fmt.Errorf("unbalanced JSON braces")
	}
	
	return nil
}

// isValidPathCharacters checks for valid path characters
// Implements additional character validation beyond basic checks
func (sv *SecurityValidator) isValidPathCharacters(path string) bool {
	// Allow alphanumeric, slash, dash, underscore, and dot
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)
	return validPattern.MatchString(path)
}

// ValidateResourceLimits validates resource usage limits
// Matches Swift resource monitoring and limits
func (sv *SecurityValidator) ValidateResourceLimits(connectionCount, handlerCount, pendingCommands int) error {
	// These limits match Swift configuration defaults
	maxConnections := 100
	maxHandlers := 500
	maxPending := 1000
	
	if connectionCount > maxConnections {
		return fmt.Errorf("connection count %d exceeds maximum %d", connectionCount, maxConnections)
	}
	
	if handlerCount > maxHandlers {
		return fmt.Errorf("handler count %d exceeds maximum %d", handlerCount, maxHandlers)
	}
	
	if pendingCommands > maxPending {
		return fmt.Errorf("pending commands %d exceeds maximum %d", pendingCommands, maxPending)
	}
	
	return nil
}

// ValidateChannelIsolation ensures commands stay within their designated channels
// Implements Swift channel security verification
func (sv *SecurityValidator) ValidateChannelIsolation(expectedChannelID, actualChannelID string) error {
	if expectedChannelID != actualChannelID {
		return fmt.Errorf("channel isolation violation: expected %s, got %s", expectedChannelID, actualChannelID)
	}
	return nil
}

// ValidateTimeout validates timeout values for security
// Prevents resource exhaustion through excessive timeouts
func (sv *SecurityValidator) ValidateTimeout(timeout float64) error {
	maxTimeout := 300.0 // 5 minutes maximum timeout
	minTimeout := 0.1   // 100ms minimum timeout
	
	if timeout < minTimeout {
		return fmt.Errorf("timeout %.2f seconds is below minimum %.2f seconds", timeout, minTimeout)
	}
	
	if timeout > maxTimeout {
		return fmt.Errorf("timeout %.2f seconds exceeds maximum %.2f seconds", timeout, maxTimeout)
	}
	
	return nil
}