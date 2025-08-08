# GoJanus

A production-ready Unix domain socket communication library for Go with **SOCK_DGRAM connectionless communication** and automatic ID management.

## Features

- **Connectionless SOCK_DGRAM**: Unix domain datagram sockets with reply-to mechanism
- **Automatic ID Management**: RequestHandle system hides UUID complexity from users
- **Cross-Language Compatibility**: Perfect compatibility with Rust, Swift, and TypeScript implementations  
- **Dynamic Manifest**: Server-provided Manifests with auto-fetch validation
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

### API Manifest (Manifest)

Before creating servers or clients, you need a Manifest file defining your API:

**my-api-manifest.json:**
```json
{
  "name": "My Application API",
  "version": "1.0.0",
  "description": "Example API for demonstration",
  "channels": {
    "default": {
      "requests": {
        "get_user": {
          "description": "Retrieve user information",
          "arguments": {
            "user_id": {
              "type": "string",
              "required": true,
              "description": "User identifier"
            }
          },
          "response": {
            "type": "object",
            "properties": {
              "id": {"type": "string"},
              "name": {"type": "string"},
              "email": {"type": "string"}
            }
          }
        },
        "update_profile": {
          "description": "Update user profile",
          "arguments": {
            "user_id": {"type": "string", "required": true},
            "name": {"type": "string", "required": false},
            "email": {"type": "string", "required": false}
          },
          "response": {
            "type": "object",
            "properties": {
              "success": {"type": "boolean"},
              "updated_fields": {"type": "array"}
            }
          }
        }
      }
    }
  }
}
```

**Note**: Built-in requests (`ping`, `echo`, `get_info`, `validate`, `slow_process`, `manifest`) are always available and cannot be overridden in Manifests.

### Simple Client Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
)

func main() {
    // Create client - manifest is fetched automatically from server
    client, err := protocol.NewJanusClient("/tmp/my-server.sock", "default")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Built-in requests (always available)
    response, err := client.SendRequest("ping", nil)
    if err != nil {
        panic(err)
    }
    
    if response.Success {
        fmt.Printf("Server ping: %v\n", response.Result)
    }
    
    // Custom request defined in Manifest (arguments validated automatically)
    userArgs := map[string]interface{}{
        "user_id": "user123",
    }
    
    response, err = client.SendRequest("get_user", userArgs)
    if err != nil {
        panic(err)
    }
    
    if response.Success {
        fmt.Printf("User data: %v\n", response.Result)
    } else {
        fmt.Printf("Error: %v\n", response.Error)
    }
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
    
    // Send request with RequestHandle for tracking
    handle, responseC, errC := client.SendRequestWithHandle(
        context.Background(),
        "process_data",
        args,
    )
    
    fmt.Printf("Request started: %s on channel %s\n", 
        handle.GetRequest(), handle.GetChannel())
    
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

### Server Usage

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/jowharshamshiri/GoJanus/pkg/server"
    "github.com/jowharshamshiri/GoJanus/pkg/models"
    "github.com/jowharshamshiri/GoJanus/pkg/manifest"
)

func main() {
    // Load API manifest from Manifest file
    manifest, err := manifest.ParseManifestFromFile("my-api-manifest.json")
    if err != nil {
        fmt.Printf("Failed to load manifest: %v\n", err)
        return
    }
    
    // Create server with configuration
    config := &server.ServerConfig{
        SocketPath:        "/tmp/my-server.sock",
        CleanupOnStart:    true,
        CleanupOnShutdown: true,
    }
    srv := server.NewJanusServer(config)
    
    // Set the server's manifest for validation and manifest request
    srv.SetManifest(manifest)
    
    // Register handlers for requests defined in the Manifest
    srv.RegisterHandler("get_user", server.NewObjectHandler(func(cmd *models.JanusRequest) (map[string]interface{}, error) {
        // Extract user_id argument (validated by Manifest)
        userID, exists := cmd.Args["user_id"]
        if !exists {
            return nil, &models.JSONRPCError{
                Code:    models.InvalidParams,
                Message: "Missing user_id argument",
            }
        }
        
        userIDStr, ok := userID.(string)
        if !ok {
            return nil, &models.JSONRPCError{
                Code:    models.InvalidParams,
                Message: "user_id must be a string",
            }
        }
        
        // Simulate user lookup
        return map[string]interface{}{
            "id":    userIDStr,
            "name":  "John Doe",
            "email": "john@example.com",
        }, nil
    }))
    
    srv.RegisterHandler("update_profile", server.NewObjectHandler(func(cmd *models.JanusRequest) (map[string]interface{}, error) {
        if cmd.Args == nil {
            return nil, &models.JSONRPCError{
                Code:    models.InvalidParams,
                Message: "No arguments provided",
            }
        }
        
        userID, exists := cmd.Args["user_id"]
        if !exists {
            return nil, &models.JSONRPCError{
                Code:    models.InvalidParams,
                Message: "Missing user_id argument",
            }
        }
        
        updatedFields := []string{}
        if _, exists := cmd.Args["name"]; exists {
            updatedFields = append(updatedFields, "name")
        }
        if _, exists := cmd.Args["email"]; exists {
            updatedFields = append(updatedFields, "email")
        }
        
        return map[string]interface{}{
            "success":        true,
            "updated_fields": updatedFields,
        }, nil
    }))
    
    // Handle graceful shutdown
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        fmt.Println("Shutting down server...")
        srv.Stop()
    }()
    
    // Start listening (blocks until stopped)
    if err := srv.StartListening("/tmp/my-server.sock"); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

### Client Usage

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
)

func main() {
    // Create client - manifest is fetched automatically from server
    client, err := protocol.NewJanusClient("/tmp/my-server.sock", "default")
    if err != nil {
        fmt.Printf("Failed to create client: %v\n", err)
        return
    }
    
    // Set timeout for requests
    client.SetTimeout(30 * time.Second)
    
    // Built-in requests (always available)
    response, err := client.SendRequest("ping", nil)
    if err != nil {
        fmt.Printf("Ping failed: %v\n", err)
        return
    }
    
    if response.Success {
        fmt.Printf("Server ping: %v\n", response.Result)
    }
    
    // Custom request defined in Manifest (arguments validated automatically)
    userArgs := map[string]interface{}{
        "user_id": "user123",
    }
    
    response, err = client.SendRequest("get_user", userArgs)
    if err != nil {
        fmt.Printf("Get user failed: %v\n", err)
        return
    }
    
    if response.Success {
        fmt.Printf("User data: %v\n", response.Result)
    } else {
        fmt.Printf("Error: %v\n", response.Error)
    }
    
    // Get server API manifest
    manifestResponse, err := client.SendRequest("manifest", nil)
    if err == nil && manifestResponse.Success {
        fmt.Printf("Server API manifest: %v\n", manifestResponse.Result)
    }
    
    // Test connectivity
    if client.Ping() {
        fmt.Println("Server is responsive")
    }
}
```

### Fire-and-Forget Requests

```go
// Send request without waiting for response
logArgs := map[string]interface{}{
    "level":   "info",
    "message": "User profile updated",
}

if err := client.SendRequestNoResponse("log_event", logArgs); err != nil {
    fmt.Printf("Fire-and-forget failed: %v\n", err)
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
        handle.GetRequest(), 
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