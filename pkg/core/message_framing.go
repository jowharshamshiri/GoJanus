package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// MessageFraming handles the 4-byte big-endian length prefix protocol
// Matches the Swift/Rust message framing exactly for cross-language compatibility
type MessageFraming struct {
	maxMessageSize int
}

// NewMessageFraming creates a new message framing handler
func NewMessageFraming(maxMessageSize int) *MessageFraming {
	return &MessageFraming{
		maxMessageSize: maxMessageSize,
	}
}

// WriteMessage writes a message with 4-byte big-endian length prefix
// Matches Swift: let length_bytes = (message_len as u32).to_be_bytes()
func (mf *MessageFraming) WriteMessage(conn net.Conn, message []byte) error {
	messageLen := len(message)
	
	// Check message size limit (matches Swift security validation)
	if messageLen > mf.maxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum allowed size %d", messageLen, mf.maxMessageSize)
	}
	
	// Write 4-byte big-endian length prefix
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(messageLen))
	
	if _, err := conn.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}
	
	// Write message payload
	if _, err := conn.Write(message); err != nil {
		return fmt.Errorf("failed to write message payload: %w", err)
	}
	
	return nil
}

// ReadMessage reads a message with 4-byte big-endian length prefix
// Matches Swift: let length = UInt32(bigEndian: lengthData.withUnsafeBytes { $0.load(as: UInt32.self) })
func (mf *MessageFraming) ReadMessage(conn net.Conn) ([]byte, error) {
	// Read 4-byte length prefix
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBytes); err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}
	
	// Parse big-endian length
	messageLen := binary.BigEndian.Uint32(lengthBytes)
	
	// Validate message size (matches Swift security validation)
	if int(messageLen) > mf.maxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum allowed size %d", messageLen, mf.maxMessageSize)
	}
	
	if messageLen == 0 {
		return []byte{}, nil
	}
	
	// Read message payload
	message := make([]byte, messageLen)
	if _, err := io.ReadFull(conn, message); err != nil {
		return nil, fmt.Errorf("failed to read message payload: %w", err)
	}
	
	return message, nil
}

// ValidateMessageFormat performs additional validation on message content
// Matches Swift security validation requirements
func (mf *MessageFraming) ValidateMessageFormat(message []byte) error {
	// Check for null bytes (security validation from Swift)
	for i, b := range message {
		if b == 0 {
			return fmt.Errorf("null byte detected at position %d", i)
		}
	}
	
	// Validate UTF-8 encoding (matches Swift UTF-8 validation)
	if !isValidUTF8(message) {
		return fmt.Errorf("message contains invalid UTF-8 sequences")
	}
	
	return nil
}

// isValidUTF8 checks if the byte slice contains valid UTF-8
// Matches Swift's UTF-8 validation for security
func isValidUTF8(data []byte) bool {
	// Go's built-in UTF-8 validation
	for len(data) > 0 {
		r, size := decodeRuneUTF8(data)
		if r == '\uFFFD' && size == 1 {
			return false
		}
		data = data[size:]
	}
	return true
}

// decodeRuneUTF8 is a simplified UTF-8 decoder for validation
func decodeRuneUTF8(data []byte) (rune, int) {
	if len(data) == 0 {
		return '\uFFFD', 0
	}
	
	b0 := data[0]
	
	// ASCII case
	if b0 < 0x80 {
		return rune(b0), 1
	}
	
	// Multi-byte cases - simplified validation
	if b0 < 0xC0 {
		return '\uFFFD', 1
	}
	
	if b0 < 0xE0 {
		if len(data) < 2 {
			return '\uFFFD', 1
		}
		b1 := data[1]
		if b1 < 0x80 || b1 >= 0xC0 {
			return '\uFFFD', 1
		}
		r := rune(b0&0x1F)<<6 | rune(b1&0x3F)
		if r < 0x80 {
			return '\uFFFD', 1
		}
		return r, 2
	}
	
	if b0 < 0xF0 {
		if len(data) < 3 {
			return '\uFFFD', 1
		}
		b1, b2 := data[1], data[2]
		if b1 < 0x80 || b1 >= 0xC0 || b2 < 0x80 || b2 >= 0xC0 {
			return '\uFFFD', 1
		}
		r := rune(b0&0x0F)<<12 | rune(b1&0x3F)<<6 | rune(b2&0x3F)
		if r < 0x800 {
			return '\uFFFD', 1
		}
		return r, 3
	}
	
	if b0 < 0xF8 {
		if len(data) < 4 {
			return '\uFFFD', 1
		}
		b1, b2, b3 := data[1], data[2], data[3]
		if b1 < 0x80 || b1 >= 0xC0 || b2 < 0x80 || b2 >= 0xC0 || b3 < 0x80 || b3 >= 0xC0 {
			return '\uFFFD', 1
		}
		r := rune(b0&0x07)<<18 | rune(b1&0x3F)<<12 | rune(b2&0x3F)<<6 | rune(b3&0x3F)
		if r < 0x10000 || r > 0x10FFFF {
			return '\uFFFD', 1
		}
		return r, 4
	}
	
	return '\uFFFD', 1
}