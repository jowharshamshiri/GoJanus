package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/user/GoUnixSocketAPI"
)

func main() {
	// Parse command line arguments
	socketPath := flag.String("socket-path", "/tmp/go_test_server.sock", "Unix socket path")
	flag.Parse()

	fmt.Printf("Starting Go Unix Socket API Server on: %s\n", *socketPath)

	// Remove existing socket file
	_ = os.Remove(*socketPath)

	// Create API specification
	spec := &api.APISpecification{
		Version:     "1.0.0",
		Name:        "Cross-Platform Test API",
		Description: "Test API for cross-platform communication",
		Channels: map[string]*api.ChannelSpec{
			"test": {
				Description: "Test channel",
				Commands: map[string]*api.CommandSpec{
					"ping": {
						Description: "Simple ping command",
						Response: &api.ResponseSpec{
							Type: "object",
							Properties: map[string]*api.ArgumentSpec{
								"pong": {
									Type:        "boolean",
									Description: "Ping response",
								},
								"timestamp": {
									Type:        "string",
									Description: "Response timestamp",
								},
							},
						},
					},
					"echo": {
						Description: "Echo back input",
						Arguments: map[string]*api.ArgumentSpec{
							"message": {
								Type:        "string",
								Required:    true,
								Description: "Message to echo",
							},
						},
						Response: &api.ResponseSpec{
							Type: "object",
							Properties: map[string]*api.ArgumentSpec{
								"echo": {
									Type:        "string",
									Description: "Echoed message",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create client
	client, err := api.NewUnixSockAPIClient(*socketPath, "test", spec)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Register ping handler
	err = client.RegisterCommandHandler("ping", func(cmd *api.SocketCommand) (*api.SocketResponse, error) {
		result := map[string]interface{}{
			"pong":      true,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		return api.NewSuccessResponse(cmd.ID, cmd.ChannelID, result), nil
	})
	if err != nil {
		log.Fatalf("Failed to register ping handler: %v", err)
	}

	// Register echo handler
	err = client.RegisterCommandHandler("echo", func(cmd *api.SocketCommand) (*api.SocketResponse, error) {
		message := "No message provided"
		if cmd.Args != nil {
			if msg, ok := cmd.Args["message"].(string); ok {
				message = msg
			}
		}
		
		result := map[string]interface{}{
			"echo": message,
		}
		return api.NewSuccessResponse(cmd.ID, cmd.ChannelID, result), nil
	})
	if err != nil {
		log.Fatalf("Failed to register echo handler: %v", err)
	}

	// Start listening
	err = client.StartListening(context.Background())
	if err != nil {
		log.Fatalf("Failed to start listening: %v", err)
	}

	fmt.Println("Go server listening. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down Go server...")
}