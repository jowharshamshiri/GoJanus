package protocol

import (
	"fmt"
	"sync"
	"time"

	"GoJanus/pkg/models"
)

// PendingRequest represents a request awaiting response
type PendingRequest struct {
	Resolve   chan *models.JanusResponse
	Reject    chan error
	Timestamp time.Time
	Timeout   time.Duration
}

// ResponseTrackerError represents response tracking errors
type ResponseTrackerError struct {
	Message string
	Code    string
	Details string
}

func (e *ResponseTrackerError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s (%s): %s", e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// TrackerConfig configures the response tracker
type TrackerConfig struct {
	MaxPendingRequests int
	CleanupInterval    time.Duration
	DefaultTimeout     time.Duration
}

// ResponseTracker manages async response correlation and timeout handling
type ResponseTracker struct {
	pendingRequests    map[string]*PendingRequest
	mutex              sync.RWMutex
	cleanupTimer       *time.Ticker
	cleanupDone        chan bool
	config             TrackerConfig
	eventHandlers      map[string][]func(interface{})
	eventMutex         sync.RWMutex
}

// NewResponseTracker creates a new response tracker
func NewResponseTracker(config TrackerConfig) *ResponseTracker {
	if config.MaxPendingRequests <= 0 {
		config.MaxPendingRequests = 1000
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 30 * time.Second
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	tracker := &ResponseTracker{
		pendingRequests: make(map[string]*PendingRequest),
		cleanupDone:     make(chan bool),
		config:          config,
		eventHandlers:   make(map[string][]func(interface{})),
	}

	tracker.startCleanupTimer()
	return tracker
}

// TrackRequest tracks a request awaiting response
func (rt *ResponseTracker) TrackRequest(
	requestID string,
	resolve chan *models.JanusResponse,
	reject chan error,
	timeout time.Duration,
) error {
	if timeout <= 0 {
		timeout = rt.config.DefaultTimeout
	}

	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	// Check limits
	if len(rt.pendingRequests) >= rt.config.MaxPendingRequests {
		return &ResponseTrackerError{
			Message: "Too many pending requests",
			Code:    "PENDING_REQUESTS_LIMIT",
			Details: fmt.Sprintf("Maximum %d requests allowed", rt.config.MaxPendingRequests),
		}
	}

	// Check for duplicate tracking
	if _, exists := rt.pendingRequests[requestID]; exists {
		return &ResponseTrackerError{
			Message: "Request already being tracked",
			Code:    "DUPLICATE_REQUEST_ID",
			Details: fmt.Sprintf("Request %s is already awaiting response", requestID),
		}
	}

	// Create pending request entry
	pending := &PendingRequest{
		Resolve:   resolve,
		Reject:    reject,
		Timestamp: time.Now(),
		Timeout:   timeout,
	}

	rt.pendingRequests[requestID] = pending

	// Set individual timeout
	go func() {
		time.Sleep(timeout)
		rt.handleTimeout(requestID)
	}()

	return nil
}

// HandleResponse handles an incoming response
func (rt *ResponseTracker) HandleResponse(response *models.JanusResponse) bool {
	rt.mutex.Lock()
	pending, exists := rt.pendingRequests[response.RequestID]
	if exists {
		delete(rt.pendingRequests, response.RequestID)
	}
	rt.mutex.Unlock()

	if !exists {
		// Response for unknown request (possibly timed out)
		return false
	}

	// Emit cleanup event
	rt.emit("cleanup", response.RequestID)

	// Emit response event
	rt.emit("response", map[string]interface{}{
		"requestId": response.RequestID,
		"response":  response,
	})

	// Resolve or reject based on response
	if response.Success {
		select {
		case pending.Resolve <- response:
		default:
		}
	} else {
		errorMsg := "Request failed"
		errorCode := "REQUEST_FAILED"
		errorDetails := ""

		if response.Error != nil {
			if response.Error.Message != "" {
				errorMsg = response.Error.Message
			}
			errorCode = response.Error.Code.String()
			if response.Error.Data != nil && response.Error.Data.Details != "" {
				errorDetails = response.Error.Data.Details
			}
		}

		err := &ResponseTrackerError{
			Message: errorMsg,
			Code:    errorCode,
			Details: errorDetails,
		}

		select {
		case pending.Reject <- err:
		default:
		}
	}

	return true
}

// CancelRequest cancels tracking for a request
func (rt *ResponseTracker) CancelRequest(requestID string, reason string) bool {
	rt.mutex.Lock()
	pending, exists := rt.pendingRequests[requestID]
	if exists {
		delete(rt.pendingRequests, requestID)
	}
	rt.mutex.Unlock()

	if !exists {
		return false
	}

	rt.emit("cleanup", requestID)

	if reason == "" {
		reason = "Request cancelled"
	}

	err := &ResponseTrackerError{
		Message: reason,
		Code:    "REQUEST_CANCELLED",
		Details: fmt.Sprintf("Request %s was cancelled", requestID),
	}

	select {
	case pending.Reject <- err:
	default:
	}

	return true
}

// CancelAllRequests cancels all pending requests
func (rt *ResponseTracker) CancelAllRequests(reason string) int {
	rt.mutex.Lock()
	requests := make(map[string]*PendingRequest)
	for id, pending := range rt.pendingRequests {
		requests[id] = pending
	}
	rt.pendingRequests = make(map[string]*PendingRequest)
	rt.mutex.Unlock()

	if reason == "" {
		reason = "All requests cancelled"
	}

	count := len(requests)
	for requestID, pending := range requests {
		rt.emit("cleanup", requestID)

		err := &ResponseTrackerError{
			Message: reason,
			Code:    "ALL_REQUESTS_CANCELLED",
			Details: reason,
		}

		select {
		case pending.Reject <- err:
		default:
		}
	}

	return count
}

// GetPendingCount returns the number of pending requests
func (rt *ResponseTracker) GetPendingCount() int {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	return len(rt.pendingRequests)
}

// GetPendingRequestIDs returns list of pending request IDs
func (rt *ResponseTracker) GetPendingRequestIDs() []string {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	ids := make([]string, 0, len(rt.pendingRequests))
	for id := range rt.pendingRequests {
		ids = append(ids, id)
	}
	return ids
}

// IsTracking checks if a request is being tracked
func (rt *ResponseTracker) IsTracking(requestID string) bool {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	_, exists := rt.pendingRequests[requestID]
	return exists
}

// RequestStatistics holds statistics about pending requests
type RequestStatistics struct {
	PendingCount   int                  `json:"pendingCount"`
	AverageAge     float64              `json:"averageAge"`
	OldestRequest  *RequestInfo         `json:"oldestRequest,omitempty"`
	NewestRequest  *RequestInfo         `json:"newestRequest,omitempty"`
}

// RequestInfo holds information about a request
type RequestInfo struct {
	ID  string  `json:"id"`
	Age float64 `json:"age"`
}

// GetStatistics returns statistics about pending requests
func (rt *ResponseTracker) GetStatistics() RequestStatistics {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	now := time.Now()
	stats := RequestStatistics{
		PendingCount: len(rt.pendingRequests),
	}

	if len(rt.pendingRequests) == 0 {
		return stats
	}

	type requestAge struct {
		id  string
		age float64
	}

	ages := make([]requestAge, 0, len(rt.pendingRequests))
	totalAge := 0.0

	for id, pending := range rt.pendingRequests {
		age := now.Sub(pending.Timestamp).Seconds()
		ages = append(ages, requestAge{id: id, age: age})
		totalAge += age
	}

	stats.AverageAge = totalAge / float64(len(ages))

	// Find oldest and newest
	var oldest, newest *requestAge
	for i := range ages {
		if oldest == nil || ages[i].age > oldest.age {
			oldest = &ages[i]
		}
		if newest == nil || ages[i].age < newest.age {
			newest = &ages[i]
		}
	}

	if oldest != nil {
		stats.OldestRequest = &RequestInfo{ID: oldest.id, Age: oldest.age}
	}
	if newest != nil {
		stats.NewestRequest = &RequestInfo{ID: newest.id, Age: newest.age}
	}

	return stats
}

// Cleanup removes expired requests
func (rt *ResponseTracker) Cleanup() int {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	now := time.Now()
	cleanedCount := 0
	expiredRequests := []string{}

	for requestID, pending := range rt.pendingRequests {
		age := now.Sub(pending.Timestamp)
		if age >= pending.Timeout {
			expiredRequests = append(expiredRequests, requestID)
		}
	}

	for _, requestID := range expiredRequests {
		delete(rt.pendingRequests, requestID)
		cleanedCount++
		go rt.handleTimeout(requestID)
	}

	return cleanedCount
}

// Shutdown cleans up resources
func (rt *ResponseTracker) Shutdown() {
	if rt.cleanupTimer != nil {
		rt.cleanupTimer.Stop()
		rt.cleanupDone <- true
	}
	rt.CancelAllRequests("Tracker shutdown")
}

// On registers an event handler
func (rt *ResponseTracker) On(event string, handler func(interface{})) {
	rt.eventMutex.Lock()
	defer rt.eventMutex.Unlock()

	if rt.eventHandlers[event] == nil {
		rt.eventHandlers[event] = make([]func(interface{}), 0)
	}
	rt.eventHandlers[event] = append(rt.eventHandlers[event], handler)
}

// handleTimeout handles request timeout
func (rt *ResponseTracker) handleTimeout(requestID string) {
	rt.mutex.Lock()
	pending, exists := rt.pendingRequests[requestID]
	if exists {
		delete(rt.pendingRequests, requestID)
	}
	rt.mutex.Unlock()

	if !exists {
		return // Already handled
	}

	rt.emit("timeout", requestID)
	rt.emit("cleanup", requestID)

	err := &ResponseTrackerError{
		Message: "Request timeout",
		Code:    "REQUEST_TIMEOUT",
		Details: fmt.Sprintf("Request %s timed out after %v", requestID, pending.Timeout),
	}

	select {
	case pending.Reject <- err:
	default:
	}
}

// emit emits an event to all registered handlers
func (rt *ResponseTracker) emit(event string, data interface{}) {
	rt.eventMutex.RLock()
	handlers := rt.eventHandlers[event]
	rt.eventMutex.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// startCleanupTimer starts the periodic cleanup timer
func (rt *ResponseTracker) startCleanupTimer() {
	rt.cleanupTimer = time.NewTicker(rt.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-rt.cleanupTimer.C:
				cleaned := rt.Cleanup()
				if cleaned > 0 {
					// Could emit cleanup stats event here
				}
			case <-rt.cleanupDone:
				return
			}
		}
	}()
}