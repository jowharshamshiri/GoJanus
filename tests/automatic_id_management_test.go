package tests

import (
	"testing"
	"time"

	"GoJanus/pkg/protocol"
	"GoJanus/pkg/models"
)

func TestRequestHandleCreation(t *testing.T) {
	// Test F0194: Request ID Assignment and F0196: RequestHandle Structure
	internalID := "test-uuid-12345"
	request := "test_request"
	channel := "test_channel"
	
	handle := models.NewRequestHandle(internalID, request, channel)
	
	// Verify handle properties
	if handle.GetRequest() != request {
		t.Errorf("Expected request %s, got %s", request, handle.GetRequest())
	}
	
	if handle.GetChannel() != channel {
		t.Errorf("Expected channel %s, got %s", channel, handle.GetChannel())
	}
	
	if handle.GetInternalID() != internalID {
		t.Errorf("Expected internal ID %s, got %s", internalID, handle.GetInternalID())
	}
	
	if handle.IsCancelled() {
		t.Error("New handle should not be cancelled")
	}
	
	// Test timestamp is recent
	if time.Since(handle.GetTimestamp()) > time.Second {
		t.Error("Handle timestamp should be recent")
	}
}

func TestRequestHandleCancellation(t *testing.T) {
	// Test F0204: Request Cancellation and F0212: Request Cleanup
	handle := models.NewRequestHandle("test-id", "test_request", "test_channel")
	
	if handle.IsCancelled() {
		t.Error("New handle should not be cancelled")
	}
	
	handle.MarkCancelled()
	
	if !handle.IsCancelled() {
		t.Error("Handle should be cancelled after marking")
	}
}

func TestRequestStatusTracking(t *testing.T) {
	// Test F0202: Request Status Query
	config := protocol.DefaultJanusClientConfig()
	config.EnableValidation = false // Skip validation for test
	
	client, err := protocol.New("/tmp/test_socket", "test_channel", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Create a handle
	handle := models.NewRequestHandle("test-id", "test_request", "test_channel")
	
	// Test initial status (should be completed since not in registry)
	status := client.GetRequestStatus(handle)
	if status != models.RequestStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.RequestStatusCompleted, status)
	}
	
	// Test cancelled status
	handle.MarkCancelled()
	status = client.GetRequestStatus(handle)
	if status != models.RequestStatusCancelled {
		t.Errorf("Expected status %s, got %s", models.RequestStatusCancelled, status)
	}
}

func TestPendingRequestManagement(t *testing.T) {
	// Test F0197: Handle Creation and F0201: Request State Management
	config := protocol.DefaultJanusClientConfig()
	config.EnableValidation = false
	
	client, err := protocol.New("/tmp/test_socket", "test_channel", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Initially no pending requests
	pending := client.GetPendingRequests()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending requests, got %d", len(pending))
	}
	
	// Test cancel all with no requests
	cancelled := client.CancelAllRequests()
	if cancelled != 0 {
		t.Errorf("Expected 0 cancelled requests, got %d", cancelled)
	}
}

func TestRequestLifecycleManagement(t *testing.T) {
	// Test F0200: Request State Management and F0211: Handle Cleanup
	config := protocol.DefaultJanusClientConfig()
	config.EnableValidation = false
	
	client, err := protocol.New("/tmp/test_socket", "test_channel", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Create multiple handles to test bulk operations
	handles := []*models.RequestHandle{
		models.NewRequestHandle("id1", "cmd1", "test_channel"),
		models.NewRequestHandle("id2", "cmd2", "test_channel"),
		models.NewRequestHandle("id3", "cmd3", "test_channel"),
	}
	
	// Test that handles start as completed (not in registry)
	for i, handle := range handles {
		status := client.GetRequestStatus(handle)
		if status != models.RequestStatusCompleted {
			t.Errorf("Handle %d should start as completed, got %s", i, status)
		}
	}
	
	// Test cancellation of non-existent handle should fail
	err = client.CancelRequest(handles[0])
	if err == nil {
		t.Error("Expected error when cancelling non-existent request")
	}
}

func TestIDVisibilityControl(t *testing.T) {
	// Test F0195: ID Visibility Control - UUIDs should be hidden from normal API
	handle := models.NewRequestHandle("internal-uuid-12345", "test_request", "test_channel")
	
	// User should only see request and channel, not internal UUID through normal API
	if handle.GetRequest() != "test_request" {
		t.Errorf("Expected request test_request, got %s", handle.GetRequest())
	}
	
	if handle.GetChannel() != "test_channel" {
		t.Errorf("Expected channel test_channel, got %s", handle.GetChannel())
	}
	
	// Internal ID should only be accessible for internal operations
	if handle.GetInternalID() != "internal-uuid-12345" {
		t.Errorf("Expected internal ID internal-uuid-12345, got %s", handle.GetInternalID())
	}
}

func TestRequestStatusConstants(t *testing.T) {
	// Test all RequestStatus constants are defined
	statuses := []models.RequestStatus{
		models.RequestStatusPending,
		models.RequestStatusCompleted,
		models.RequestStatusFailed,
		models.RequestStatusCancelled,
		models.RequestStatusTimeout,
	}
	
	expectedValues := []string{"pending", "completed", "failed", "cancelled", "timeout"}
	
	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Expected status %s, got %s", expectedValues[i], string(status))
		}
	}
}