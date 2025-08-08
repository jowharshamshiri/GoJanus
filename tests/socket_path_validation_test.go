package tests

import (
	"testing"
	"GoJanus/pkg/core"
)

// TestSocketPathValidation tests socket path validation without external dependencies
func TestSocketPathValidation(t *testing.T) {
	// Test valid socket path
	validPath := "/tmp/test.sock"
	validator := core.NewSecurityValidator()
	
	err := validator.ValidateSocketPath(validPath)
	if err != nil {
		t.Errorf("Expected valid path to pass validation, got error: %v", err)
	}
}

// TestSocketPathValidationInvalidPath tests invalid socket paths
func TestSocketPathValidationInvalidPath(t *testing.T) {
	validator := core.NewSecurityValidator()
	
	// Test null byte injection
	invalidPath := "/tmp/test\x00.sock"
	err := validator.ValidateSocketPath(invalidPath)
	if err == nil {
		t.Error("Expected null byte path to fail validation")
	}
	
	// Test path traversal
	traversalPath := "/tmp/../etc/passwd"
	err = validator.ValidateSocketPath(traversalPath)
	if err == nil {
		t.Error("Expected path traversal to fail validation")
	}
}