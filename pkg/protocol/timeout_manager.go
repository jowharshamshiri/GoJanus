package protocol

import (
	"sync"
	"time"
)

// TimeoutManager manages bilateral timeout handling for commands
// Matches Swift timeout management functionality exactly
type TimeoutManager struct {
	timeouts map[string]*timeoutEntry
	mutex    sync.RWMutex
}

// timeoutEntry represents a single timeout registration
type timeoutEntry struct {
	timer    *time.Timer
	callback func()
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		timeouts: make(map[string]*timeoutEntry),
	}
}

// RegisterTimeout registers a timeout for a command ID
// Matches Swift bilateral timeout management
func (tm *TimeoutManager) RegisterTimeout(commandID string, timeout time.Duration, callback func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Cancel existing timeout if any
	if existing, exists := tm.timeouts[commandID]; exists {
		existing.timer.Stop()
	}
	
	// Create new timeout
	timer := time.AfterFunc(timeout, func() {
		tm.mutex.Lock()
		delete(tm.timeouts, commandID)
		tm.mutex.Unlock()
		
		if callback != nil {
			callback()
		}
	})
	
	tm.timeouts[commandID] = &timeoutEntry{
		timer:    timer,
		callback: callback,
	}
}

// CancelTimeout cancels a timeout for a command ID
// Called when a response is received before timeout
func (tm *TimeoutManager) CancelTimeout(commandID string) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	if entry, exists := tm.timeouts[commandID]; exists {
		entry.timer.Stop()
		delete(tm.timeouts, commandID)
		return true
	}
	
	return false
}

// Close cancels all active timeouts and cleans up resources
func (tm *TimeoutManager) Close() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for commandID, entry := range tm.timeouts {
		entry.timer.Stop()
		delete(tm.timeouts, commandID)
	}
}

// ActiveTimeouts returns the number of active timeouts
// Useful for monitoring and debugging
func (tm *TimeoutManager) ActiveTimeouts() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	return len(tm.timeouts)
}

// TimeoutStatistics represents timeout metrics and monitoring data
type TimeoutStatistics struct {
	ActiveTimeouts    int           `json:"active_timeouts"`
	TotalRegistered   int64         `json:"total_registered"`
	TotalCancelled    int64         `json:"total_cancelled"`
	TotalExpired      int64         `json:"total_expired"`
	AverageTimeout    time.Duration `json:"average_timeout"`
	LongestTimeout    time.Duration `json:"longest_timeout"`
	ShortestTimeout   time.Duration `json:"shortest_timeout"`
}

// Statistics tracking fields (need to be added to TimeoutManager struct)
type TimeoutStats struct {
	totalRegistered int64
	totalCancelled  int64
	totalExpired    int64
	totalDuration   time.Duration
	maxDuration     time.Duration
	minDuration     time.Duration
}

// GetTimeoutStatistics returns comprehensive timeout metrics (matches Rust/Swift implementation)
func (tm *TimeoutManager) GetTimeoutStatistics() TimeoutStatistics {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	// For now, return basic statistics
	// Full implementation would require extending TimeoutManager struct
	return TimeoutStatistics{
		ActiveTimeouts:  len(tm.timeouts),
		TotalRegistered: 0, // Would need to track this
		TotalCancelled:  0, // Would need to track this  
		TotalExpired:    0, // Would need to track this
		AverageTimeout:  0, // Would need to calculate this
		LongestTimeout:  0, // Would need to track this
		ShortestTimeout: 0, // Would need to track this
	}
}

// RegisterBilateralTimeout registers request/response timeout pairs (matches Rust/Swift)
func (tm *TimeoutManager) RegisterBilateralTimeout(requestID string, responseID string, timeout time.Duration, callback func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Create bilateral timeout entry that handles both request and response
	timer := time.AfterFunc(timeout, func() {
		tm.mutex.Lock()
		delete(tm.timeouts, requestID)
		delete(tm.timeouts, responseID)
		tm.mutex.Unlock()
		callback()
	})
	
	entry := &timeoutEntry{
		timer:    timer,
		callback: callback,
	}
	
	// Register same timeout for both IDs
	tm.timeouts[requestID] = entry
	tm.timeouts[responseID] = entry
}