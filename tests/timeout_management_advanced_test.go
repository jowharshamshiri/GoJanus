package tests

import (
	"fmt"
	"sync/atomic"
	"testing" 
	"time"
	
	"GoJanus/pkg/protocol"
)

// TestTimeoutExtension tests the timeout extension capability
// Matches Swift/TypeScript timeout extension feature
func TestTimeoutExtension(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	var callbackCalled int32

	// Register a timeout for 100ms
	callback := func() {
		atomic.StoreInt32(&callbackCalled, 1)
	}

	requestID := "test-extend-request"
	manager.RegisterTimeout(requestID, 100*time.Millisecond, callback)

	// Wait 50ms, then extend by 100ms
	time.Sleep(50 * time.Millisecond)
	extended := manager.ExtendTimeout(requestID, 100*time.Millisecond)

	if !extended {
		t.Error("Expected timeout extension to succeed")
	}

	// Wait another 100ms (should not fire yet since we extended)
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callbackCalled) != 0 {
		t.Error("Callback should not have fired yet after extension")
	}

	// Wait for the extended timeout to fire
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callbackCalled) != 1 {
		t.Error("Callback should have fired after extended timeout")
	}

	// Test extending non-existent timeout
	nonExistentExtended := manager.ExtendTimeout("non-existent", 100*time.Millisecond)
	if nonExistentExtended {
		t.Error("Expected extension of non-existent timeout to fail")
	}
}

// TestErrorHandledRegistration tests timeout registration with error callbacks
// Matches TypeScript error-handled registration pattern
func TestErrorHandledRegistration(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	var callbackCalled int32
	var errorCallbackCalled int32

	callback := func() {
		atomic.StoreInt32(&callbackCalled, 1)
	}

	errorCallback := func(err error) {
		atomic.StoreInt32(&errorCallbackCalled, 1)
	}

	// Register timeout with error callback
	requestID := "test-error-handled"
	manager.RegisterTimeoutWithErrorCallback(requestID, 50*time.Millisecond, callback, errorCallback)

	if manager.ActiveTimeouts() != 1 {
		t.Errorf("Expected 1 active timeout, got %d", manager.ActiveTimeouts())
	}

	// Wait for timeout to fire
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&callbackCalled) != 1 {
		t.Error("Main callback should have been called")
	}

	if manager.ActiveTimeouts() != 0 {
		t.Errorf("Expected 0 active timeouts after firing, got %d", manager.ActiveTimeouts())
	}
}

// TestBilateralTimeoutManagement tests bilateral timeout registration and cancellation
// Matches TypeScript bilateral timeout implementation
func TestBilateralTimeoutManagement(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	var bilateralCallbackCalled int32

	bilateralCallback := func() {
		atomic.StoreInt32(&bilateralCallbackCalled, 1)
	}

	// Register bilateral timeout
	baseRequestID := "test-bilateral"
	requestID := baseRequestID + "-request"
	responseID := baseRequestID + "-response"

	manager.RegisterBilateralTimeout(requestID, responseID, 100*time.Millisecond, bilateralCallback)

	// Should have 2 active timeouts (request and response)
	if manager.ActiveTimeouts() != 2 {
		t.Errorf("Expected 2 active timeouts for bilateral, got %d", manager.ActiveTimeouts())
	}

	// Cancel bilateral timeout
	cancelledCount := manager.CancelBilateralTimeout(baseRequestID)

	if cancelledCount != 2 {
		t.Errorf("Expected to cancel 2 timeouts, cancelled %d", cancelledCount)
	}

	if manager.ActiveTimeouts() != 0 {
		t.Errorf("Expected 0 active timeouts after cancellation, got %d", manager.ActiveTimeouts())
	}

	// Wait to ensure callback doesn't fire
	time.Sleep(150 * time.Millisecond)

	if atomic.LoadInt32(&bilateralCallbackCalled) != 0 {
		t.Error("Bilateral callback should not have fired after cancellation")
	}
}

// TestBilateralTimeoutExpiration tests bilateral timeout expiration
func TestBilateralTimeoutExpiration(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	var bilateralCallbackCalled int32

	bilateralCallback := func() {
		atomic.StoreInt32(&bilateralCallbackCalled, 1)
	}

	// Register bilateral timeout with short duration
	baseRequestID := "test-bilateral-expire"
	requestID := baseRequestID + "-request"
	responseID := baseRequestID + "-response"

	manager.RegisterBilateralTimeout(requestID, responseID, 50*time.Millisecond, bilateralCallback)

	// Wait for timeout to expire
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&bilateralCallbackCalled) != 1 {
		t.Error("Bilateral callback should have fired after timeout")
	}

	if manager.ActiveTimeouts() != 0 {
		t.Errorf("Expected 0 active timeouts after expiration, got %d", manager.ActiveTimeouts())
	}
}

// TestTimeoutStatisticsAccuracy tests comprehensive timeout statistics tracking
// Matches TypeScript/Swift statistics implementation
func TestTimeoutStatisticsAccuracy(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	// Register multiple timeouts with different durations
	timeout1 := 100 * time.Millisecond
	timeout2 := 200 * time.Millisecond
	timeout3 := 50 * time.Millisecond

	manager.RegisterTimeout("cmd1", timeout1, func() {})
	manager.RegisterTimeout("cmd2", timeout2, func() {})
	manager.RegisterTimeout("cmd3", timeout3, func() {})

	stats := manager.GetTimeoutStatistics()

	if stats.ActiveTimeouts != 3 {
		t.Errorf("Expected 3 active timeouts, got %d", stats.ActiveTimeouts)
	}

	if stats.TotalRegistered != 3 {
		t.Errorf("Expected 3 total registered, got %d", stats.TotalRegistered)
	}

	if stats.TotalCancelled != 0 {
		t.Errorf("Expected 0 total cancelled, got %d", stats.TotalCancelled)
	}

	if stats.TotalExpired != 0 {
		t.Errorf("Expected 0 total expired, got %d", stats.TotalExpired)
	}

	// Check duration statistics
	expectedAverage := (timeout1 + timeout2 + timeout3) / 3
	if stats.AverageTimeout != expectedAverage {
		t.Errorf("Expected average timeout %v, got %v", expectedAverage, stats.AverageTimeout)
	}

	if stats.LongestTimeout != timeout2 {
		t.Errorf("Expected longest timeout %v, got %v", timeout2, stats.LongestTimeout)
	}

	if stats.ShortestTimeout != timeout3 {
		t.Errorf("Expected shortest timeout %v, got %v", timeout3, stats.ShortestTimeout)
	}

	// Cancel one timeout
	cancelled := manager.CancelTimeout("cmd2")
	if !cancelled {
		t.Error("Expected timeout cancellation to succeed")
	}

	// Check updated statistics
	statsAfterCancel := manager.GetTimeoutStatistics()
	if statsAfterCancel.TotalCancelled != 1 {
		t.Errorf("Expected 1 total cancelled after cancellation, got %d", statsAfterCancel.TotalCancelled)
	}

	if statsAfterCancel.ActiveTimeouts != 2 {
		t.Errorf("Expected 2 active timeouts after cancellation, got %d", statsAfterCancel.ActiveTimeouts)
	}

	// Wait for remaining timeouts to expire
	time.Sleep(250 * time.Millisecond)

	finalStats := manager.GetTimeoutStatistics()
	if finalStats.TotalExpired != 2 {
		t.Errorf("Expected 2 total expired, got %d", finalStats.TotalExpired)
	}

	if finalStats.ActiveTimeouts != 0 {
		t.Errorf("Expected 0 active timeouts after expiration, got %d", finalStats.ActiveTimeouts)
	}
}

// TestTimeoutManagerConcurrency tests concurrent timeout operations
// Ensures thread safety of enhanced timeout management
func TestTimeoutManagerConcurrency(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	// Launch multiple goroutines registering timeouts concurrently
	numGoroutines := 10
	timeoutsPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < timeoutsPerGoroutine; j++ {
				requestID := fmt.Sprintf("concurrent-%d-%d", goroutineID, j)
				manager.RegisterTimeout(requestID, 100*time.Millisecond, func() {})
			}
		}(i)
	}

	// Give goroutines time to register timeouts
	time.Sleep(50 * time.Millisecond)

	stats := manager.GetTimeoutStatistics()
	expectedTimeouts := numGoroutines * timeoutsPerGoroutine

	if stats.TotalRegistered != int64(expectedTimeouts) {
		t.Errorf("Expected %d total registered timeouts, got %d", expectedTimeouts, stats.TotalRegistered)
	}

	if stats.ActiveTimeouts != expectedTimeouts {
		t.Errorf("Expected %d active timeouts, got %d", expectedTimeouts, stats.ActiveTimeouts)
	}

	// Test concurrent cancellations
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < timeoutsPerGoroutine/2; j++ {
				requestID := fmt.Sprintf("concurrent-%d-%d", goroutineID, j)
				manager.CancelTimeout(requestID)
			}
		}(i)
	}

	// Give cancellations time to complete
	time.Sleep(50 * time.Millisecond)

	// Wait for remaining timeouts to expire
	time.Sleep(200 * time.Millisecond)

	finalStats := manager.GetTimeoutStatistics()

	// Should have some cancelled and some expired
	totalProcessed := finalStats.TotalCancelled + finalStats.TotalExpired
	if totalProcessed != int64(expectedTimeouts) {
		t.Errorf("Expected total processed (cancelled + expired) to be %d, got %d", expectedTimeouts, totalProcessed)
	}

	if finalStats.ActiveTimeouts != 0 {
		t.Errorf("Expected 0 active timeouts after test completion, got %d", finalStats.ActiveTimeouts)
	}
}

// TestTimeoutExtensionBoundaryConditions tests edge cases for timeout extension
func TestTimeoutExtensionBoundaryConditions(t *testing.T) {
	manager := protocol.NewTimeoutManager()
	defer manager.Close()

	// Test extending with zero duration
	manager.RegisterTimeout("test-zero-extend", 100*time.Millisecond, func() {})
	extended := manager.ExtendTimeout("test-zero-extend", 0)
	if !extended {
		t.Error("Expected zero-duration extension to succeed")
	}

	// Test extending with negative duration (should still work as it's just adding to current)
	extended = manager.ExtendTimeout("test-zero-extend", -50*time.Millisecond)
	if !extended {
		t.Error("Expected negative extension to succeed (reduces timeout)")
	}

	// Test extending already expired timeout
	manager.RegisterTimeout("test-quick-expire", 1*time.Millisecond, func() {})
	time.Sleep(10 * time.Millisecond) // Let it expire
	extended = manager.ExtendTimeout("test-quick-expire", 100*time.Millisecond)
	if extended {
		t.Error("Expected extension of expired timeout to fail")
	}
}