package protocol

import (
	"fmt"
	"sync"
	"time"

	"github.com/jowharshamshiri/GoJanus/pkg/models"
)

// PendingCommand represents a command awaiting response
type PendingCommand struct {
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
	MaxPendingCommands int
	CleanupInterval    time.Duration
	DefaultTimeout     time.Duration
}

// ResponseTracker manages async response correlation and timeout handling
type ResponseTracker struct {
	pendingCommands    map[string]*PendingCommand
	mutex              sync.RWMutex
	cleanupTimer       *time.Ticker
	cleanupDone        chan bool
	config             TrackerConfig
	eventHandlers      map[string][]func(interface{})
	eventMutex         sync.RWMutex
}

// NewResponseTracker creates a new response tracker
func NewResponseTracker(config TrackerConfig) *ResponseTracker {
	if config.MaxPendingCommands <= 0 {
		config.MaxPendingCommands = 1000
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 30 * time.Second
	}
	if config.DefaultTimeout <= 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	tracker := &ResponseTracker{
		pendingCommands: make(map[string]*PendingCommand),
		cleanupDone:     make(chan bool),
		config:          config,
		eventHandlers:   make(map[string][]func(interface{})),
	}

	tracker.startCleanupTimer()
	return tracker
}

// TrackCommand tracks a command awaiting response
func (rt *ResponseTracker) TrackCommand(
	commandID string,
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
	if len(rt.pendingCommands) >= rt.config.MaxPendingCommands {
		return &ResponseTrackerError{
			Message: "Too many pending commands",
			Code:    "PENDING_COMMANDS_LIMIT",
			Details: fmt.Sprintf("Maximum %d commands allowed", rt.config.MaxPendingCommands),
		}
	}

	// Check for duplicate tracking
	if _, exists := rt.pendingCommands[commandID]; exists {
		return &ResponseTrackerError{
			Message: "Command already being tracked",
			Code:    "DUPLICATE_COMMAND_ID",
			Details: fmt.Sprintf("Command %s is already awaiting response", commandID),
		}
	}

	// Create pending command entry
	pending := &PendingCommand{
		Resolve:   resolve,
		Reject:    reject,
		Timestamp: time.Now(),
		Timeout:   timeout,
	}

	rt.pendingCommands[commandID] = pending

	// Set individual timeout
	go func() {
		time.Sleep(timeout)
		rt.handleTimeout(commandID)
	}()

	return nil
}

// HandleResponse handles an incoming response
func (rt *ResponseTracker) HandleResponse(response *models.JanusResponse) bool {
	rt.mutex.Lock()
	pending, exists := rt.pendingCommands[response.CommandID]
	if exists {
		delete(rt.pendingCommands, response.CommandID)
	}
	rt.mutex.Unlock()

	if !exists {
		// Response for unknown command (possibly timed out)
		return false
	}

	// Emit cleanup event
	rt.emit("cleanup", response.CommandID)

	// Emit response event
	rt.emit("response", map[string]interface{}{
		"commandId": response.CommandID,
		"response":  response,
	})

	// Resolve or reject based on response
	if response.Success {
		select {
		case pending.Resolve <- response:
		default:
		}
	} else {
		errorMsg := "Command failed"
		errorCode := "COMMAND_FAILED"
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

// CancelCommand cancels tracking for a command
func (rt *ResponseTracker) CancelCommand(commandID string, reason string) bool {
	rt.mutex.Lock()
	pending, exists := rt.pendingCommands[commandID]
	if exists {
		delete(rt.pendingCommands, commandID)
	}
	rt.mutex.Unlock()

	if !exists {
		return false
	}

	rt.emit("cleanup", commandID)

	if reason == "" {
		reason = "Command cancelled"
	}

	err := &ResponseTrackerError{
		Message: reason,
		Code:    "COMMAND_CANCELLED",
		Details: fmt.Sprintf("Command %s was cancelled", commandID),
	}

	select {
	case pending.Reject <- err:
	default:
	}

	return true
}

// CancelAllCommands cancels all pending commands
func (rt *ResponseTracker) CancelAllCommands(reason string) int {
	rt.mutex.Lock()
	commands := make(map[string]*PendingCommand)
	for id, pending := range rt.pendingCommands {
		commands[id] = pending
	}
	rt.pendingCommands = make(map[string]*PendingCommand)
	rt.mutex.Unlock()

	if reason == "" {
		reason = "All commands cancelled"
	}

	count := len(commands)
	for commandID, pending := range commands {
		rt.emit("cleanup", commandID)

		err := &ResponseTrackerError{
			Message: reason,
			Code:    "ALL_COMMANDS_CANCELLED",
			Details: reason,
		}

		select {
		case pending.Reject <- err:
		default:
		}
	}

	return count
}

// GetPendingCount returns the number of pending commands
func (rt *ResponseTracker) GetPendingCount() int {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	return len(rt.pendingCommands)
}

// GetPendingCommandIDs returns list of pending command IDs
func (rt *ResponseTracker) GetPendingCommandIDs() []string {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	ids := make([]string, 0, len(rt.pendingCommands))
	for id := range rt.pendingCommands {
		ids = append(ids, id)
	}
	return ids
}

// IsTracking checks if a command is being tracked
func (rt *ResponseTracker) IsTracking(commandID string) bool {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	_, exists := rt.pendingCommands[commandID]
	return exists
}

// CommandStatistics holds statistics about pending commands
type CommandStatistics struct {
	PendingCount   int                  `json:"pendingCount"`
	AverageAge     float64              `json:"averageAge"`
	OldestCommand  *CommandInfo         `json:"oldestCommand,omitempty"`
	NewestCommand  *CommandInfo         `json:"newestCommand,omitempty"`
}

// CommandInfo holds information about a command
type CommandInfo struct {
	ID  string  `json:"id"`
	Age float64 `json:"age"`
}

// GetStatistics returns statistics about pending commands
func (rt *ResponseTracker) GetStatistics() CommandStatistics {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	now := time.Now()
	stats := CommandStatistics{
		PendingCount: len(rt.pendingCommands),
	}

	if len(rt.pendingCommands) == 0 {
		return stats
	}

	type commandAge struct {
		id  string
		age float64
	}

	ages := make([]commandAge, 0, len(rt.pendingCommands))
	totalAge := 0.0

	for id, pending := range rt.pendingCommands {
		age := now.Sub(pending.Timestamp).Seconds()
		ages = append(ages, commandAge{id: id, age: age})
		totalAge += age
	}

	stats.AverageAge = totalAge / float64(len(ages))

	// Find oldest and newest
	var oldest, newest *commandAge
	for i := range ages {
		if oldest == nil || ages[i].age > oldest.age {
			oldest = &ages[i]
		}
		if newest == nil || ages[i].age < newest.age {
			newest = &ages[i]
		}
	}

	if oldest != nil {
		stats.OldestCommand = &CommandInfo{ID: oldest.id, Age: oldest.age}
	}
	if newest != nil {
		stats.NewestCommand = &CommandInfo{ID: newest.id, Age: newest.age}
	}

	return stats
}

// Cleanup removes expired commands
func (rt *ResponseTracker) Cleanup() int {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	now := time.Now()
	cleanedCount := 0
	expiredCommands := []string{}

	for commandID, pending := range rt.pendingCommands {
		age := now.Sub(pending.Timestamp)
		if age >= pending.Timeout {
			expiredCommands = append(expiredCommands, commandID)
		}
	}

	for _, commandID := range expiredCommands {
		delete(rt.pendingCommands, commandID)
		cleanedCount++
		go rt.handleTimeout(commandID)
	}

	return cleanedCount
}

// Shutdown cleans up resources
func (rt *ResponseTracker) Shutdown() {
	if rt.cleanupTimer != nil {
		rt.cleanupTimer.Stop()
		rt.cleanupDone <- true
	}
	rt.CancelAllCommands("Tracker shutdown")
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

// handleTimeout handles command timeout
func (rt *ResponseTracker) handleTimeout(commandID string) {
	rt.mutex.Lock()
	pending, exists := rt.pendingCommands[commandID]
	if exists {
		delete(rt.pendingCommands, commandID)
	}
	rt.mutex.Unlock()

	if !exists {
		return // Already handled
	}

	rt.emit("timeout", commandID)
	rt.emit("cleanup", commandID)

	err := &ResponseTrackerError{
		Message: "Command timeout",
		Code:    "COMMAND_TIMEOUT",
		Details: fmt.Sprintf("Command %s timed out after %v", commandID, pending.Timeout),
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