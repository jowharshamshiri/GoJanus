package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/user/GoUnixSockAPI/pkg/core"
	"github.com/user/GoUnixSockAPI/pkg/models"
	"github.com/user/GoUnixSockAPI/pkg/specification"
)

func main() {
	var (
		socketPath = flag.String("socket", "/tmp/go-unixsock.sock", "Unix socket path")
		listen     = flag.Bool("listen", false, "Listen for datagrams on socket")
		sendTo     = flag.String("send-to", "", "Send datagram to socket path")
		command    = flag.String("command", "ping", "Command to send")
		message    = flag.String("message", "hello", "Message to send")
		specPath   = flag.String("spec", "", "API specification file (required for validation)")
		channelID  = flag.String("channel", "test", "Channel ID for command routing")
	)
	flag.Parse()

	// Load API specification if provided
	var apiSpec *specification.APISpecification
	if *specPath != "" {
		specData, err := os.ReadFile(*specPath)
		if err != nil {
			log.Fatalf("Failed to read API specification: %v", err)
		}
		
		parser := specification.NewAPISpecificationParser()
		apiSpec, err = parser.ParseJSON(specData)
		if err != nil {
			log.Fatalf("Failed to parse API specification: %v", err)
		}
		
		fmt.Printf("Loaded API specification: %s v%s\n", apiSpec.Name, apiSpec.Version)
	}

	if *listen {
		listenForDatagrams(*socketPath, apiSpec, *channelID)
	} else if *sendTo != "" {
		sendDatagram(*sendTo, *command, *message, apiSpec, *channelID)
	} else {
		fmt.Println("Usage: either --listen or --send-to required")
		flag.Usage()
		os.Exit(1)
	}
}

func listenForDatagrams(socketPath string, apiSpec *specification.APISpecification, channelID string) {
	fmt.Printf("Listening for SOCK_DGRAM on: %s\n", socketPath)
	if apiSpec != nil {
		fmt.Printf("API validation enabled for channel: %s\n", channelID)
	}
	
	os.Remove(socketPath)
	
	addr, err := net.ResolveUnixAddr("unixgram", socketPath)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
	}
	defer conn.Close()
	defer os.Remove(socketPath)

	fmt.Println("Ready to receive datagrams")

	for {
		buffer := make([]byte, 64*1024)
		n, _, err := conn.ReadFromUnix(buffer)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}

		var cmd models.SocketCommand
		if err := json.Unmarshal(buffer[:n], &cmd); err != nil {
			log.Printf("Failed to parse datagram: %v", err)
			continue
		}

		fmt.Printf("Received datagram: %s (ID: %s)\n", cmd.Command, cmd.ID)

		// Send response via reply_to if specified
		if cmd.ReplyTo != "" {
			sendResponse(cmd.ID, cmd.ChannelID, cmd.Command, cmd.Args, cmd.ReplyTo, apiSpec)
		}
	}
}

func sendDatagram(targetSocket, command, message string, apiSpec *specification.APISpecification, channelID string) {
	fmt.Printf("Sending SOCK_DGRAM to: %s\n", targetSocket)

	client, err := core.NewUnixDatagramClient(targetSocket)
	if err != nil {
		log.Fatalf("Failed to create datagram client: %v", err)
	}

	// Create response socket path
	responseSocket := fmt.Sprintf("/tmp/go-response-%d.sock", os.Getpid())
	
	args := map[string]interface{}{}
	
	// Add arguments based on command type
	if command == "echo" || command == "get_info" {
		args["message"] = message
	}

	// Validate command against API specification if provided
	if apiSpec != nil {
		if !apiSpec.HasCommand(channelID, command) {
			log.Fatalf("Command '%s' not found in channel '%s'", command, channelID)
		}
		
		commandSpec, err := apiSpec.GetCommand(channelID, command)
		if err != nil {
			log.Fatalf("Command validation failed: %v", err)
		}
		
		if err := apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
			log.Fatalf("Argument validation failed: %v", err)
		}
		
		fmt.Printf("Command validation passed for %s in channel %s\n", command, channelID)
	}

	cmd := models.SocketCommand{
		ID:        generateID(),
		ChannelID: channelID,
		Command:   command,
		ReplyTo:   responseSocket,
		Args:      args,
		Timeout:   func() *float64 { f := 5.0; return &f }(),
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}

	cmdData, err := json.Marshal(cmd)
	if err != nil {
		log.Fatalf("Failed to marshal command: %v", err)
	}

	// Send datagram and wait for response
	ctx := context.Background()
	responseData, err := client.SendDatagram(ctx, cmdData, responseSocket)
	if err != nil {
		log.Fatalf("Failed to send datagram: %v", err)
	}

	var response models.SocketResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		log.Printf("Failed to parse response: %v", err)
		return
	}

	fmt.Printf("Response: Success=%v, Result=%+v\n", response.Success, response.Result)
}

func sendResponse(cmdID, channelID, command string, args map[string]interface{}, replyTo string, apiSpec *specification.APISpecification) {
	var result map[string]interface{}
	var success = true
	var errorMsg *models.SocketError

	// Validate command against API specification if provided
	if apiSpec != nil {
		if !apiSpec.HasCommand(channelID, command) {
			success = false
			errorMsg = &models.SocketError{
				Code:    "UNKNOWN_COMMAND",
				Message: fmt.Sprintf("Command '%s' not found in channel '%s'", command, channelID),
			}
		} else {
			// Validate command arguments
			commandSpec, err := apiSpec.GetCommand(channelID, command)
			if err != nil {
				success = false
				errorMsg = &models.SocketError{
					Code:    "VALIDATION_ERROR",
					Message: fmt.Sprintf("Command validation failed: %v", err),
				}
			} else if err := apiSpec.ValidateCommandArgs(commandSpec, args); err != nil {
				success = false
				errorMsg = &models.SocketError{
					Code:    "INVALID_ARGUMENTS",
					Message: fmt.Sprintf("Argument validation failed: %v", err),
				}
			}
		}
	}

	// Only process command if validation passed
	if success {
		switch command {
	case "ping":
		result = map[string]interface{}{
			"pong": true,
			"echo": args,
		}
	case "echo":
		result = map[string]interface{}{
			"message": args["message"],
		}
	case "get_info":
		result = map[string]interface{}{
			"implementation": "Go",
			"version":        "1.0.0",
			"protocol":       "SOCK_DGRAM",
		}
	default:
		success = false
		errorMsg = &models.SocketError{
			Code:    "UNKNOWN_COMMAND",
			Message: fmt.Sprintf("Unknown command: %s", command),
		}
	}
	}

	response := models.SocketResponse{
		CommandID: cmdID,
		ChannelID: channelID,
		Success:   success,
		Result:    result,
		Error:     errorMsg,
		Timestamp: float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Send response datagram to reply_to socket
	replyAddr, err := net.ResolveUnixAddr("unixgram", replyTo)
	if err != nil {
		log.Printf("Failed to resolve reply address: %v", err)
		return
	}

	conn, err := net.DialUnix("unixgram", nil, replyAddr)
	if err != nil {
		log.Printf("Failed to dial reply socket: %v", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(responseData)
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}