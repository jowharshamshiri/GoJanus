# GoJanus

A production-ready Unix domain socket communication library for Go with **async response tracking** and cross-language compatibility.

## Features

- **Async Response Tracking**: Proper async patterns with persistent connections and response correlation
- **Cross-Language Compatibility**: Seamless communication with Rust and Swift implementations  
- **Persistent Connection Management**: Background message listeners with proper async task management
- **Security Framework**: Comprehensive path validation, resource limits, and attack prevention
- **Manifest Engine**: JSON/YAML-driven command validation and type safety
- **Performance Optimized**: Async communication patterns optimized for Unix socket inherent async nature
- **Production Ready**: Enterprise-grade error handling and resource management
- **Cross-Platform**: Works on all Unix-like systems (Linux, macOS, BSD)

## Installation

```bash
go mod init your-project
go get github.com/jowharshamshiri/GoJanus
```

## Quick Start

### Async Client Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
    "github.com/jowharshamshiri/GoJanus/pkg/specification"
)

func main() {
    // Load Manifest
    parser := specification.NewManifestParser()
    spec, err := parser.ParseFromFile("manifest.json")
    if err != nil {
        panic(err)
    }
    
    // Create async client with proper configuration
    client, err := protocol.NewJanusClient(
        "/tmp/my_socket.sock",
        "my_channel", 
        spec,
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Send async command with response tracking
    args := map[string]interface{}{
        "message": "Hello World",
    }
    
    ctx := context.Background()
    response, err := client.SendCommand(
        ctx,
        "echo",
        args,
        5*time.Second,
        nil, // timeout handler
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Response: %+v\n", response)
}
```

### Server with Command Handlers

```go
package main

import (
    "context"
    "fmt"
    "github.com/jowharshamshiri/GoJanus/pkg/protocol"
    "github.com/jowharshamshiri/GoJanus/pkg/models"
)

func main() {
    // Create server client for handling commands
    client, err := protocol.NewJanusClient(
        "/tmp/my_socket.sock",
        "my_channel",
        spec,
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Register async command handlers
    err = client.RegisterCommandHandler("echo", func(cmd models.SocketCommand, args map[string]interface{}) (*models.SocketResponse, error) {
        message, ok := args["message"].(string)
        if !ok {
            message = "No message provided"
        }
        
        return &models.SocketResponse{
            CommandID: cmd.ID,
            ChannelID: cmd.ChannelID,
            Success:   true,
            Result:    map[string]interface{}{"echo": message},
        }, nil
    })
    if err != nil {
        panic(err)
    }
    
    // Start listening for commands
    fmt.Println("Server listening...")
    err = client.StartListening()
    if err != nil {
        panic(err)
    }
}
```

## Key Async Architecture Features

### Response Tracking
- **ResponseTracker**: Correlates async responses with pending commands using UUID tracking
- **Persistent Connections**: `ReceiveMessage()` method for async listening instead of blocking
- **Background Listeners**: `messageListenerLoop()` runs in separate goroutine for response handling

### Cross-Platform Compatibility
- **Protocol Compatibility**: Works seamlessly with Rust and Swift implementations
- **Message Format**: Standardized JSON message format across all languages
- **Manifest**: Shared JSON/YAML Manifest format

### Security & Performance
- **Path Validation**: Comprehensive socket path security validation
- **Resource Limits**: Configurable limits on connections, message sizes, pending commands
- **Timeout Management**: Bilateral timeout protection with proper cleanup
- **Memory Management**: Automatic resource cleanup and connection pooling

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
config := protocol.JanusClientConfig{
    MaxConcurrentConnections: 100,
    MaxMessageSize:          10 * 1024 * 1024, // 10MB
    ConnectionTimeout:       30 * time.Second,
    MaxPendingCommands:      1000,
    MaxCommandHandlers:      500,
    EnableResourceMonitoring: true,
    MaxChannelNameLength:    256,
    MaxCommandNameLength:    256,
    MaxArgsDataSize:         5 * 1024 * 1024, // 5MB
}
```

## License

MIT License - see LICENSE file for details.