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
	specPath := flag.String("spec", "test-api-spec.json", "API specification file")
	flag.Parse()

	fmt.Printf("Starting Go Unix Socket API Server on: %s\n", *socketPath)

	// Remove existing socket file
	_ = os.Remove(*socketPath)
	
	specData, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("Failed to read API specification: %v", err)
	}
	
	parser := api.NewAPISpecificationParser()
	spec, err := parser.ParseJSON(specData)
	if err != nil {
		log.Fatalf("Failed to parse API specification: %v", err)
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