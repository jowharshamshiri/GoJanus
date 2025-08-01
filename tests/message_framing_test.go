package tests

import (
	"encoding/json"
	"testing"

	"github.com/user/GoJanus/pkg/models"
	"github.com/user/GoJanus/pkg/protocol"
)

func TestMessageFraming_EncodeMessage(t *testing.T) {
	framing := protocol.NewMessageFraming()
	
	t.Run("should encode a command message", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		encoded, err := framing.EncodeMessage(command)
		if err != nil {
			t.Fatalf("Failed to encode command: %v", err)
		}
		
		if len(encoded) <= protocol.LengthPrefixSize {
			t.Errorf("Encoded message too short: %d", len(encoded))
		}
		
		// Check length prefix (first 4 bytes)
		expectedLength := len(encoded) - protocol.LengthPrefixSize
		actualLength := int(encoded[0])<<24 | int(encoded[1])<<16 | int(encoded[2])<<8 | int(encoded[3])
		if actualLength != expectedLength {
			t.Errorf("Length prefix mismatch: expected %d, got %d", expectedLength, actualLength)
		}
	})
	
	t.Run("should encode a response message", func(t *testing.T) {
		response := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded, err := framing.EncodeMessage(response)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
		
		if len(encoded) <= protocol.LengthPrefixSize {
			t.Errorf("Encoded message too short: %d", len(encoded))
		}
	})
	
	t.Run("should reject messages that are too large", func(t *testing.T) {
		// Create a command with very large args
		largeArgs := make(map[string]interface{})
		largeArgs["data"] = string(make([]byte, 20*1024*1024)) // 20MB
		
		command := models.NewJanusCommand("test-service", "large", largeArgs, nil)
		
		_, err := framing.EncodeMessage(command)
		if err == nil {
			t.Error("Expected error for oversized message")
		}
		
		// Validate error code instead of error message details
		if jsonErr, ok := err.(*models.JSONRPCError); ok {
			if jsonErr.Code != models.MessageFramingError {
				t.Errorf("Expected MessageFramingError code (%d), got %d", models.MessageFramingError, jsonErr.Code)
			}
		} else {
			t.Error("Expected JSONRPCError with MessageFramingError code")
		}
	})
}

func TestMessageFraming_DecodeMessage(t *testing.T) {
	framing := protocol.NewMessageFraming()
	
	t.Run("should decode a command message", func(t *testing.T) {
		originalCommand := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		encoded, err := framing.EncodeMessage(originalCommand)
		if err != nil {
			t.Fatalf("Failed to encode command: %v", err)
		}
		
		decoded, remaining, err := framing.DecodeMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode command: %v", err)
		}
		
		if len(remaining) != 0 {
			t.Errorf("Expected no remaining buffer, got %d bytes", len(remaining))
		}
		
		decodedCommand, ok := decoded.(models.JanusCommand)
		if !ok {
			t.Fatalf("Decoded message is not a JanusCommand")
		}
		
		if decodedCommand.ID != originalCommand.ID {
			t.Errorf("Command ID mismatch: expected %s, got %s", originalCommand.ID, decodedCommand.ID)
		}
		
		if decodedCommand.ChannelID != originalCommand.ChannelID {
			t.Errorf("Channel ID mismatch: expected %s, got %s", originalCommand.ChannelID, decodedCommand.ChannelID)
		}
		
		if decodedCommand.Command != originalCommand.Command {
			t.Errorf("Command mismatch: expected %s, got %s", originalCommand.Command, decodedCommand.Command)
		}
	})
	
	t.Run("should decode a response message", func(t *testing.T) {
		originalResponse := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded, err := framing.EncodeMessage(originalResponse)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
		
		decoded, remaining, err := framing.DecodeMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if len(remaining) != 0 {
			t.Errorf("Expected no remaining buffer, got %d bytes", len(remaining))
		}
		
		decodedResponse, ok := decoded.(models.JanusResponse)
		if !ok {
			t.Fatalf("Decoded message is not a JanusResponse")
		}
		
		if decodedResponse.CommandID != originalResponse.CommandID {
			t.Errorf("Command ID mismatch: expected %s, got %s", originalResponse.CommandID, decodedResponse.CommandID)
		}
		
		if decodedResponse.Success != originalResponse.Success {
			t.Errorf("Success mismatch: expected %v, got %v", originalResponse.Success, decodedResponse.Success)
		}
	})
	
	t.Run("should handle multiple messages in buffer", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		response := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded1, err := framing.EncodeMessage(command)
		if err != nil {
			t.Fatalf("Failed to encode command: %v", err)
		}
		
		encoded2, err := framing.EncodeMessage(response)
		if err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
		
		combined := append(encoded1, encoded2...)
		
		// Extract first message
		message1, remaining, err := framing.DecodeMessage(combined)
		if err != nil {
			t.Fatalf("Failed to decode first message: %v", err)
		}
		
		if _, ok := message1.(models.JanusCommand); !ok {
			t.Error("First message should be a command")
		}
		
		// Extract second message
		message2, finalRemaining, err := framing.DecodeMessage(remaining)
		if err != nil {
			t.Fatalf("Failed to decode second message: %v", err)
		}
		
		if _, ok := message2.(models.JanusResponse); !ok {
			t.Error("Second message should be a response")
		}
		
		if len(finalRemaining) != 0 {
			t.Errorf("Expected no final remaining buffer, got %d bytes", len(finalRemaining))
		}
	})
	
	t.Run("should return error for incomplete length prefix", func(t *testing.T) {
		shortBuffer := []byte{0x00, 0x00} // Only 2 bytes
		
		_, _, err := framing.DecodeMessage(shortBuffer)
		if err == nil {
			t.Error("Expected error for incomplete length prefix")
		}
		
		// Validate error code instead of error message details
		if jsonErr, ok := err.(*models.JSONRPCError); ok {
			if jsonErr.Code != models.MessageFramingError {
				t.Errorf("Expected MessageFramingError code (%d), got %d", models.MessageFramingError, jsonErr.Code)
			}
		} else {
			t.Error("Expected JSONRPCError with MessageFramingError code")
		}
	})
	
	t.Run("should return error for incomplete message", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		encoded, err := framing.EncodeMessage(command)
		if err != nil {
			t.Fatalf("Failed to encode command: %v", err)
		}
		
		// Truncate the message
		truncated := encoded[:len(encoded)-10]
		
		_, _, err = framing.DecodeMessage(truncated)
		if err == nil {
			t.Error("Expected error for incomplete message")
		}
		
		// Validate error code instead of error message details
		if jsonErr, ok := err.(*models.JSONRPCError); ok {
			if jsonErr.Code != models.MessageFramingError {
				t.Errorf("Expected MessageFramingError code (%d), got %d", models.MessageFramingError, jsonErr.Code)
			}
		} else {
			t.Error("Expected JSONRPCError with MessageFramingError code")
		}
	})
	
	t.Run("should return error for zero-length message", func(t *testing.T) {
		zeroLengthBuffer := []byte{0x00, 0x00, 0x00, 0x00} // 0 length
		
		_, _, err := framing.DecodeMessage(zeroLengthBuffer)
		if err == nil {
			t.Error("Expected error for zero-length message")
		}
		
		// Validate error code instead of error message details
		if jsonErr, ok := err.(*models.JSONRPCError); ok {
			if jsonErr.Code != models.MessageFramingError {
				t.Errorf("Expected MessageFramingError code (%d), got %d", models.MessageFramingError, jsonErr.Code)
			}
		} else {
			t.Error("Expected JSONRPCError with MessageFramingError code")
		}
	})
}

func TestMessageFraming_ExtractMessages(t *testing.T) {
	framing := protocol.NewMessageFraming()
	
	t.Run("should extract multiple complete messages", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		response := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded1, _ := framing.EncodeMessage(command)
		encoded2, _ := framing.EncodeMessage(response)
		combined := append(encoded1, encoded2...)
		
		messages, remaining, err := framing.ExtractMessages(combined)
		if err != nil {
			t.Fatalf("Failed to extract messages: %v", err)
		}
		
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
		
		if len(remaining) != 0 {
			t.Errorf("Expected no remaining buffer, got %d bytes", len(remaining))
		}
		
		if _, ok := messages[0].(models.JanusCommand); !ok {
			t.Error("First message should be a command")
		}
		
		if _, ok := messages[1].(models.JanusResponse); !ok {
			t.Error("Second message should be a response")
		}
	})
	
	t.Run("should handle partial messages", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		response := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded1, _ := framing.EncodeMessage(command)
		encoded2, _ := framing.EncodeMessage(response)
		combined := append(encoded1, encoded2...)
		
		// Take only part of the second message
		partial := combined[:len(encoded1)+10]
		
		messages, remaining, err := framing.ExtractMessages(partial)
		if err != nil {
			t.Fatalf("Failed to extract messages: %v", err)
		}
		
		if len(messages) != 1 {
			t.Errorf("Expected 1 complete message, got %d", len(messages))
		}
		
		if len(remaining) != 10 {
			t.Errorf("Expected 10 bytes remaining (partial second message), got %d", len(remaining))
		}
		
		if _, ok := messages[0].(models.JanusCommand); !ok {
			t.Error("First message should be a command")
		}
	})
	
	t.Run("should handle empty buffer", func(t *testing.T) {
		messages, remaining, err := framing.ExtractMessages([]byte{})
		if err != nil {
			t.Fatalf("Failed to extract from empty buffer: %v", err)
		}
		
		if len(messages) != 0 {
			t.Errorf("Expected no messages, got %d", len(messages))
		}
		
		if len(remaining) != 0 {
			t.Errorf("Expected no remaining buffer, got %d bytes", len(remaining))
		}
	})
	
	t.Run("should handle buffer with only partial length prefix", func(t *testing.T) {
		partial := []byte{0x00, 0x00} // Incomplete length prefix
		
		messages, remaining, err := framing.ExtractMessages(partial)
		if err != nil {
			t.Fatalf("Failed to extract from partial buffer: %v", err)
		}
		
		if len(messages) != 0 {
			t.Errorf("Expected no messages, got %d", len(messages))
		}
		
		if len(remaining) != 2 {
			t.Errorf("Expected 2 bytes remaining, got %d", len(remaining))
		}
	})
}

func TestMessageFraming_CalculateFramedSize(t *testing.T) {
	framing := protocol.NewMessageFraming()
	
	t.Run("should calculate correct framed size", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		size, err := framing.CalculateFramedSize(command)
		if err != nil {
			t.Fatalf("Failed to calculate framed size: %v", err)
		}
		
		encoded, err := framing.EncodeMessage(command)
		if err != nil {
			t.Fatalf("Failed to encode message: %v", err)
		}
		
		if size != len(encoded) {
			t.Errorf("Size mismatch: calculated %d, actual %d", size, len(encoded))
		}
	})
}

func TestMessageFraming_DirectMessage(t *testing.T) {
	framing := protocol.NewMessageFraming()
	
	t.Run("should encode direct message without envelope", func(t *testing.T) {
		command := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		encoded, err := framing.EncodeDirectMessage(command)
		if err != nil {
			t.Fatalf("Failed to encode direct message: %v", err)
		}
		
		if len(encoded) <= protocol.LengthPrefixSize {
			t.Errorf("Encoded direct message too short: %d", len(encoded))
		}
		
		// Should be smaller than envelope version (no envelope overhead)
		envelopeEncoded, _ := framing.EncodeMessage(command)
		if len(encoded) >= len(envelopeEncoded) {
			t.Errorf("Direct message should be smaller than envelope message: %d >= %d", len(encoded), len(envelopeEncoded))
		}
	})
	
	t.Run("should decode direct message without envelope", func(t *testing.T) {
		originalCommand := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		encoded, err := framing.EncodeDirectMessage(originalCommand)
		if err != nil {
			t.Fatalf("Failed to encode direct message: %v", err)
		}
		
		decoded, remaining, err := framing.DecodeDirectMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode direct message: %v", err)
		}
		
		if len(remaining) != 0 {
			t.Errorf("Expected no remaining buffer, got %d bytes", len(remaining))
		}
		
		decodedCommand, ok := decoded.(models.JanusCommand)
		if !ok {
			t.Fatalf("Decoded message is not a JanusCommand")
		}
		
		if decodedCommand.ID != originalCommand.ID {
			t.Errorf("Command ID mismatch: expected %s, got %s", originalCommand.ID, decodedCommand.ID)
		}
	})
	
	t.Run("should roundtrip command through direct encoding", func(t *testing.T) {
		originalCommand := models.NewJanusCommand("test-service", "ping", nil, nil)
		
		encoded, err := framing.EncodeDirectMessage(originalCommand)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		
		decoded, _, err := framing.DecodeDirectMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
		
		decodedCommand := decoded.(models.JanusCommand)
		
		// Compare JSON representations for deep equality
		originalJSON, _ := json.Marshal(originalCommand)
		decodedJSON, _ := json.Marshal(decodedCommand)
		
		if string(originalJSON) != string(decodedJSON) {
			t.Errorf("Command roundtrip failed:\nOriginal: %s\nDecoded:  %s", originalJSON, decodedJSON)
		}
	})
	
	t.Run("should roundtrip response through direct encoding", func(t *testing.T) {
		originalResponse := &models.JanusResponse{
			CommandID: "550e8400-e29b-41d4-a716-446655440000",
			ChannelID: "test-service",
			Success:   true,
			Result:    map[string]interface{}{"pong": true},
			Timestamp: 1722248201,
		}
		
		encoded, err := framing.EncodeDirectMessage(originalResponse)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		
		decoded, _, err := framing.DecodeDirectMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
		
		decodedResponse := decoded.(models.JanusResponse)
		
		// Compare JSON representations for deep equality
		originalJSON, _ := json.Marshal(originalResponse)
		decodedJSON, _ := json.Marshal(decodedResponse)
		
		if string(originalJSON) != string(decodedJSON) {
			t.Errorf("Response roundtrip failed:\nOriginal: %s\nDecoded:  %s", originalJSON, decodedJSON)
		}
	})
}