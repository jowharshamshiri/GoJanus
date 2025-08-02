package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JanusCommand represents a command message sent through the Unix socket
// SOCK_DGRAM version with reply_to field for connectionless communication
type JanusCommand struct {
	ID        string                 `json:"id"`
	ChannelID string                 `json:"channelId"`
	Command   string                 `json:"command"`
	ReplyTo   *string                `json:"reply_to,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
	Timeout   *float64               `json:"timeout,omitempty"`
	Timestamp float64                `json:"timestamp"`
}

// JanusResponse represents a response message from the Unix socket
// Matches the Swift JanusResponse structure exactly for cross-language compatibility
type JanusResponse struct {
	CommandID string                 `json:"commandId"`
	ChannelID string                 `json:"channelId"`
	Success   bool                   `json:"success"`
	Result    interface{} `json:"result,omitempty"`
	Error     *JSONRPCError          `json:"error,omitempty"`
	Timestamp float64                `json:"timestamp"`
}


// SocketMessage represents the envelope for all socket communications
// Uses the same structure as Swift/Rust for protocol compatibility
type SocketMessage struct {
	Type    string `json:"type"` // "command" or "response"
	Payload string `json:"payload"` // base64-encoded data
}

// NewJanusCommand creates a new command with generated UUID and timestamp
func NewJanusCommand(channelID, command string, args map[string]interface{}, timeout *float64) *JanusCommand {
	return &JanusCommand{
		ID:        uuid.New().String(),
		ChannelID: channelID,
		Command:   command,
		Args:      args,
		Timeout:   timeout,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}

// NewSuccessResponse creates a successful response for a command
func NewSuccessResponse(commandID, channelID string, result map[string]interface{}) *JanusResponse {
	return &JanusResponse{
		CommandID: commandID,
		ChannelID: channelID,
		Success:   true,
		Result:    result,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}

// NewErrorResponse creates an error response for a command
func NewErrorResponse(commandID, channelID string, err *JSONRPCError) *JanusResponse {
	return &JanusResponse{
		CommandID: commandID,
		ChannelID: channelID,
		Success:   false,
		Error:     err,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}
}


// ToJSON serializes the command to JSON bytes
func (c *JanusCommand) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON deserializes JSON bytes to a command
func (c *JanusCommand) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// ToJSON serializes the response to JSON bytes
func (r *JanusResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON deserializes JSON bytes to a response
func (r *JanusResponse) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// CommandHandler represents a function that handles incoming commands
// Matches the Swift CommandHandler signature for compatibility
type CommandHandler func(command *JanusCommand) (*JanusResponse, error)

// TimeoutHandler represents a function called when a command times out
// Matches the Swift TimeoutHandler signature for compatibility
type TimeoutHandler func(commandID string)