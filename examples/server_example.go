package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/user/GoUnixSocketAPI"
)

// In-memory data stores for the example
var (
	books = make(map[string]map[string]interface{})
	tasks = make(map[string]map[string]interface{})
	booksMutex sync.RWMutex
	tasksMutex sync.RWMutex
	bookCounter = 1
	taskCounter = 1
)

// Example demonstrating how to use GoUnixSocketAPI as a server
// This shows handler registration, persistent listening, and command processing
func main() {
	fmt.Println("GoUnixSocketAPI Server Example")
	fmt.Println("==============================")
	
	// Parse API specification
	fmt.Println("1. Loading API specification...")
	spec, err := gounixsocketapi.ParseAPISpecFromFile("examples/example-api-spec.json")
	if err != nil {
		log.Fatalf("Failed to parse API specification: %v", err)
	}
	fmt.Printf("   ✓ Loaded API spec: %s v%s\n", spec.Name, spec.Version)
	
	// Initialize example data
	initializeExampleData()
	
	// Create server clients for different channels
	clients := make([]*gounixsocketapi.UnixSockAPIClient, 0)
	
	// Library management server
	fmt.Println("\n2. Setting up library management server...")
	libraryClient, err := createLibraryServer(spec)
	if err != nil {
		log.Fatalf("Failed to create library server: %v", err)
	}
	clients = append(clients, libraryClient)
	fmt.Println("   ✓ Library management handlers registered")
	
	// Task management server
	fmt.Println("\n3. Setting up task management server...")
	taskClient, err := createTaskServer(spec)
	if err != nil {
		log.Fatalf("Failed to create task server: %v", err)
	}
	clients = append(clients, taskClient)
	fmt.Println("   ✓ Task management handlers registered")
	
	// LLM operations server (mock implementation)
	fmt.Println("\n4. Setting up LLM operations server...")
	llmClient, err := createLLMServer(spec)
	if err != nil {
		log.Fatalf("Failed to create LLM server: %v", err)
	}
	clients = append(clients, llmClient)
	fmt.Println("   ✓ LLM operations handlers registered")
	
	// Start all servers
	fmt.Println("\n5. Starting all servers...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	for i, client := range clients {
		go func(idx int, c *gounixsocketapi.UnixSockAPIClient) {
			if err := c.StartListening(ctx); err != nil {
				log.Printf("Server %d failed to start listening: %v", idx, err)
			}
		}(i, client)
		fmt.Printf("   ✓ Server %d listening on channel: %s\n", i+1, client.ChannelIdentifier())
	}
	
	fmt.Println("\n🚀 All servers running! Socket: /tmp/example-service.sock")
	fmt.Println("\nAvailable commands:")
	fmt.Println("Library Management (library-management):")
	fmt.Println("  - get-book {\"id\": \"book-1\"}")
	fmt.Println("  - add-book {\"title\": \"...\", \"author\": \"...\"}")
	fmt.Println("  - search-books {\"query\": \"...\", \"field\": \"title\"}")
	fmt.Println("\nTask Management (task-management):")
	fmt.Println("  - create-task {\"title\": \"...\", \"priority\": \"high\"}")
	fmt.Println("  - get-task {\"id\": \"task-1\"}")
	fmt.Println("  - update-task-status {\"id\": \"task-1\", \"status\": \"completed\"}")
	fmt.Println("\nLLM Operations (llm-operations):")
	fmt.Println("  - generate-text {\"prompt\": \"...\", \"max_tokens\": 100}")
	fmt.Println("  - analyze-sentiment {\"text\": \"...\"}")
	
	// Wait for interrupt signal
	fmt.Println("\nPress Ctrl+C to stop the server...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\n6. Shutting down servers...")
	cancel()
	
	// Close all clients
	for i, client := range clients {
		if err := client.Close(); err != nil {
			log.Printf("Error closing client %d: %v", i, err)
		}
	}
	
	fmt.Println("   ✓ All servers shut down cleanly")
	fmt.Println("   ✓ Server example completed")
}

// createLibraryServer sets up the library management server with handlers
func createLibraryServer(spec *gounixsocketapi.APISpecification) (*gounixsocketapi.UnixSockAPIClient, error) {
	client, err := gounixsocketapi.NewUnixSockAPIClient(
		"/tmp/example-service.sock",
		"library-management",
		spec,
	)
	if err != nil {
		return nil, err
	}
	
	// Register get-book handler
	err = client.RegisterCommandHandler("get-book", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		bookID, ok := command.Args["id"].(string)
		if !ok {
			return gounixsocketapi.NewErrorResponse(command.ID, command.ChannelID, &gounixsocketapi.SocketError{
				Code:    "INVALID_ID",
				Message: "Book ID must be a string",
			}), nil
		}
		
		booksMutex.RLock()
		book, exists := books[bookID]
		booksMutex.RUnlock()
		
		if !exists {
			return gounixsocketapi.NewErrorResponse(command.ID, command.ChannelID, &gounixsocketapi.SocketError{
				Code:    "BOOK_NOT_FOUND",
				Message: fmt.Sprintf("Book with ID '%s' not found", bookID),
			}), nil
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, book), nil
	})
	if err != nil {
		return nil, err
	}
	
	// Register add-book handler
	err = client.RegisterCommandHandler("add-book", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		title, _ := command.Args["title"].(string)
		author, _ := command.Args["author"].(string)
		isbn, _ := command.Args["isbn"].(string)
		pages, _ := command.Args["pages"].(float64) // JSON numbers are float64
		
		booksMutex.Lock()
		bookID := fmt.Sprintf("book-%d", bookCounter)
		bookCounter++
		
		book := map[string]interface{}{
			"id":        bookID,
			"title":     title,
			"author":    author,
			"isbn":      isbn,
			"pages":     int(pages),
			"available": true,
			"created":   time.Now().UTC().Format(time.RFC3339),
		}
		books[bookID] = book
		booksMutex.Unlock()
		
		result := map[string]interface{}{
			"id":      bookID,
			"created": book["created"],
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, result), nil
	})
	if err != nil {
		return nil, err
	}
	
	// Register search-books handler
	err = client.RegisterCommandHandler("search-books", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		query, _ := command.Args["query"].(string)
		field, _ := command.Args["field"].(string)
		if field == "" {
			field = "all"
		}
		limit, _ := command.Args["limit"].(float64)
		if limit == 0 {
			limit = 10
		}
		
		booksMutex.RLock()
		var results []map[string]interface{}
		for _, book := range books {
			matches := false
			switch field {
			case "title":
				if title, ok := book["title"].(string); ok && contains(title, query) {
					matches = true
				}
			case "author":
				if author, ok := book["author"].(string); ok && contains(author, query) {
					matches = true
				}
			case "isbn":
				if isbn, ok := book["isbn"].(string); ok && contains(isbn, query) {
					matches = true
				}
			case "all":
				if title, ok := book["title"].(string); ok && contains(title, query) {
					matches = true
				} else if author, ok := book["author"].(string); ok && contains(author, query) {
					matches = true
				} else if isbn, ok := book["isbn"].(string); ok && contains(isbn, query) {
					matches = true
				}
			}
			
			if matches {
				results = append(results, book)
				if len(results) >= int(limit) {
					break
				}
			}
		}
		booksMutex.RUnlock()
		
		response := map[string]interface{}{
			"books": results,
			"total": len(results),
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, response), nil
	})
	
	return client, err
}

// createTaskServer sets up the task management server with handlers
func createTaskServer(spec *gounixsocketapi.APISpecification) (*gounixsocketapi.UnixSockAPIClient, error) {
	client, err := gounixsocketapi.NewUnixSockAPIClient(
		"/tmp/example-service.sock",
		"task-management",
		spec,
	)
	if err != nil {
		return nil, err
	}
	
	// Register create-task handler
	err = client.RegisterCommandHandler("create-task", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		title, _ := command.Args["title"].(string)
		description, _ := command.Args["description"].(string)
		priority, _ := command.Args["priority"].(string)
		if priority == "" {
			priority = "medium"
		}
		dueDate, _ := command.Args["due_date"].(string)
		
		tasksMutex.Lock()
		taskID := fmt.Sprintf("task-%d", taskCounter)
		taskCounter++
		
		now := time.Now().UTC().Format(time.RFC3339)
		task := map[string]interface{}{
			"id":          taskID,
			"title":       title,
			"description": description,
			"status":      "pending",
			"priority":    priority,
			"due_date":    dueDate,
			"created":     now,
			"updated":     now,
		}
		tasks[taskID] = task
		tasksMutex.Unlock()
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, task), nil
	})
	if err != nil {
		return nil, err
	}
	
	// Register get-task handler
	err = client.RegisterCommandHandler("get-task", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		taskID, ok := command.Args["id"].(string)
		if !ok {
			return gounixsocketapi.NewErrorResponse(command.ID, command.ChannelID, &gounixsocketapi.SocketError{
				Code:    "INVALID_ID",
				Message: "Task ID must be a string",
			}), nil
		}
		
		tasksMutex.RLock()
		task, exists := tasks[taskID]
		tasksMutex.RUnlock()
		
		if !exists {
			return gounixsocketapi.NewErrorResponse(command.ID, command.ChannelID, &gounixsocketapi.SocketError{
				Code:    "TASK_NOT_FOUND",
				Message: fmt.Sprintf("Task with ID '%s' not found", taskID),
			}), nil
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, task), nil
	})
	if err != nil {
		return nil, err
	}
	
	// Register update-task-status handler
	err = client.RegisterCommandHandler("update-task-status", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		taskID, _ := command.Args["id"].(string)
		newStatus, _ := command.Args["status"].(string)
		
		tasksMutex.Lock()
		task, exists := tasks[taskID]
		if exists {
			task["status"] = newStatus
			task["updated"] = time.Now().UTC().Format(time.RFC3339)
			tasks[taskID] = task
		}
		tasksMutex.Unlock()
		
		if !exists {
			return gounixsocketapi.NewErrorResponse(command.ID, command.ChannelID, &gounixsocketapi.SocketError{
				Code:    "TASK_NOT_FOUND",
				Message: fmt.Sprintf("Task with ID '%s' not found", taskID),
			}), nil
		}
		
		result := map[string]interface{}{
			"id":      taskID,
			"status":  newStatus,
			"updated": task["updated"],
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, result), nil
	})
	
	return client, err
}

// createLLMServer sets up the LLM operations server with mock handlers
func createLLMServer(spec *gounixsocketapi.APISpecification) (*gounixsocketapi.UnixSockAPIClient, error) {
	client, err := gounixsocketapi.NewUnixSockAPIClient(
		"/tmp/example-service.sock",
		"llm-operations",
		spec,
	)
	if err != nil {
		return nil, err
	}
	
	// Register generate-text handler (mock implementation)
	err = client.RegisterCommandHandler("generate-text", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		prompt, _ := command.Args["prompt"].(string)
		maxTokens, _ := command.Args["max_tokens"].(float64)
		temperature, _ := command.Args["temperature"].(float64)
		model, _ := command.Args["model"].(string)
		
		if maxTokens == 0 {
			maxTokens = 100
		}
		if temperature == 0 {
			temperature = 0.7
		}
		if model == "" {
			model = "gpt-3.5"
		}
		
		// Mock text generation
		generatedText := fmt.Sprintf("This is a mock response to the prompt: '%s'. In a real implementation, this would be generated by the %s model with temperature %.1f.", prompt, model, temperature)
		tokensUsed := len(generatedText) / 4 // Rough approximation
		
		result := map[string]interface{}{
			"text":        generatedText,
			"tokens_used": tokensUsed,
			"model_used":  model,
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, result), nil
	})
	if err != nil {
		return nil, err
	}
	
	// Register analyze-sentiment handler (mock implementation)
	err = client.RegisterCommandHandler("analyze-sentiment", func(command *gounixsocketapi.SocketCommand) (*gounixsocketapi.SocketResponse, error) {
		text, _ := command.Args["text"].(string)
		
		// Mock sentiment analysis
		sentiment := "neutral"
		confidence := 0.75
		
		// Simple keyword-based mock analysis
		if contains(text, "good") || contains(text, "great") || contains(text, "excellent") {
			sentiment = "positive"
			confidence = 0.85
		} else if contains(text, "bad") || contains(text, "terrible") || contains(text, "awful") {
			sentiment = "negative"
			confidence = 0.80
		}
		
		result := map[string]interface{}{
			"sentiment":  sentiment,
			"confidence": confidence,
			"scores": map[string]interface{}{
				"positive": confidence,
				"negative": 1.0 - confidence,
				"neutral":  0.5,
			},
		}
		
		return gounixsocketapi.NewSuccessResponse(command.ID, command.ChannelID, result), nil
	})
	
	return client, err
}

// initializeExampleData creates some initial data for the example
func initializeExampleData() {
	fmt.Println("   Initializing example data...")
	
	// Add some example books
	booksMutex.Lock()
	books["book-1"] = map[string]interface{}{
		"id":        "book-1",
		"title":     "The Go Programming Language",
		"author":    "Alan Donovan and Brian Kernighan",
		"isbn":      "978-0134190440",
		"pages":     380,
		"available": true,
		"created":   "2025-01-01T12:00:00Z",
	}
	books["book-2"] = map[string]interface{}{
		"id":        "book-2",
		"title":     "Clean Code",
		"author":    "Robert C. Martin",
		"isbn":      "978-0132350884",
		"pages":     464,
		"available": true,
		"created":   "2025-01-01T12:00:00Z",
	}
	bookCounter = 3
	booksMutex.Unlock()
	
	// Add some example tasks
	tasksMutex.Lock()
	tasks["task-1"] = map[string]interface{}{
		"id":          "task-1",
		"title":       "Implement Unix Socket API",
		"description": "Create a cross-language compatible Unix socket API",
		"status":      "in_progress",
		"priority":    "high",
		"due_date":    "2025-08-01T12:00:00Z",
		"created":     "2025-07-20T12:00:00Z",
		"updated":     "2025-07-28T15:00:00Z",
	}
	taskCounter = 2
	tasksMutex.Unlock()
	
	fmt.Printf("   ✓ Added %d books and %d tasks\n", len(books), len(tasks))
}

// contains is a simple case-insensitive string contains function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}