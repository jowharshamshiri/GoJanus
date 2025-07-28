package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	api "github.com/user/GoUnixSocketAPI"
)

func main() {
	// Parse command line arguments
	socketPath := flag.String("socket-path", "/tmp/go_test_server.sock", "Unix socket path")
	specPath := flag.String("spec", "test-api-spec.json", "API specification file")
	flag.Parse()

	fmt.Printf("Connecting Go client to: %s\n", *socketPath)

	// Load API specification from file
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