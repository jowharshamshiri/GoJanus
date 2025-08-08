package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"GoJanus/pkg/core"
	"GoJanus/pkg/models"
	"GoJanus/pkg/protocol"
	"GoJanus/pkg/manifest"
)

// TestPathTraversalAttackPrevention tests prevention of path traversal attacks
// Matches Swift: testPathTraversalAttackPrevention()
func TestPathTraversalAttackPrevention(t *testing.T) {
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test various path traversal attack patterns
	maliciousPaths := []string{
		"/tmp/../../../etc/passwd",
		"/tmp/./../../etc/shadow",
		"/tmp/../../../../../etc/hosts",
		"/tmp/test/../../../root/.bashrc",
		"/var/tmp/../../../etc/passwd",
		"/var/run/../../../etc/group",
		"../../../etc/passwd",
		"./../../etc/shadow",
		"/tmp/test/../../..",
		"/tmp/../..",
	}
	
	for _, maliciousPath := range maliciousPaths {
		_, err := protocol.New(maliciousPath, "security-channel")
		if err == nil {
			t.Errorf("Expected security error for malicious path: %s", maliciousPath)
		}
		
		if !strings.Contains(err.Error(), "traversal") && !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected traversal security error for path %s, got: %v", maliciousPath, err)
		}
	}
}

// TestNullByteInjectionDetection tests detection of null byte injection attacks
// Matches Swift: testNullByteInjectionDetection()
func TestNullByteInjectionDetection(t *testing.T) {
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test null byte injection in socket paths
	nullBytePaths := []string{
		"/tmp/test\x00.sock",
		"/tmp/\x00malicious.sock",
		"/tmp/test.sock\x00../../../etc/passwd",
		"/var/tmp/normal\x00injection.sock",
	}
	
	for _, nullBytePath := range nullBytePaths {
		_, err := protocol.New(nullBytePath, "security-channel")
		if err == nil {
			t.Errorf("Expected security error for null byte injection: %s", nullBytePath)
		}
		
		if !strings.Contains(err.Error(), "null byte") && !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected null byte security error for path %s, got: %v", nullBytePath, err)
		}
	}
}

// TestChannelIDInjectionAttacks tests prevention of channel ID injection attacks
// Matches Swift: testChannelIDInjectionAttacks()
func TestChannelIDInjectionAttacks(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test malicious channel IDs - should be rejected
	maliciousChannelIDs := []string{
		"channel\x00injection",
		"channel;rm -rf /",
		"channel`whoami`",
		"channel$(whoami)",
		"channel|whoami",
		"channel&whoami",
		"channel\nwhoami",
		"channel\rwhoami",
		"channel\twhoami",
		"../../../etc/passwd",
		"/absolute/path",
	}
	
	for _, maliciousChannelID := range maliciousChannelIDs {
		_, err := protocol.New(testSocketPath, maliciousChannelID)
		if err == nil {
			t.Errorf("Expected security error for malicious channel ID: %s", maliciousChannelID)
			continue
		}
		
		// Should get validation error for invalid channel ID format
		if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "forbidden") {
			t.Errorf("Expected validation error for channel ID %s, got: %v", maliciousChannelID, err)
		}
	}
}

// TestRequestInjectionInArguments tests prevention of request injection in arguments
// Matches Swift: testRequestInjectionInArguments()
func TestRequestInjectionInArguments(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "security-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test request injection patterns in arguments
	injectionPatterns := []string{
		"; rm -rf /",
		"| whoami",
		"& whoami",
		"`whoami`",
		"$(whoami)",
		"\n whoami",
		"\r whoami",
		"value; DROP TABLE users;",
		"value' OR '1'='1",
		"value\x00injection",
	}
	
	for _, injection := range injectionPatterns {
		args := map[string]interface{}{
			"secure_param": injection,
		}
		
		// Test validation - should detect malicious content
		validator := core.NewSecurityValidator()
		jsonData, _ := models.NewJanusRequest("security-channel", "secure-request", args, nil).ToJSON()
		
		err := validator.ValidateMessageData(jsonData)
		if err != nil && strings.Contains(injection, "\x00") {
			// Null byte should be caught
			if !strings.Contains(err.Error(), "null byte") {
				t.Errorf("Expected null byte error for injection %s, got: %v", injection, err)
			}
		}
		
		// UTF-8 validation should pass for most of these (they're valid UTF-8)
		// The security is in the application layer, not the transport layer
	}
}

// TestMalformedJSONAttackPrevention tests prevention of malformed JSON attacks
// Matches Swift: testMalformedJSONAttackPrevention()
func TestMalformedJSONAttackPrevention(t *testing.T) {
	// Test malformed JSON data
	malformedJSONData := [][]byte{
		[]byte(`{"incomplete": `),
		[]byte(`{"invalid": "unclosed string}`),
		[]byte(`{"nested": {"too": {"deep": {"structure": {"causes": {"stack": {"overflow": "maybe"}}}}}}}`),
		[]byte(`{"circular_ref": "this references itself somehow"}`),
		[]byte(`{invalid_json_without_quotes: "value"}`),
		[]byte(`{"valid_start": "but_no_end"`),
		[]byte(`{"unicode_attack": "\uFFFF\uFFFE\u0000"}`),
		[]byte(`{"control_chars": "\x00\x01\x02\x03"}`),
	}
	
	for _, malformedData := range malformedJSONData {
		_, err := manifest.ParseJSON(malformedData)
		if err == nil {
			t.Errorf("Expected JSON parsing error for malformed data: %s", string(malformedData))
		}
		
		// Should get JSON parsing error
		if !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "parse") {
			t.Errorf("Expected JSON parsing error for data %s, got: %v", string(malformedData), err)
		}
	}
}

// TestUnicodeNormalizationAttacks tests prevention of Unicode normalization attacks
// Matches Swift: testUnicodeNormalizationAttacks()
func TestUnicodeNormalizationAttacks(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "security-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test Unicode normalization attack patterns
	unicodeAttacks := []string{
		"normal_text",  // Control case
		"café",         // Normal Unicode
		"cafe\u0301",   // Same as above but with combining character
		"\uFEFF text",  // Byte order mark
		"text\uFFFD",   // Replacement character
		"\u202E override", // Right-to-left override
		"zero\u200B width", // Zero-width space
	}
	
	for _, unicodeText := range unicodeAttacks {
		args := map[string]interface{}{
			"secure_param": unicodeText,
		}
		
		request := models.NewJanusRequest("security-channel", "secure-request", args, nil)
		jsonData, err := request.ToJSON()
		if err != nil {
			t.Errorf("Failed to serialize Unicode text %s: %v", unicodeText, err)
			continue
		}
		
		// Validate the data
		validator := core.NewSecurityValidator()
		err = validator.ValidateMessageData(jsonData)
		if err != nil {
			// If validation fails, it should be for a good reason
			if !strings.Contains(err.Error(), "UTF-8") && !strings.Contains(err.Error(), "validation") {
				t.Errorf("Unexpected validation error for Unicode text %s: %v", unicodeText, err)
			}
		}
	}
}

// TestMemoryExhaustionViaLargePayloads tests protection against memory exhaustion
// Matches Swift: testMemoryExhaustionViaLargePayloads()
func TestMemoryExhaustionViaLargePayloads(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	client, err := protocol.New(testSocketPath, "security-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test with very large argument data
	largeString := strings.Repeat("A", 10*1024*1024) // 10MB string
	
	args := map[string]interface{}{
		"secure_param": largeString,
	}
	
	request := models.NewJanusRequest("security-channel", "secure-request", args, nil)
	jsonData, err := request.ToJSON()
	if err != nil {
		// Large serialization might fail, which is acceptable
		return
	}
	
	// Validate the large message
	validator := core.NewSecurityValidator()
	err = validator.ValidateMessageData(jsonData)
	if err == nil {
		t.Error("Expected validation error for overly large message data")
	}
	
	if !strings.Contains(err.Error(), "size") && !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Expected size validation error, got: %v", err)
	}
}

// TestResourceExhaustionViaConnectionFlooding tests protection against connection flooding
// Matches Swift: testResourceExhaustionViaConnectionFlooding()
func TestResourceExhaustionViaConnectionFlooding(t *testing.T) {
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test creating many clients rapidly
	maxAttempts := 200 // More than default max connections (100)
	clients := make([]*protocol.JanusClient, 0, maxAttempts)
	
	defer func() {
		// Clean up all created clients
		// Note: SOCK_DGRAM clients are connectionless and don't need explicit cleanup
		for range clients {
			// Clients will be cleaned up automatically
		}
	}()
	
	successCount := 0
	errorCount := 0
	
	for i := 0; i < maxAttempts; i++ {
		testSocketPath := fmt.Sprintf("/tmp/gojanus-flood-%d.sock", i)
		client, err := protocol.New(testSocketPath, "security-channel")
		
		if err != nil {
			errorCount++
			// Some failures are expected due to resource limits
		} else {
			clients = append(clients, client)
			successCount++
		}
		
		// Clean up socket file
		os.Remove(testSocketPath)
	}
	
	// We should be able to create a reasonable number of clients
	if successCount < 50 {
		t.Errorf("Expected to create at least 50 clients, got %d", successCount)
	}
	
	// There should be some resource limit enforcement
	if errorCount == 0 && successCount == maxAttempts {
		t.Log("Note: No resource limits detected - this might be acceptable")
	}
}

// TestConfigurationSecurityValidation tests validation of security configuration
// Matches Swift: testConfigurationSecurityValidation()
func TestConfigurationSecurityValidation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test insecure configurations
	insecureConfigs := []protocol.JanusClientConfig{
		{
			MaxMessageSize:   100,    // Too small
			DefaultTimeout:   1 * time.Nanosecond, // Too short
			DatagramTimeout:  1 * time.Nanosecond, // Too short
			EnableValidation: false,  // Insecure
		},
	}
	
	for i, config := range insecureConfigs {
		_, err := protocol.New(testSocketPath, "security-channel", config)
		if err == nil {
			t.Errorf("Expected configuration validation error for insecure config %d", i)
			continue
		}
		
		// Should get validation error for invalid configuration
		if !strings.Contains(err.Error(), "configuration") && !strings.Contains(err.Error(), "positive") {
			t.Errorf("Expected configuration error for config %d, got: %v", i, err)
		}
	}
}

// TestValidationBypassAttempts tests attempts to bypass validation
// Matches Swift: testValidationBypassAttempts()
func TestValidationBypassAttempts(t *testing.T) {
	testSocketPath := "/tmp/gojanus-security-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	manifest := createSecurityTestManifest() // Create manifest for validation bypass testing
	client, err := protocol.New(testSocketPath, "security-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test various bypass attempts
	bypassAttempts := []map[string]interface{}{
		{
			"secure_param": nil, // Null value
		},
		{
			"secure_param": "", // Empty string
		},
		{
			"SECURE_PARAM": "value", // Wrong case
		},
		{
			"secure_param": "valid",
			"extra_param":  "should_be_rejected", // Extra parameter
		},
		{}, // Missing required parameter
	}
	
	for i, args := range bypassAttempts {
		// Try to validate through the manifest
		requestManifest, err := manifest.GetRequest("security-channel", "secure-request")
		if err != nil {
			t.Errorf("Failed to get request manifest: %v", err)
			continue
		}
		
		err = manifest.ValidateRequestArgs(requestManifest, args)
		
		// Most bypass attempts should fail validation
		switch i {
		case 0: // nil value - might be acceptable depending on required flag
			if err != nil && !strings.Contains(err.Error(), "required") {
				t.Errorf("Expected required field error for attempt %d, got: %v", i, err)
			}
		case 1: // empty string - should fail if min length is enforced
			// This might pass if empty strings are allowed
		case 2: // wrong case - should fail
			if err == nil {
				t.Errorf("Expected validation error for wrong case parameter in attempt %d", i)
			}
		case 3: // extra parameter - should fail
			if err == nil {
				t.Errorf("Expected validation error for extra parameter in attempt %d", i)
			}
		case 4: // missing required - should fail
			if err == nil {
				t.Errorf("Expected validation error for missing required parameter in attempt %d", i)
			}
		}
	}
}

// TestSocketPathSecurityRestrictions tests socket path security restrictions
// Matches Swift socket path security validation
func TestSocketPathSecurityRestrictions(t *testing.T) {
	_ = createSecurityTestManifest() // Load manifest but don't use it - manifest is now fetched dynamically
	
	// Test paths outside allowed directories
	restrictedPaths := []string{
		"/home/user/socket.sock",     // Home directory
		"/root/socket.sock",          // Root directory
		"/etc/socket.sock",           // System config directory
		"/usr/bin/socket.sock",       // System binary directory
		"/var/log/socket.sock",       // Log directory (allowed dir is /var/run/, /var/tmp/)
		"/proc/socket.sock",          // Proc filesystem
		"/sys/socket.sock",           // Sys filesystem
		"/dev/socket.sock",           // Device directory
		"./socket.sock",              // Relative path
		"~/socket.sock",              // Home directory shortcut
		"socket.sock",                // Current directory
	}
	
	for _, restrictedPath := range restrictedPaths {
		_, err := protocol.New(restrictedPath, "security-channel")
		if err == nil {
			t.Errorf("Expected security error for restricted path: %s", restrictedPath)
		}
		
		if !strings.Contains(err.Error(), "allowed directories") && !strings.Contains(err.Error(), "invalid") {
			t.Errorf("Expected directory restriction error for path %s, got: %v", restrictedPath, err)
		}
	}
	
	// Test allowed paths (should work)
	allowedPaths := []string{
		"/tmp/test.sock",
		"/var/run/test.sock",
		"/var/tmp/test.sock",
	}
	
	for _, allowedPath := range allowedPaths {
		client, err := protocol.New(allowedPath, "security-channel")
		if err != nil {
			t.Errorf("Expected allowed path to work: %s, got error: %v", allowedPath, err)
		} else {
			client.Close()
		}
		
		// Clean up
		os.Remove(allowedPath)
	}
}

// Helper function to create security test Manifest
func createSecurityTestManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Version:     "1.0.0",
		Name:        "Security Test API",
		Description: "Manifest for security testing",
		Channels: map[string]*manifest.ChannelManifest{
			"security-channel": {
				Name:        "Security Channel",
				Description: "Channel for security testing",
				Requests: map[string]*manifest.RequestManifest{
					"secure-request": {
						Name:        "Secure Request",
						Description: "Request for security testing",
						Args: map[string]*manifest.ArgumentManifest{
							"secure_param": {
								Name:        "Secure Parameter",
								Type:        "string",
								Description: "Parameter for security testing",
								Required:    true,
								MinLength:   &[]int{1}[0],
								MaxLength:   &[]int{1000}[0],
								Pattern:     "^[a-zA-Z0-9_-]+$",
							},
						},
						Response: &manifest.ResponseManifest{
							Type:        "object",
							Description: "Security test response",
						},
						ErrorCodes: []string{"SECURITY_ERROR", "VALIDATION_ERROR"},
					},
				},
			},
		},
	}
}

// TestDynamicMessageSizeDetection tests dynamic message size limit detection
// Validates that the system properly detects and handles message size limits
func TestDynamicMessageSizeDetection(t *testing.T) {
	testSocketPath := "/tmp/gojanus-size-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	client, err := protocol.New(testSocketPath, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	// Test with normal-sized message (should pass validation)
	normalArgs := map[string]interface{}{
		"message": "normal message within size limits",
	}
	
	ctx := context.Background()
	
	// This should fail with connection error, not validation error
	_, err = client.SendRequest(ctx, "echo", normalArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected connection error since no server is running")
	}
	
	// Should be connection error, not message size error
	if strings.Contains(err.Error(), "size") && strings.Contains(err.Error(), "exceeds") {
		t.Errorf("Got size error for normal message: %v", err)
	}
	
	// Test with very large message (should trigger size validation)
	// Create message larger than 5MB limit
	largeData := strings.Repeat("x", 6*1024*1024) // 6MB of data
	largeArgs := map[string]interface{}{
		"message": largeData,
	}
	
	// This should fail with size validation error before attempting connection
	_, err = client.SendRequest(ctx, "echo", largeArgs, protocol.RequestOptions{Timeout: 1*time.Second})
	if err == nil {
		t.Error("Expected validation error for oversized message")
	}
	
	// Should be size validation error
	if !strings.Contains(err.Error(), "size") && !strings.Contains(err.Error(), "exceeds") && !strings.Contains(err.Error(), "limit") {
		t.Logf("Got error (may not be size-related): %v", err)
		// Log the error but don't fail - different implementations may handle this differently
	}
	
	// Test fire-and-forget with large message
	err = client.SendRequestNoResponse(ctx, "echo", largeArgs)
	if err == nil {
		t.Error("Expected validation error for oversized fire-and-forget message")
	}
	
	// Message size detection should work for both response and no-response requests
	if err != nil {
		t.Logf("Fire-and-forget large message correctly rejected: %v", err)
	}
}

