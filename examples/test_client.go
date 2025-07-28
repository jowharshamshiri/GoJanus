package main

import (
    "context"
    "fmt"
    "log"
    "time"
    api "github.com/user/GoUnixSocketAPI"
)

func main() {
    socketPath := "/tmp/go_socket_test.sock"
    
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
    
    // Create client (no handlers, so it should connect to existing socket)
    client, err := api.NewUnixSockAPIClient(socketPath, "test", spec)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()
    
    fmt.Println("Client created successfully")
    
    // Execute ping command
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    cmdID, err := client.ExecuteCommand(ctx, "ping", nil)
    if err != nil {
        log.Fatalf("Failed to execute command: %v", err)
    }
    
    fmt.Printf("Command sent with ID: %s\n", cmdID)
    fmt.Println("Test complete")
}