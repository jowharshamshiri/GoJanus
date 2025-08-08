package protocol

import (
	"sync"
	"time"
)

// TimeoutManager manages bilateral timeout handling for requests
// Matches Swift timeout management functionality exactly
type TimeoutManager struct {
	timeouts map[string]*timeoutEntry
	mutex    sync.RWMutex
	stats    TimeoutStats
}

// timeoutEntry represents a single timeout registration
type timeoutEntry struct {
	timer       *time.Timer
	callback    func()
	errorCallback func(error)
	timeout     time.Duration
	registeredAt time.Time
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		timeouts: make(map[string]*timeoutEntry),
		stats: TimeoutStats{
			minDuration: time.Hour, // Initialize with large value
		},
	}
}

// RegisterTimeout registers a timeout for a request ID
// Matches Swift bilateral timeout management
func (tm *TimeoutManager) RegisterTimeout(requestID string, timeout time.Duration, callback func()) {
	tm.registerTimeoutWithErrorCallback(requestID, timeout, callback, nil)
}

// RegisterTimeoutWithErrorCallback registers a timeout with error handling callback
// Matches TypeScript error-handled registration pattern
func (tm *TimeoutManager) RegisterTimeoutWithErrorCallback(requestID string, timeout time.Duration, callback func(), errorCallback func(error)) {
	tm.registerTimeoutWithErrorCallback(requestID, timeout, callback, errorCallback)
}

// Internal method for timeout registration with optional error callback
func (tm *TimeoutManager) registerTimeoutWithErrorCallback(requestID string, timeout time.Duration, callback func(), errorCallback func(error)) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Cancel existing timeout if any
	if existing, exists := tm.timeouts[requestID]; exists {
		existing.timer.Stop()
		tm.stats.totalCancelled++
	}
	
	// Update stats
	tm.stats.totalRegistered++
	tm.stats.totalDuration += timeout
	if timeout > tm.stats.maxDuration {
		tm.stats.maxDuration = timeout
	}
	if timeout < tm.stats.minDuration {
		tm.stats.minDuration = timeout
	}
	
	// Create new timeout
	registeredAt := time.Now()
	timer := time.AfterFunc(timeout, func() {
		tm.mutex.Lock()
		delete(tm.timeouts, requestID)
		tm.stats.totalExpired++
		tm.mutex.Unlock()
		
		if callback != nil {
			callback()
		}
	})
	
	tm.timeouts[requestID] = &timeoutEntry{
		timer:         timer,
		callback:      callback,
		errorCallback: errorCallback,
		timeout:       timeout,
		registeredAt:  registeredAt,
	}
}

// CancelTimeout cancels a timeout for a request ID
// Called when a response is received before timeout
func (tm *TimeoutManager) CancelTimeout(requestID string) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	if entry, exists := tm.timeouts[requestID]; exists {
		entry.timer.Stop()
		delete(tm.timeouts, requestID)
		tm.stats.totalCancelled++
		return true
	}
	
	return false
}

// ExtendTimeout extends an existing timeout by the manifestified duration
// Matches Swift/TypeScript timeout extension capability
func (tm *TimeoutManager) ExtendTimeout(requestID string, extension time.Duration) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	entry, exists := tm.timeouts[requestID]
	if !exists {
		return false
	}
	
	// Stop the existing timer
	entry.timer.Stop()
	
	// Create a new timer with extended duration
	newTimeout := entry.timeout + extension
	entry.timeout = newTimeout
	
	entry.timer = time.AfterFunc(newTimeout, func() {
		tm.mutex.Lock()
		delete(tm.timeouts, requestID)
		tm.stats.totalExpired++
		tm.mutex.Unlock()
		
		if entry.callback != nil {
			entry.callback()
		}
	})
	
	// Update the entry in the map
	tm.timeouts[requestID] = entry
	
	return true
}

// Close cancels all active timeouts and cleans up resources
func (tm *TimeoutManager) Close() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for requestID, entry := range tm.timeouts {
		entry.timer.Stop()
		delete(tm.timeouts, requestID)
		tm.stats.totalCancelled++
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
	
	var averageTimeout time.Duration
	if tm.stats.totalRegistered > 0 {
		averageTimeout = tm.stats.totalDuration / time.Duration(tm.stats.totalRegistered)
	}
	
	return TimeoutStatistics{
		ActiveTimeouts:  len(tm.timeouts),
		TotalRegistered: tm.stats.totalRegistered,
		TotalCancelled:  tm.stats.totalCancelled,
		TotalExpired:    tm.stats.totalExpired,
		AverageTimeout:  averageTimeout,
		LongestTimeout:  tm.stats.maxDuration,
		ShortestTimeout: tm.stats.minDuration,
	}
}

// RegisterBilateralTimeout registers request/response timeout pairs (matches Rust/Swift)
func (tm *TimeoutManager) RegisterBilateralTimeout(requestID string, responseID string, timeout time.Duration, callback func()) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	// Update stats
	tm.stats.totalRegistered++
	tm.stats.totalDuration += timeout
	if timeout > tm.stats.maxDuration {
		tm.stats.maxDuration = timeout
	}
	if timeout < tm.stats.minDuration {
		tm.stats.minDuration = timeout
	}
	
	registeredAt := time.Now()
	
	// Create bilateral timeout entry that handles both request and response
	timer := time.AfterFunc(timeout, func() {
		tm.mutex.Lock()
		delete(tm.timeouts, requestID)
		delete(tm.timeouts, responseID)
		tm.stats.totalExpired++
		tm.mutex.Unlock()
		callback()
	})
	
	entry := &timeoutEntry{
		timer:         timer,
		callback:      callback,
		errorCallback: nil,
		timeout:       timeout,
		registeredAt:  registeredAt,
	}
	
	// Register same timeout for both IDs
	tm.timeouts[requestID] = entry
	tm.timeouts[responseID] = entry
}

// CancelBilateralTimeout cancels both request and response timeouts
// Matches TypeScript implementation pattern
func (tm *TimeoutManager) CancelBilateralTimeout(baseRequestID string) int {
	requestID := baseRequestID + "-request"
	responseID := baseRequestID + "-response"
	
	cancelledCount := 0
	
	if tm.CancelTimeout(requestID) {
		cancelledCount++
	}
	
	if tm.CancelTimeout(responseID) {
		cancelledCount++
	}
	
	return cancelledCount
}