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