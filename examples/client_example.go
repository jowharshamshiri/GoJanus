package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/user/GoUnixSocketAPI"
)

// Example demonstrating how to use GoUnixSocketAPI as a client
// This shows all major features including API specification parsing, 
// client creation, command sending, and error handling
func main() {
	fmt.Println("GoUnixSocketAPI Client Example")
	fmt.Println("==============================")
	
	// Parse API specification from file
	fmt.Println("1. Parsing API specification...")
	spec, err := gounixsocketapi.ParseAPISpecFromFile("examples/example-api-spec.json")
	if err != nil {
		log.Fatalf("Failed to parse API specification: %v", err)
	}
	fmt.Printf("   ✓ Loaded API spec: %s v%s\n", spec.Name, spec.Version)
	fmt.Printf("   ✓ Channels: %d, Models: %d\n", len(spec.Channels), len(spec.Models))
	
	// Create client with custom configuration
	fmt.Println("\n2. Creating Unix socket API client...")
	config := gounixsocketapi.UnixSockAPIClientConfig{
		MaxConcurrentConnections: 50,                  // Reduced for example
		MaxMessageSize:          5 * 1024 * 1024,     // 5MB
		ConnectionTimeout:       15 * time.Second,     // 15s timeout
		MaxPendingCommands:      100,                  // 100 pending commands
		MaxCommandHandlers:      50,                   // 50 handlers max
		EnableResourceMonitoring: true,               // Enable monitoring
		MaxChannelNameLength:    256,                 // Standard limit
		MaxCommandNameLength:    256,                 // Standard limit
		MaxArgsDataSize:         1 * 1024 * 1024,     // 1MB args max
	}
	
	client, err := gounixsocketapi.NewUnixSockAPIClientWithConfig(
		"/tmp/example-service.sock",
		"library-management",
		spec,
		config,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	fmt.Printf("   ✓ Client created for channel: %s\n", client.ChannelIdentifier())
	fmt.Printf("   ✓ Socket path: %s\n", client.SocketPathString())
	
	// Example 1: Send a command with response
	fmt.Println("\n3. Sending 'get-book' command...")
	ctx := context.Background()
	
	args := map[string]interface{}{
		"id": "book-12345",
	}
	
	response, err := client.SendCommand(
		ctx,
		"get-book",
		args,
		30*time.Second,
		func(commandID string) {
			fmt.Printf("   ⚠ Command %s timed out\n", commandID)
		},
	)
	
	if err != nil {
		fmt.Printf("   ✗ Command failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Command successful: %s\n", response.CommandID)
		if response.Success {
			fmt.Printf("   ✓ Result: %+v\n", response.Result)
		} else {
			fmt.Printf("   ✗ Error: %+v\n", response.Error)
		}
	}
	
	// Example 2: Fire-and-forget command
	fmt.Println("\n4. Publishing 'add-book' command (fire-and-forget)...")
	
	addArgs := map[string]interface{}{
		"title":  "The Go Programming Language",
		"author": "Alan Donovan and Brian Kernighan",
		"isbn":   "978-0134190440",
		"pages":  380,
	}
	
	commandID, err := client.PublishCommand(ctx, "add-book", addArgs)
	if err != nil {
		fmt.Printf("   ✗ Publish failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Command published: %s\n", commandID)
	}
	
	// Example 3: Search with complex arguments
	fmt.Println("\n5. Sending 'search-books' command with complex arguments...")
	
	searchArgs := map[string]interface{}{
		"query": "golang",
		"field": "title",
		"limit": 5,
	}
	
	searchResponse, err := client.SendCommand(
		ctx,
		"search-books",
		searchArgs,
		30*time.Second,
		nil, // No timeout handler
	)
	
	if err != nil {
		fmt.Printf("   ✗ Search failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Search completed: %s\n", searchResponse.CommandID)
		if searchResponse.Success {
			fmt.Printf("   ✓ Found books: %+v\n", searchResponse.Result)
		}
	}
	
	// Example 4: Working with different channels
	fmt.Println("\n6. Creating client for task-management channel...")
	
	taskClient, err := gounixsocketapi.NewUnixSockAPIClient(
		"/tmp/example-service.sock",
		"task-management",
		spec,
	)
	if err != nil {
		fmt.Printf("   ✗ Failed to create task client: %v\n", err)
	} else {
		defer taskClient.Close()
		fmt.Printf("   ✓ Task client created for channel: %s\n", taskClient.ChannelIdentifier())
		
		// Create a task
		taskArgs := map[string]interface{}{
			"title":       "Implement GoUnixSocketAPI tests",
			"description": "Create comprehensive test suite for the library",
			"priority":    "high",
			"due_date":    "2025-08-01T12:00:00Z",
		}
		
		taskResponse, err := taskClient.SendCommand(
			ctx,
			"create-task",
			taskArgs,
			30*time.Second,
			nil,
		)
		
		if err != nil {
			fmt.Printf("   ✗ Task creation failed: %v\n", err)
		} else {
			fmt.Printf("   ✓ Task created: %s\n", taskResponse.CommandID)
			if taskResponse.Success {
				fmt.Printf("   ✓ Task details: %+v\n", taskResponse.Result)
			}
		}
	}
	
	// Example 5: Error handling and validation
	fmt.Println("\n7. Testing error handling with invalid command...")
	
	_, err = client.SendCommand(
		ctx,
		"invalid-command",
		map[string]interface{}{"test": "value"},
		10*time.Second,
		nil,
	)
	
	if err != nil {
		fmt.Printf("   ✓ Expected error caught: %v\n", err)
	} else {
		fmt.Printf("   ✗ Expected error but command succeeded\n")
	}
	
	// Example 6: Testing argument validation
	fmt.Println("\n8. Testing argument validation with invalid data...")
	
	invalidArgs := map[string]interface{}{
		"id": "", // Empty ID should fail validation
	}
	
	_, err = client.SendCommand(
		ctx,
		"get-book",
		invalidArgs,
		10*time.Second,
		nil,
	)
	
	if err != nil {
		fmt.Printf("   ✓ Validation error caught: %v\n", err)
	} else {
		fmt.Printf("   ✗ Expected validation error but command succeeded\n")
	}
	
	fmt.Println("\n9. Client example completed successfully!")
	fmt.Println("\nKey features demonstrated:")
	fmt.Println("  ✓ API specification parsing from JSON")
	fmt.Println("  ✓ Client creation with custom configuration")
	fmt.Println("  ✓ Command sending with response handling")
	fmt.Println("  ✓ Fire-and-forget command publishing")
	fmt.Println("  ✓ Multi-channel support")
	fmt.Println("  ✓ Timeout management")
	fmt.Println("  ✓ Error handling and validation")
	fmt.Println("  ✓ Complex argument validation")
	fmt.Println("  ✓ Resource management and cleanup")
}