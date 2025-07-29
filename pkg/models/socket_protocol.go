package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SocketCommand represents a command message sent through the Unix socket
// SOCK_DGRAM version with reply_to field for connectionless communication
type SocketCommand struct {
	ID        string                 `json:"id"`
	ChannelID string                 `json:"channelId"`
	Command   string                 `json:"command"`
	ReplyTo   string                 `json:"reply_to,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
	Timeout   *float64               `json:"timeout,omitempty"`
	Timestamp float64                `json:"timestamp"`
}

// SocketResponse represents a response message from the Unix socket
// Matches the Swift SocketResponse structure exactly for cross-language compatibility
type SocketResponse struct {
	CommandID string                 `json:"commandId"`
	ChannelID string                 `json:"channelId"`
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     *SocketError           `json:"error,omitempty"`
	Timestamp float64                `json:"timestamp"`
}

// SocketError represents error information in responses
// Matches the Swift error structure for consistency
type SocketError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *SocketError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// SocketMessage represents the envelope for all socket communications
// Uses the same structure as Swift/Rust for protocol compatibility
type SocketMessage struct {
	Type    string `json:"type"` // "command" or "response"
	Payload string `json:"payload"` // base64-encoded data
}

// NewSocketCommand creates a new command with generated UUID and timestamp
func NewSocketCommand(channelID, command string, args map[string]interface{}, timeout *float64) *SocketCommand {
	return &SocketCommand{
		ID:        uuid.New().String(),
		ChannelID: channelID,
		Command:   command,
		Args:      args,
		Timeout:   timeout,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}

// NewSuccessResponse creates a successful response for a command
func NewSuccessResponse(commandID, channelID string, result map[string]interface{}) *SocketResponse {
	return &SocketResponse{
		CommandID: commandID,
		ChannelID: channelID,
		Success:   true,
		Result:    result,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}

// NewErrorResponse creates an error response for a command
func NewErrorResponse(commandID, channelID string, err *SocketError) *SocketResponse {
	return &SocketResponse{
		CommandID: commandID,
		ChannelID: channelID,
		Success:   false,
		Error:     err,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}

// ToJSON serializes the command to JSON bytes
func (c *SocketCommand) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON deserializes JSON bytes to a command
func (c *SocketCommand) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// ToJSON serializes the response to JSON bytes
func (r *SocketResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON deserializes JSON bytes to a response
func (r *SocketResponse) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// CommandHandler represents a function that handles incoming commands
// Matches the Swift CommandHandler signature for compatibility
type CommandHandler func(command *SocketCommand) (*SocketResponse, error)

// TimeoutHandler represents a function called when a command times out
// Matches the Swift TimeoutHandler signature for compatibility
type TimeoutHandler func(commandID string)