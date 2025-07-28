package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	api "github.com/user/GoUnixSocketAPI"
)

func main() {
	// Parse command line arguments
	socketPath := flag.String("socket-path", "/tmp/go_test_server.sock", "Unix socket path")
	flag.Parse()

	fmt.Printf("Connecting Go client to: %s\n", *socketPath)

	// Create API specification (matching server)
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

	fmt.Println("Testing ping command...")

	// Test ping command
	response, err := client.SendCommand(
		context.Background(),
		"ping",
		nil,
		5*time.Second,
		nil,
	)
	if err != nil {
		fmt.Printf("✗ Ping test error: %v\n", err)
		return
	}

	fmt.Printf("Ping response: %+v\n", response)
	if response.Success {
		fmt.Println("✓ Ping test passed")
	} else {
		fmt.Printf("✗ Ping test failed: %s\n", response.Error.Message)
		return
	}

	fmt.Println("Testing echo command...")

	// Test echo command
	args := map[string]interface{}{
		"message": "Hello from Go client!",
	}

	response, err = client.SendCommand(
		context.Background(),
		"echo",
		args,
		5*time.Second,
		nil,
	)
	if err != nil {
		fmt.Printf("✗ Echo test error: %v\n", err)
		return
	}

	fmt.Printf("Echo response: %+v\n", response)
	if response.Success {
		fmt.Println("✓ Echo test passed")
	} else {
		fmt.Printf("✗ Echo test failed: %s\n", response.Error.Message)
		return
	}

	fmt.Println("All Go client tests completed successfully! 🎉")
}