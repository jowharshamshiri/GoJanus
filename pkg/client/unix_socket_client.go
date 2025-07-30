package client

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/user/GoUnixSockAPI/pkg/models"
	"github.com/google/uuid"
)

// UnixSocketClient provides a high-level API for sending commands to Unix socket servers
type UnixSocketClient struct {
	timeout   time.Duration
	channelID string
}

// NewUnixSocketClient creates a new client instance (DEPRECATED: use UnixSocketClient{} directly)
func NewUnixSocketClient(channelID string, timeout time.Duration) *UnixSocketClient {
	if channelID == "" {
		channelID = "default"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	return &UnixSocketClient{
		timeout:   timeout,
		channelID: channelID,
	}
}

// SendCommand sends a command to a server and waits for response
//
// Args:
//   targetSocket - Path to the target server socket
//   command - Command name to execute
//   args - Optional command arguments as key-value pairs
//
// Returns:
//   The server's response
//
// Example:
//   client := &UnixSocketClient{channelID: "default", timeout: 30 * time.Second}
//   args := map[string]interface{}{"message": "Hello"}
//   response, err := client.SendCommand("/tmp/server.sock", "ping", args)
//   if err == nil && response.Success {
//       fmt.Printf("Success: %v\n", response.Result)
//   }
func (c *UnixSocketClient) SendCommand(targetSocket, command string, args map[string]interface{}) (*models.SocketResponse, error) {
	return c.SendCommandWithTimeout(targetSocket, command, args, 0)
}

// SendCommandWithTimeout sends a command with a custom timeout
func (c *UnixSocketClient) SendCommandWithTimeout(targetSocket, command string, args map[string]interface{}, timeout time.Duration) (*models.SocketResponse, error) {
	if timeout == 0 {
		timeout = c.timeout
	}

	// Create command
	timeoutSecs := timeout.Seconds()
	cmd := &models.SocketCommand{
		ID:        uuid.New().String(),
		ChannelID: c.channelID,
		Command:   command,
		ReplyTo:   nil, // Not used in connection-based approach
		Args:      args,
		Timeout:   &timeoutSecs,
		Timestamp: float64(time.Now().Unix()),
	}

	// Connect to target socket
	conn, err := net.DialTimeout("unix", targetSocket, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	// Set timeouts
	conn.SetDeadline(time.Now().Add(timeout))

	// Send command
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var response models.SocketResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &response, nil
}

// SendCommandNoResponse sends a fire-and-forget command (no response expected)
func (c *UnixSocketClient) SendCommandNoResponse(targetSocket, command string, args map[string]interface{}) error {
	// For fire-and-forget, we still need to connect briefly
	timeout := 5 * time.Second
	
	conn, err := net.DialTimeout("unix", targetSocket, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	// Create command without expecting response
	cmd := &models.SocketCommand{
		ID:        uuid.New().String(),
		ChannelID: c.channelID,
		Command:   command,
		ReplyTo:   nil,
		Args:      args,
		Timeout:   nil,
		Timestamp: float64(time.Now().Unix()),
	}

	// Send command
	encoder := json.NewEncoder(conn)
	return encoder.Encode(cmd)
}

// Ping tests connectivity to a server
func (c *UnixSocketClient) Ping(targetSocket string) bool {
	_, err := c.SendCommandWithTimeout(targetSocket, "ping", nil, 5*time.Second)
	return err == nil
}

// SetChannelID sets the default channel ID for this client
func (c *UnixSocketClient) SetChannelID(channelID string) {
	c.channelID = channelID
}

// SetTimeout sets the default timeout for this client
func (c *UnixSocketClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// GetChannelID returns the current channel ID
func (c *UnixSocketClient) GetChannelID() string {
	return c.channelID
}

// GetTimeout returns the current timeout
func (c *UnixSocketClient) GetTimeout() time.Duration {
	return c.timeout
}