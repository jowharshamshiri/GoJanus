package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JanusRequest represents a request message sent through the Unix socket
// PRIME DIRECTIVE: Exact format for 100% cross-platform compatibility
type JanusRequest struct {
	ID        string                 `json:"id"`
	Method    string                 `json:"method"`
	Request   string                 `json:"request"`
	ReplyTo   *string                `json:"reply_to,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
	Timeout   *float64               `json:"timeout,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// JanusResponse represents a response message from the Unix socket
// PRIME DIRECTIVE: Exact format for 100% cross-platform compatibility
type JanusResponse struct {
	Result    interface{}   `json:"result"`
	Error     *JSONRPCError `json:"error"`
	Success   bool          `json:"success"`
	RequestID string        `json:"request_id"`
	ID        string        `json:"id"`
	Timestamp string        `json:"timestamp"`
}


// SocketMessage represents the envelope for all socket communications
// Uses the same structure as Swift/Rust for protocol compatibility
type SocketMessage struct {
	Type    string `json:"type"` // "request" or "response"
	Payload string `json:"payload"` // base64-encoded data
}

// NewJanusRequest creates a new request with generated UUID and timestamp
func NewJanusRequest(request string, args map[string]interface{}, timeout *float64) *JanusRequest {
	// PRIME DIRECTIVE: Use RFC 3339 timestamp format with milliseconds
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return &JanusRequest{
		ID:        uuid.New().String(),
		Method:    request, // PRIME DIRECTIVE: method field matches request name
		Request:   request,
		Args:      args,
		Timeout:   timeout,
		Timestamp: timestamp,
	}
}

// NewSuccessResponse creates a successful response for a request
// PRIME DIRECTIVE: result is unwrapped, request_id references original request, id is unique for response
func NewSuccessResponse(requestID string, result interface{}) *JanusResponse {
	// PRIME DIRECTIVE: Use RFC 3339 timestamp format with milliseconds
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return &JanusResponse{
		Result:    result,
		Error:     nil,
		Success:   true,
		RequestID: requestID,
		ID:        uuid.New().String(),
		Timestamp: timestamp,
	}
}

// NewErrorResponse creates an error response for a request
// PRIME DIRECTIVE: error contains JSONRPCError, result is nil, request_id references original request, id is unique for response
func NewErrorResponse(requestID string, err *JSONRPCError) *JanusResponse {
	// PRIME DIRECTIVE: Use RFC 3339 timestamp format with milliseconds
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return &JanusResponse{
		Result:    nil,
		Error:     err,
		Success:   false,
		RequestID: requestID,
		ID:        uuid.New().String(),
		Timestamp: timestamp,
	}
}


// ToJSON serializes the request to JSON bytes
func (c *JanusRequest) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON deserializes JSON bytes to a request
func (c *JanusRequest) FromJSON(data []byte) error {
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

// RequestHandler represents a function that handles incoming requests
// Matches the Swift RequestHandler signature for compatibility
type RequestHandler func(request *JanusRequest) (*JanusResponse, error)

// TimeoutHandler represents a function called when a request times out
// Matches the Swift TimeoutHandler signature for compatibility
type TimeoutHandler func(requestID string)

// RequestHandle provides a user-friendly interface to track and manage requests
// Hides internal UUID complexity from users
type RequestHandle struct {
	internalID string
	request    string
	timestamp  time.Time
	cancelled  bool
}

// NewRequestHandle creates a new request handle from internal UUID
func NewRequestHandle(internalID, request string) *RequestHandle {
	return &RequestHandle{
		internalID: internalID,
		request:    request,
		timestamp:  time.Now(),
		cancelled:  false,
	}
}

// GetRequest returns the request name for this request
func (h *RequestHandle) GetRequest() string {
	return h.request
}


// GetTimestamp returns when this request was created
func (h *RequestHandle) GetTimestamp() time.Time {
	return h.timestamp
}

// IsCancelled returns whether this request has been cancelled
func (h *RequestHandle) IsCancelled() bool {
	return h.cancelled
}

// GetInternalID returns the internal UUID (for internal use only)
func (h *RequestHandle) GetInternalID() string {
	return h.internalID
}

// MarkCancelled marks this handle as cancelled (internal use only)
func (h *RequestHandle) MarkCancelled() {
	h.cancelled = true
}

// RequestStatus represents the status of a tracked request
type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusCompleted RequestStatus = "completed"
	RequestStatusFailed    RequestStatus = "failed"
	RequestStatusCancelled RequestStatus = "cancelled"
	RequestStatusTimeout   RequestStatus = "timeout"
)