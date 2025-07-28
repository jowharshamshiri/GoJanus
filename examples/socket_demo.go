package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    api "github.com/user/GoUnixSocketAPI"
)

func main() {
    socketPath := "/tmp/go_socket_test.sock"
    
    // Remove any existing socket
    os.Remove(socketPath)
    
    // Create simple API spec
    spec := &api.APISpecification{
        Version: "1.0.0",
        Name: "Test API",
        Channels: map[string]*api.ChannelSpec{
            "test": {
                Name: "test",
                Description: "Test channel",
                Commands: map[string]*api.CommandSpec{
                    "ping": {
                        Name: "ping",
                        Description: "Ping command",
                        Response: &api.ResponseSpec{Type: "object"},
                    },
                },
            },
        },
    }
    
    // Create client
    client, err := api.NewUnixSockAPIClient(socketPath, "test", spec)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()
    
    fmt.Println("Created client successfully")
    
    // Register handler - this should trigger server mode
    err = client.RegisterCommandHandler("ping", func(cmd *api.SocketCommand) (*api.SocketResponse, error) {
        return api.NewSuccessResponse(cmd.ID, cmd.ChannelID, map[string]interface{}{"pong": true}), nil
    })
    if err != nil {
        log.Fatalf("Failed to register handler: %v", err)
    }
    
    fmt.Println("Registered handler. Starting listening (should create socket)...")
    
    // Start listening in background
    go func() {
        if err := client.StartListening(context.Background()); err != nil {
            log.Printf("Error in StartListening: %v", err)
        }
    }()
    
    // Give it time to create socket
    time.Sleep(1 * time.Second)
    
    // Check if socket was created
    if _, err := os.Stat(socketPath); err == nil {
        fmt.Println("✓ Socket created successfully at:", socketPath)
    } else {
        fmt.Println("✗ Socket was not created:", err)
    }
    
    fmt.Println("Keeping server running for 5 seconds...")
    time.Sleep(5 * time.Second)
    
    fmt.Println("Test complete")
}