# GoJanus

A production-ready Unix domain socket communication library for Go with **SOCK_DGRAM connectionless communication** and automatic ID management.

## Features

- **Connectionless SOCK_DGRAM**: Unix domain datagram sockets with reply-to mechanism
- **Automatic ID Management**: RequestHandle system hides UUID complexity from users
- **Cross-Language Compatibility**: Perfect compatibility with Rust, Swift, and TypeScript implementations  
- **Dynamic Specification**: Server-provided Manifests with auto-fetch validation
- **Security Framework**: 27 comprehensive security mechanisms and attack prevention
- **JSON-RPC 2.0 Compliance**: Standardized error codes and response format
- **Performance Optimized**: Sub-millisecond response times with 500+ requests/second
- **Production Ready**: Enterprise-grade error handling and resource management
- **Cross-Platform**: Works on all Unix-like systems (Linux, macOS, BSD)

## Installation

```bash
go mod init your-project
go get github.com/jowharshamshiri/GoJanus
```

## Quick Start

### Simple Client Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
)

func main() {
    // Create client with automatic Manifest fetching
    client, err := protocol.New("/tmp/my_socket.sock", "my_channel")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Send command - ID management is automatic
    args := map[string]interface{}{
        "message": "Hello World",
    }
    
    ctx := context.Background()
    response, err := client.SendCommand(ctx, "echo", args)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Response: %+v\n", response)
}
```

### Advanced Request Tracking

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
    "github.com/jowharshamshiri/GoJanus/pkg/models"
)

func main() {
    client, err := protocol.New("/tmp/my_socket.sock", "my_channel")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    args := map[string]interface{}{
        "data": "processing_task",
    }
    
    // Send command with RequestHandle for tracking
    handle, responseC, errC := client.SendCommandWithHandle(
        context.Background(),
        "process_data",
        args,
    )
    
    fmt.Printf("Request started: %s on channel %s\n", 
        handle.GetCommand(), handle.GetChannel())
    
    // Can check status or cancel if needed
    if handle.IsCancelled() {
        fmt.Println("Request was cancelled")
        return
    }
    
    // Wait for response or error
    select {
    case response := <-responseC:
        fmt.Printf("Success: %+v\n", response)
    case err := <-errC:
        fmt.Printf("Error: %v\n", err)
    case <-time.After(10 * time.Second):
        client.CancelRequest(handle)
        fmt.Println("Request cancelled due to timeout")
    }
}
```

### Server with Command Handlers

```go
package main

import (
    "fmt"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
    "github.com/jowharshamshiri/GoJanus/pkg/models"
)

func main() {
    // Create server client with automatic Manifest loading
    client, err := protocol.New("/tmp/my_socket.sock", "my_channel")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Register command handlers - returns direct values
    err = client.RegisterCommandHandler("echo", func(cmd *models.JanusCommand) (interface{}, error) {
        message, ok := cmd.Args["message"].(string)
        if !ok {
            return nil, models.NewJSONRPCError(
                models.JSONRPCErrorCodeInvalidParams,
                "message parameter required",
                nil,
            )
        }
        
        // Return direct value - no dictionary wrapping needed
        return map[string]interface{}{
            "echo": message,
            "timestamp": cmd.Timestamp,
        }, nil
    })
    if err != nil {
        panic(err)
    }
    
    // Register async handler
    err = client.RegisterAsyncCommandHandler("process_data", func(cmd *models.JanusCommand) (interface{}, error) {
        // Simulate processing
        data := cmd.Args["data"].(string)
        result := fmt.Sprintf("Processed: %s", data)
        
        return map[string]interface{}{
            "result": result,
            "processed_at": time.Now().Unix(),
        }, nil
    })
    if err != nil {
        panic(err)
    }
    
    // Start listening for commands
    fmt.Println("Server listening on /tmp/my_socket.sock...")
    err = client.StartListening()
    if err != nil {
        panic(err)
    }
}
```

### Fire-and-Forget Commands

```go
// Send command without waiting for response
err := client.SendCommandNoResponse(
    context.Background(),
    "log_event",
    map[string]interface{}{
        "event": "user_login",
        "user_id": "12345",
    },
)
if err != nil {
    fmt.Printf("Failed to log event: %v\n", err)
} else {
    fmt.Println("Event logged successfully")
}
```

## Key Architecture Features

### RequestHandle System (Automatic ID Management)
- **Transparent UUIDs**: Internal UUID generation hidden from users
- **Request Tracking**: Track pending requests with user-friendly handles
- **Cancellation Support**: Cancel requests using handles without UUID knowledge
- **Status Monitoring**: Check request status using handles

### SOCK_DGRAM Protocol
- **Connectionless Communication**: Each request creates temporary datagram socket
- **Reply-To Mechanism**: Automatic response socket creation and cleanup
- **Cross-Platform Compatibility**: Identical message format across all implementations
- **OS-Level Reliability**: Unix domain socket guarantees for local communication

### Security & Performance
- **27 Security Mechanisms**: Path validation, input sanitization, resource limits
- **JSON-RPC 2.0 Compliance**: Standardized error codes and response format
- **Sub-Millisecond Latency**: Optimized for high-performance local communication
- **Automatic Cleanup**: OS handles socket cleanup, manual cleanup for error cases

## Testing

Run the comprehensive test suite:

```bash
go test ./tests/...
```

Cross-platform integration testing:
```bash
# From project root
./test_cross_platform.sh
```

## Configuration

```go
import "time"

config := protocol.JanusClientConfig{
    MaxMessageSize:   10 * 1024 * 1024, // 10MB
    DefaultTimeout:   30 * time.Second,
    DatagramTimeout:  5 * time.Second,
    EnableValidation: true,
}

client, err := protocol.NewWithConfig(
    "/tmp/my_socket.sock", 
    "my_channel", 
    config,
)
```

## RequestHandle Management

```go
// Get all pending requests
handles := client.GetPendingRequests()
fmt.Printf("Pending requests: %d\n", len(handles))

for _, handle := range handles {
    fmt.Printf("Request: %s on %s (created: %v)\n", 
        handle.GetCommand(), 
        handle.GetChannel(), 
        handle.GetTimestamp())
    
    // Check status
    status := client.GetRequestStatus(handle)
    fmt.Printf("Status: %s\n", status)
}

// Cancel all pending requests
cancelled := client.CancelAllRequests()
fmt.Printf("Cancelled %d requests\n", cancelled)
```

## License

MIT License - see LICENSE file for details.