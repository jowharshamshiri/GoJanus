package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConnectionPool manages a pool of Unix socket connections for efficient reuse
// Matches Swift connection pooling functionality for performance and resource management
type ConnectionPool struct {
	socketPath        string
	maxConnections    int
	connectionTimeout time.Duration
	maxMessageSize    int
	
	connections       []*UnixSocketClient
	available         []bool
	mutex            sync.Mutex
	validator        *SecurityValidator
}

// ConnectionPoolConfig holds configuration options for the connection pool
// Matches Swift configuration structure
type ConnectionPoolConfig struct {
	MaxConnections    int
	ConnectionTimeout time.Duration
	MaxMessageSize    int
}

// DefaultConnectionPoolConfig returns default configuration matching Swift defaults
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxConnections:    100,                  // Matches Swift default
		ConnectionTimeout: 30 * time.Second,    // Matches Swift default
		MaxMessageSize:    10 * 1024 * 1024,    // 10MB matches Swift
	}
}

// NewConnectionPool creates a new connection pool
// Matches Swift connection pool initialization
func NewConnectionPool(socketPath string, config ...ConnectionPoolConfig) (*ConnectionPool, error) {
	cfg := DefaultConnectionPoolConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	validator := NewSecurityValidator()
	
	// Validate socket path using security validator
	if err := validator.ValidateSocketPath(socketPath); err != nil {
		return nil, fmt.Errorf("invalid socket path for connection pool: %w", err)
	}
	
	// Validate resource limits
	if err := validator.ValidateResourceLimits(cfg.MaxConnections, 0, 0); err != nil {
		return nil, fmt.Errorf("invalid connection pool configuration: %w", err)
	}
	
	return &ConnectionPool{
		socketPath:        socketPath,
		maxConnections:    cfg.MaxConnections,
		connectionTimeout: cfg.ConnectionTimeout,
		maxMessageSize:    cfg.MaxMessageSize,
		connections:       make([]*UnixSocketClient, 0, cfg.MaxConnections),
		available:         make([]bool, 0, cfg.MaxConnections),
		validator:        validator,
	}, nil
}

// BorrowConnection borrows a connection from the pool
// Creates new connection if pool not full, otherwise waits for available connection
// Matches Swift connection borrowing pattern
func (cp *ConnectionPool) BorrowConnection(ctx context.Context) (*UnixSocketClient, int, error) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	// Look for available connection
	for i, isAvailable := range cp.available {
		if isAvailable {
			cp.available[i] = false
			return cp.connections[i], i, nil
		}
	}
	
	// Create new connection if pool not full
	if len(cp.connections) < cp.maxConnections {
		clientConfig := UnixSocketClientConfig{
			MaxMessageSize:    cp.maxMessageSize,
			ConnectionTimeout: cp.connectionTimeout,
		}
		
		client, err := NewUnixSocketClient(cp.socketPath, clientConfig)
		if err != nil {
			return nil, -1, fmt.Errorf("failed to create new connection: %w", err)
		}
		
		// Test connection before adding to pool
		if err := client.TestConnection(ctx); err != nil {
			return nil, -1, fmt.Errorf("connection test failed: %w", err)
		}
		
		index := len(cp.connections)
		cp.connections = append(cp.connections, client)
		cp.available = append(cp.available, false)
		
		return client, index, nil
	}
	
	// Pool is full, return error (Swift behavior)
	return nil, -1, fmt.Errorf("connection pool exhausted: maximum %d connections reached", cp.maxConnections)
}

// ReturnConnection returns a connection to the pool
// Matches Swift connection returning pattern with cleanup
func (cp *ConnectionPool) ReturnConnection(index int) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if index < 0 || index >= len(cp.connections) {
		return fmt.Errorf("invalid connection index: %d", index)
	}
	
	// Mark connection as available
	cp.available[index] = true
	
	return nil
}

// SendMessage sends a message using a pooled connection
// Implements the stateless communication pattern from Swift
func (cp *ConnectionPool) SendMessage(ctx context.Context, data []byte) ([]byte, error) {
	// Borrow connection from pool
	client, index, err := cp.BorrowConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to borrow connection: %w", err)
	}
	
	// Ensure connection is returned to pool
	defer func() {
		if returnErr := cp.ReturnConnection(index); returnErr != nil {
			// Log error but don't override the main error
			// In production, this would use proper logging
		}
	}()
	
	// Send message through the connection
	response, err := client.Send(ctx, data)
	if err != nil {
		// On connection error, remove connection from pool and create new one
		cp.mutex.Lock()
		if index < len(cp.connections) {
			cp.connections[index].Disconnect()
			
			// Create replacement connection
			clientConfig := UnixSocketClientConfig{
				MaxMessageSize:    cp.maxMessageSize,
				ConnectionTimeout: cp.connectionTimeout,
			}
			
			newClient, newErr := NewUnixSocketClient(cp.socketPath, clientConfig)
			if newErr == nil {
				cp.connections[index] = newClient
			}
		}
		cp.mutex.Unlock()
		
		return nil, fmt.Errorf("failed to send message through pool: %w", err)
	}
	
	return response, nil
}

// SendMessageNoResponse sends a message without expecting a response using a pooled connection
// Implements fire-and-forget pattern with connection pooling
func (cp *ConnectionPool) SendMessageNoResponse(ctx context.Context, data []byte) error {
	// Borrow connection from pool
	client, index, err := cp.BorrowConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to borrow connection: %w", err)
	}
	
	// Ensure connection is returned to pool
	defer func() {
		if returnErr := cp.ReturnConnection(index); returnErr != nil {
			// Log error but don't override the main error
			// In production, this would use proper logging
		}
	}()
	
	// Send message through the connection
	err = client.SendNoResponse(ctx, data)
	if err != nil {
		// On connection error, remove connection from pool and create new one
		cp.mutex.Lock()
		if index < len(cp.connections) {
			cp.connections[index].Disconnect()
			
			// Create replacement connection
			clientConfig := UnixSocketClientConfig{
				MaxMessageSize:    cp.maxMessageSize,
				ConnectionTimeout: cp.connectionTimeout,
			}
			
			newClient, newErr := NewUnixSocketClient(cp.socketPath, clientConfig)
			if newErr == nil {
				cp.connections[index] = newClient
			}
		}
		cp.mutex.Unlock()
		
		return fmt.Errorf("failed to send message through pool: %w", err)
	}
	
	return nil
}

// Close closes all connections in the pool
// Matches Swift cleanup and resource management
func (cp *ConnectionPool) Close() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	var lastError error
	
	for i, client := range cp.connections {
		if err := client.Disconnect(); err != nil {
			lastError = err
		}
		cp.available[i] = false
	}
	
	cp.connections = cp.connections[:0]
	cp.available = cp.available[:0]
	
	return lastError
}

// Stats returns connection pool statistics
// Useful for monitoring and debugging (matches Swift resource monitoring)
func (cp *ConnectionPool) Stats() ConnectionPoolStats {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	availableCount := 0
	for _, isAvailable := range cp.available {
		if isAvailable {
			availableCount++
		}
	}
	
	return ConnectionPoolStats{
		TotalConnections:     len(cp.connections),
		AvailableConnections: availableCount,
		BusyConnections:      len(cp.connections) - availableCount,
		MaxConnections:       cp.maxConnections,
	}
}

// ConnectionPoolStats holds statistics about the connection pool
type ConnectionPoolStats struct {
	TotalConnections     int
	AvailableConnections int
	BusyConnections      int
	MaxConnections       int
}

// TestAllConnections tests all connections in the pool
// Useful for health checks and monitoring
func (cp *ConnectionPool) TestAllConnections(ctx context.Context) error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	for i, client := range cp.connections {
		if err := client.TestConnection(ctx); err != nil {
			return fmt.Errorf("connection %d test failed: %w", i, err)
		}
	}
	
	return nil
}

// MaxConnections returns the maximum number of connections (read-only property)
// Matches Swift configuration access pattern
func (cp *ConnectionPool) MaxConnections() int {
	return cp.maxConnections
}

// SocketPath returns the socket path (read-only property)
func (cp *ConnectionPool) SocketPath() string {
	return cp.socketPath
}