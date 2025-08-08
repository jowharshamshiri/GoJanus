package core

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// SecurityValidator implements all 25+ security mechanisms from Swift manifest
// Provides defensive security for Unix socket communication
type SecurityValidator struct {
	maxSocketPathLength   int
	maxChannelNameLength  int
	maxRequestNameLength  int
	maxArgsDataSize       int
	allowedDirectories    []string
	requestNamePattern    *regexp.Regexp
	channelNamePattern    *regexp.Regexp
}

// NewSecurityValidator creates a new security validator with Swift-compatible defaults
func NewSecurityValidator() *SecurityValidator {
	// Request and channel names: alphanumeric + hyphen + underscore only
	// Matches Swift security validation exactly
	requestPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	channelPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	return &SecurityValidator{
		maxSocketPathLength:  108, // Unix socket path limit
		maxChannelNameLength: 256, // Matches Swift default
		maxRequestNameLength: 256, // Matches Swift default
		maxArgsDataSize:      5 * 1024 * 1024, // 5MB matches Swift
		allowedDirectories: []string{
			"/tmp/",
			"/var/run/",
			"/var/tmp/",
		},
		requestNamePattern: requestPattern,
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

// ValidateRequestName validates request name for security
// Matches Swift request validation rules exactly
func (sv *SecurityValidator) ValidateRequestName(requestName string) error {
	if requestName == "" {
		return fmt.Errorf("request name cannot be empty")
	}
	
	if len(requestName) > sv.maxRequestNameLength {
		return fmt.Errorf("request name length %d exceeds maximum %d", len(requestName), sv.maxRequestNameLength)
	}
	
	// Null byte injection prevention
	if strings.Contains(requestName, "\x00") {
		return fmt.Errorf("null byte detected in request name")
	}
	
	// Pattern validation: alphanumeric + hyphen + underscore only
	if !sv.requestNamePattern.MatchString(requestName) {
		return fmt.Errorf("request name contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	}
	
	// UTF-8 validation
	if !utf8.ValidString(requestName) {
		return fmt.Errorf("request name contains invalid UTF-8 sequences")
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
func (sv *SecurityValidator) ValidateResourceLimits(connectionCount, handlerCount, pendingRequests int) error {
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
	
	if pendingRequests > maxPending {
		return fmt.Errorf("pending requests %d exceeds maximum %d", pendingRequests, maxPending)
	}
	
	return nil
}

// ValidateChannelIsolation ensures requests stay within their designated channels
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

// ValidateStringContent validates string content for UTF-8 and null bytes (matches Swift implementation)
func (sv *SecurityValidator) ValidateStringContent(str string) error {
	// Check for null bytes
	if strings.Contains(str, "\000") {
		return fmt.Errorf("string contains null bytes")
	}
	
	// Validate UTF-8 encoding
	if !utf8.ValidString(str) {
		return fmt.Errorf("string contains invalid UTF-8")
	}
	
	return nil
}

// ValidateRequestId validates request ID format and security (matches Swift implementation)
func (sv *SecurityValidator) ValidateRequestId(requestId string) error {
	if requestId == "" {
		return fmt.Errorf("request ID cannot be empty")
	}
	
	if len(requestId) > 64 {
		return fmt.Errorf("request ID exceeds maximum length of 64 characters")
	}
	
	// Must be alphanumeric or UUID-like format (matches Swift pattern)
	uuidPattern := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !uuidPattern.MatchString(requestId) {
		return fmt.Errorf("request ID contains invalid characters")
	}
	
	return nil
}

// ValidateTimestamp validates timestamp for reasonableness (matches Swift implementation)
func (sv *SecurityValidator) ValidateTimestamp(timestamp float64) error {
	now := float64(time.Now().Unix())
	timeDiff := timestamp - now
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	
	// Allow 5-minute clock skew (300 seconds)
	if timeDiff > 300 {
		return fmt.Errorf("request timestamp is too far from current time")
	}
	
	return nil
}

// ValidateReservedChannelName validates channel names aren't reserved (matches Swift implementation)
func (sv *SecurityValidator) ValidateReservedChannelName(channelName string) error {
	reservedChannels := []string{"system", "admin", "root", "test"}
	lowerChannelName := strings.ToLower(channelName)
	
	for _, reserved := range reservedChannels {
		if lowerChannelName == reserved {
			return fmt.Errorf("channel name '%s' is reserved", channelName)
		}
	}
	
	return nil
}

// ValidateDangerousRequestName validates request names don't contain dangerous patterns (matches Swift implementation)
func (sv *SecurityValidator) ValidateDangerousRequestName(requestName string) error {
	dangerousPatterns := []string{"eval", "exec", "system", "shell", "rm", "delete", "drop"}
	lowerRequestName := strings.ToLower(requestName)
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerRequestName, pattern) {
			return fmt.Errorf("request name contains dangerous pattern: %s", pattern)
		}
	}
	
	return nil
}

// ValidateArgumentValue validates argument values for injection attempts (matches Swift implementation)
func (sv *SecurityValidator) ValidateArgumentValue(key string, value interface{}) error {
	// Only validate string values for injection patterns
	stringValue, ok := value.(string)
	if !ok {
		return nil
	}
	
	lowerValue := strings.ToLower(stringValue)
	
	// Check for SQL injection patterns
	sqlPatterns := []string{"'", "\"", "--", "/*", "*/", "union", "select", "drop", "delete", "insert", "update"}
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerValue, pattern) {
			return fmt.Errorf("argument '%s' contains potentially dangerous pattern: %s", key, pattern)
		}
	}
	
	// Check for script injection patterns
	scriptPatterns := []string{"<script", "javascript:", "vbscript:", "onload=", "onerror="}
	for _, pattern := range scriptPatterns {
		if strings.Contains(lowerValue, pattern) {
			return fmt.Errorf("argument '%s' contains script injection pattern: %s", key, pattern)
		}
	}
	
	return nil
}

// ValidateDangerousArgumentName validates argument names aren't dangerous (matches Swift implementation)
func (sv *SecurityValidator) ValidateDangerousArgumentName(argName string) error {
	dangerousArgs := []string{"__proto__", "constructor", "prototype", "eval", "function"}
	lowerArgName := strings.ToLower(argName)
	
	for _, dangerous := range dangerousArgs {
		if lowerArgName == dangerous {
			return fmt.Errorf("dangerous argument name: %s", argName)
		}
	}
	
	return nil
}

// ValidateSocketPathForResourceLimits validates socket path for resource exhaustion (matches Swift implementation)
func (sv *SecurityValidator) ValidateSocketPathForResourceLimits(path string) error {
	// Check for patterns that might cause resource exhaustion
	components := strings.Split(path, "/")
	
	// Too many path components could indicate an attack
	if len(components) > 10 {
		return fmt.Errorf("socket path has too many components")
	}
	
	// Check for excessively long component names
	for _, component := range components {
		if len(component) > 50 {
			return fmt.Errorf("socket path component exceeds maximum length")
		}
	}
	
	return nil
}

// ValidateEnhancedJSONStructure validates JSON structure more robustly (matches Swift implementation)
func (sv *SecurityValidator) ValidateEnhancedJSONStructure(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	
	// Parse JSON to ensure it's valid
	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		return fmt.Errorf("message contains invalid JSON structure: %v", err)
	}
	
	// Ensure it's a JSON object (map), not just any JSON
	if _, ok := jsonObj.(map[string]interface{}); !ok {
		return fmt.Errorf("message must be a JSON object")
	}
	
	return nil
}

// ValidateUUIDFormat validates request ID UUID format (matches TypeScript implementation)
func (sv *SecurityValidator) ValidateUUIDFormat(uuid string) error {
	// UUID v4 format: 8-4-4-4-12 hexadecimal digits
	uuidPattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, err := regexp.MatchString(uuidPattern, uuid)
	if err != nil {
		return fmt.Errorf("UUID pattern validation failed: %v", err)
	}
	if !matched {
		return fmt.Errorf("invalid UUID format: %s", uuid)
	}
	return nil
}

// ValidateTimestampFormat validates ISO 8601 timestamp format (matches Swift/TypeScript implementation)
func (sv *SecurityValidator) ValidateTimestampFormat(timestamp string) error {
	// ISO 8601 format validation
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, timestamp); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("invalid timestamp format: %s (expected ISO 8601)", timestamp)
}

// ValidateReservedChannels checks for reserved channel names (matches Swift implementation)
func (sv *SecurityValidator) ValidateReservedChannels(channelId string) error {
	reservedChannels := map[string]bool{
		"system":      true,
		"admin":       true,
		"root":        true,
		"internal":    true,
		"__proto__":   true,
		"constructor": true,
	}
	
	if reservedChannels[strings.ToLower(channelId)] {
		return fmt.Errorf("channel ID '%s' is reserved and cannot be used", channelId)
	}
	return nil
}

// ValidateDangerousRequest checks for dangerous request patterns (matches Swift implementation)
func (sv *SecurityValidator) ValidateDangerousRequest(requestName string) error {
	dangerousPatterns := []string{"eval", "exec", "system", "shell", "rm", "delete", "drop"}
	lowerRequest := strings.ToLower(requestName)
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerRequest, pattern) {
			return fmt.Errorf("request name contains dangerous pattern: %s", pattern)
		}
	}
	return nil
}

// ValidateArgumentSecurity checks for dangerous argument names (matches Swift implementation)
func (sv *SecurityValidator) ValidateArgumentSecurity(args map[string]interface{}) error {
	dangerousArgs := map[string]bool{
		"__proto__":    true,
		"constructor":  true,
		"prototype":    true,
		"eval":         true,
		"function":     true,
	}
	
	for argName := range args {
		if dangerousArgs[strings.ToLower(argName)] {
			return fmt.Errorf("dangerous argument name: %s", argName)
		}
	}
	
	// Validate argument values for injection attempts
	for key, value := range args {
		if err := sv.validateArgumentValue(key, value); err != nil {
			return err
		}
	}
	
	return nil
}

// validateArgumentValue checks for SQL and script injection patterns (matches Swift implementation)
func (sv *SecurityValidator) validateArgumentValue(key string, value interface{}) error {
	if stringValue, ok := value.(string); ok {
		lowerValue := strings.ToLower(stringValue)
		
		// Check for SQL injection patterns
		sqlPatterns := []string{"'", "\"", "--", "/*", "*/", "union", "select", "drop", "delete", "insert", "update"}
		for _, pattern := range sqlPatterns {
			if strings.Contains(lowerValue, pattern) {
				return fmt.Errorf("argument '%s' contains potentially dangerous SQL pattern: %s", key, pattern)
			}
		}
		
		// Check for script injection patterns
		scriptPatterns := []string{"<script", "javascript:", "vbscript:", "onload=", "onerror="}
		for _, pattern := range scriptPatterns {
			if strings.Contains(lowerValue, pattern) {
				return fmt.Errorf("argument '%s' contains script injection pattern: %s", key, pattern)
			}
		}
	}
	return nil
}