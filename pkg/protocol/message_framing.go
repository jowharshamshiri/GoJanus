package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"GoJanus/pkg/models"
)

const (
	// LengthPrefixSize is the size of the 4-byte big-endian length prefix
	LengthPrefixSize = 4
	// MaxMessageSize is the maximum allowed message size (10MB default)
	MaxMessageSize = 10 * 1024 * 1024
)


// MessageFraming provides message framing functionality with 4-byte length prefix
type MessageFraming struct{}

// SocketMessage represents the message envelope for framing
type SocketMessage struct {
	Type    string `json:"type"`    // "request" or "response"
	Payload string `json:"payload"` // Base64 encoded payload
}

// EncodeMessage encodes a message with 4-byte big-endian length prefix
func (mf *MessageFraming) EncodeMessage(message interface{}) ([]byte, error) {
	// Determine message type
	var messageType string
	switch message.(type) {
	case models.JanusRequest, *models.JanusRequest:
		messageType = "request"
	case models.JanusResponse, *models.JanusResponse:
		messageType = "response"
	default:
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Invalid message type",
			Data:    &models.JSONRPCErrorData{Details: "INVALID_MESSAGE_TYPE"},
		}
	}

	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Failed to marshal payload: %v", err),
			Data:    &models.JSONRPCErrorData{Details: "MARSHAL_FAILED"},
		}
	}

	// Create envelope with base64 payload
	envelope := SocketMessage{
		Type:    messageType,
		Payload: string(payloadBytes), // Direct JSON for efficiency
	}

	// Serialize envelope to JSON
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Failed to marshal envelope: %v", err),
			Data:    &models.JSONRPCErrorData{Details: "ENVELOPE_MARSHAL_FAILED"},
		}
	}

	// Validate message size
	if len(envelopeBytes) > MaxMessageSize {
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Message size %d exceeds maximum %d", len(envelopeBytes), MaxMessageSize),
			Data:    &models.JSONRPCErrorData{Details: "MESSAGE_TOO_LARGE"},
		}
	}

	// Create length prefix (4-byte big-endian)
	lengthBuffer := make([]byte, LengthPrefixSize)
	binary.BigEndian.PutUint32(lengthBuffer, uint32(len(envelopeBytes)))

	// Combine length prefix and message
	result := make([]byte, 0, LengthPrefixSize+len(envelopeBytes))
	result = append(result, lengthBuffer...)
	result = append(result, envelopeBytes...)

	return result, nil
}

// DecodeMessage decodes a message from buffer with length prefix
func (mf *MessageFraming) DecodeMessage(buffer []byte) (interface{}, []byte, error) {
	// Check if we have at least the length prefix
	if len(buffer) < LengthPrefixSize {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Buffer too small for length prefix: %d < %d", len(buffer), LengthPrefixSize),
			Data:    &models.JSONRPCErrorData{Details: "INCOMPLETE_LENGTH_PREFIX"},
		}
	}

	// Read message length from big-endian prefix
	messageLength := binary.BigEndian.Uint32(buffer[:LengthPrefixSize])

	// Validate message length
	if messageLength > MaxMessageSize {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Message length %d exceeds maximum %d", messageLength, MaxMessageSize),
			Data:    &models.JSONRPCErrorData{Details: "MESSAGE_TOO_LARGE"},
		}
	}

	if messageLength == 0 {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Message length cannot be zero",
			Data:    &models.JSONRPCErrorData{Details: "ZERO_LENGTH_MESSAGE"},
		}
	}

	// Check if we have the complete message
	totalRequired := LengthPrefixSize + int(messageLength)
	if len(buffer) < totalRequired {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Buffer too small for complete message: %d < %d", len(buffer), totalRequired),
			Data:    &models.JSONRPCErrorData{Details: "INCOMPLETE_MESSAGE"},
		}
	}

	// Extract message data
	messageBuffer := buffer[LengthPrefixSize : LengthPrefixSize+int(messageLength)]
	remainingBuffer := buffer[LengthPrefixSize+int(messageLength):]

	// Parse JSON envelope
	var envelope SocketMessage
	if err := json.Unmarshal(messageBuffer, &envelope); err != nil {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Failed to parse message envelope JSON: %v", err),
			Data:    &models.JSONRPCErrorData{Details: "INVALID_JSON_ENVELOPE"},
		}
	}

	// Validate envelope structure
	if envelope.Type == "" || envelope.Payload == "" {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Message envelope missing required fields (type, payload)",
			Data:    &models.JSONRPCErrorData{Details: "MISSING_ENVELOPE_FIELDS"},
		}
	}

	if envelope.Type != "request" && envelope.Type != "response" {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Invalid message type: %s", envelope.Type),
			Data:    &models.JSONRPCErrorData{Details: "INVALID_MESSAGE_TYPE"},
		}
	}

	// Parse payload JSON directly (no base64 decoding needed)
	var message interface{}
	if envelope.Type == "request" {
		var cmd models.JanusRequest
		if err := json.Unmarshal([]byte(envelope.Payload), &cmd); err != nil {
			return nil, buffer, &models.JSONRPCError{
				Code:    models.MessageFramingError,
				Message: fmt.Sprintf("Failed to parse request payload JSON: %v", err),
				Data:    &models.JSONRPCErrorData{Details: "INVALID_PAYLOAD_JSON"},
			}
		}
		
		// Validate request structure
		if err := mf.validateRequestStructure(&cmd); err != nil {
			return nil, buffer, err
		}
		message = cmd
	} else {
		var resp models.JanusResponse
		if err := json.Unmarshal([]byte(envelope.Payload), &resp); err != nil {
			return nil, buffer, &models.JSONRPCError{
				Code:    models.MessageFramingError,
				Message: fmt.Sprintf("Failed to parse response payload JSON: %v", err),
				Data:    &models.JSONRPCErrorData{Details: "INVALID_PAYLOAD_JSON"},
			}
		}
		
		// Validate response structure
		if err := mf.validateResponseStructure(&resp); err != nil {
			return nil, buffer, err
		}
		message = resp
	}

	return message, remainingBuffer, nil
}

// ExtractMessages extracts complete messages from a buffer, handling partial messages
func (mf *MessageFraming) ExtractMessages(buffer []byte) ([]interface{}, []byte, error) {
	var messages []interface{}
	currentBuffer := buffer

	for len(currentBuffer) > 0 {
		message, remainingBuffer, err := mf.DecodeMessage(currentBuffer)
		if err != nil {
			if jsonErr, ok := err.(*models.JSONRPCError); ok && jsonErr.Code == models.MessageFramingError {
				if jsonErr.Data != nil && (jsonErr.Data.Details == "INCOMPLETE_LENGTH_PREFIX" || jsonErr.Data.Details == "INCOMPLETE_MESSAGE") {
					// Not enough data for complete message, save remaining buffer
					break
				}
			}
			return nil, buffer, err
		}

		messages = append(messages, message)
		currentBuffer = remainingBuffer
	}

	return messages, currentBuffer, nil
}

// CalculateFramedSize calculates the total size needed for a message when framed
func (mf *MessageFraming) CalculateFramedSize(message interface{}) (int, error) {
	encoded, err := mf.EncodeMessage(message)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}

// EncodeDirectMessage creates a direct JSON message for simple cases (without envelope)
func (mf *MessageFraming) EncodeDirectMessage(message interface{}) ([]byte, error) {
	// Serialize message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Failed to marshal message: %v", err),
			Data:    &models.JSONRPCErrorData{Details: "MARSHAL_FAILED"},
		}
	}

	// Validate message size
	if len(messageBytes) > MaxMessageSize {
		return nil, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Message size %d exceeds maximum %d", len(messageBytes), MaxMessageSize),
			Data:    &models.JSONRPCErrorData{Details: "MESSAGE_TOO_LARGE"},
		}
	}

	// Create length prefix
	lengthBuffer := make([]byte, LengthPrefixSize)
	binary.BigEndian.PutUint32(lengthBuffer, uint32(len(messageBytes)))

	// Combine length prefix and message
	result := make([]byte, 0, LengthPrefixSize+len(messageBytes))
	result = append(result, lengthBuffer...)
	result = append(result, messageBytes...)

	return result, nil
}

// DecodeDirectMessage decodes a direct JSON message (without envelope)
func (mf *MessageFraming) DecodeDirectMessage(buffer []byte) (interface{}, []byte, error) {
	// Check length prefix
	if len(buffer) < LengthPrefixSize {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Buffer too small for length prefix: %d < %d", len(buffer), LengthPrefixSize),
			Data:    &models.JSONRPCErrorData{Details: "INCOMPLETE_LENGTH_PREFIX"},
		}
	}

	messageLength := binary.BigEndian.Uint32(buffer[:LengthPrefixSize])
	totalRequired := LengthPrefixSize + int(messageLength)

	if len(buffer) < totalRequired {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Buffer too small for complete message: %d < %d", len(buffer), totalRequired),
			Data:    &models.JSONRPCErrorData{Details: "INCOMPLETE_MESSAGE"},
		}
	}

	// Extract and parse message
	messageBuffer := buffer[LengthPrefixSize : LengthPrefixSize+int(messageLength)]
	remainingBuffer := buffer[LengthPrefixSize+int(messageLength):]

	// Try to determine message type by looking for key fields
	var rawMessage map[string]interface{}
	if err := json.Unmarshal(messageBuffer, &rawMessage); err != nil {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: fmt.Sprintf("Failed to parse message JSON: %v", err),
			Data:    &models.JSONRPCErrorData{Details: "INVALID_JSON"},
		}
	}

	// Determine message type and parse accordingly
	var message interface{}
	if _, hasRequest := rawMessage["request"]; hasRequest {
		var cmd models.JanusRequest
		if err := json.Unmarshal(messageBuffer, &cmd); err != nil {
			return nil, buffer, &models.JSONRPCError{
				Code:    models.MessageFramingError,
				Message: fmt.Sprintf("Failed to parse request: %v", err),
				Data:    &models.JSONRPCErrorData{Details: "INVALID_REQUEST"},
			}
		}
		message = cmd
	} else if _, hasRequestId := rawMessage["requestId"]; hasRequestId {
		var resp models.JanusResponse
		if err := json.Unmarshal(messageBuffer, &resp); err != nil {
			return nil, buffer, &models.JSONRPCError{
				Code:    models.MessageFramingError,
				Message: fmt.Sprintf("Failed to parse response: %v", err),
				Data:    &models.JSONRPCErrorData{Details: "INVALID_RESPONSE"},
			}
		}
		message = resp
	} else {
		return nil, buffer, &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Cannot determine message type",
			Data:    &models.JSONRPCErrorData{Details: "UNKNOWN_MESSAGE_TYPE"},
		}
	}

	return message, remainingBuffer, nil
}

// validateRequestStructure validates request structure
func (mf *MessageFraming) validateRequestStructure(cmd *models.JanusRequest) error {
	if cmd.ID == "" {
		return &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Request missing required string field: id",
			Data:    &models.JSONRPCErrorData{Details: "MISSING_REQUEST_FIELD"},
		}
	}
	if cmd.Request == "" {
		return &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Request missing required string field: request",
			Data:    &models.JSONRPCErrorData{Details: "MISSING_REQUEST_FIELD"},
		}
	}
	return nil
}

// validateResponseStructure validates response structure
func (mf *MessageFraming) validateResponseStructure(resp *models.JanusResponse) error {
	if resp.RequestID == "" {
		return &models.JSONRPCError{
			Code:    models.MessageFramingError,
			Message: "Response missing required field: requestId",
			Data:    &models.JSONRPCErrorData{Details: "MISSING_RESPONSE_FIELD"},
		}
	}
	// PRIME DIRECTIVE: Response no longer includes channelId field
	return nil
}

// NewMessageFraming creates a new MessageFraming instance
func NewMessageFraming() *MessageFraming {
	return &MessageFraming{}
}