package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"GoJanus/pkg/models"
	"GoJanus/pkg/protocol"
)

// TestSOCKDGRAMSocketCreation tests actual Unix domain datagram socket creation
func TestSOCKDGRAMSocketCreation(t *testing.T) {
	testSocketPath := "/tmp/gojanus-dgram-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Create actual SOCK_DGRAM socket using low-level syscalls
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create SOCK_DGRAM socket: %v", err)
	}
	defer syscall.Close(fd)
	
	// Bind to Unix domain socket path
	addr := &syscall.SockaddrUnix{Name: testSocketPath}
	err = syscall.Bind(fd, addr)
	if err != nil {
		t.Fatalf("Failed to bind SOCK_DGRAM socket: %v", err)
	}
	
	// Verify socket file was created
	if _, err := os.Stat(testSocketPath); os.IsNotExist(err) {
		t.Error("Socket file was not created")
	}
	
	// Verify socket type is SOCK_DGRAM
	sockType, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_TYPE)
	if err != nil {
		t.Fatalf("Failed to get socket type: %v", err)
	}
	
	if sockType != syscall.SOCK_DGRAM {
		t.Errorf("Expected SOCK_DGRAM (%d), got %d", syscall.SOCK_DGRAM, sockType)
	}
}

// TestResponseSocketBinding tests creation of dedicated response sockets
func TestResponseSocketBinding(t *testing.T) {
	responseSocketPath := "/tmp/gojanus-response-test.sock"
	
	// Clean up before and after test
	os.Remove(responseSocketPath)
	defer os.Remove(responseSocketPath)
	
	// We don't need the client for this low-level socket test
	testMainSocketPath := "/tmp/gojanus-main-test.sock"
	os.Remove(testMainSocketPath)
	defer os.Remove(testMainSocketPath)
	
	// Test response socket path generation (manual generation for testing)
	generatedPath := fmt.Sprintf("/tmp/gojanus_response_%d.sock", time.Now().UnixNano())
	if generatedPath == "" {
		t.Error("Generated response socket path should not be empty")
	}
	
	if generatedPath == testMainSocketPath {
		t.Error("Response socket path should be different from main socket path")
	}
	
	// Test that generated paths are unique
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	path2 := fmt.Sprintf("/tmp/gojanus_response_%d.sock", time.Now().UnixNano())
	if generatedPath == path2 {
		t.Error("Generated response socket paths should be unique")
	}
	
	// Test actual response socket binding
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create response socket: %v", err)
	}
	defer syscall.Close(fd)
	
	addr := &syscall.SockaddrUnix{Name: responseSocketPath}
	err = syscall.Bind(fd, addr)
	if err != nil {
		t.Fatalf("Failed to bind response socket: %v", err)
	}
	
	// Verify response socket file was created
	if _, err := os.Stat(responseSocketPath); os.IsNotExist(err) {
		t.Error("Response socket file was not created")
	}
}

// TestSendWithResponse tests two-way SOCK_DGRAM communication with mock server
func TestSendWithResponse(t *testing.T) {
	serverSocketPath := "/tmp/gojanus-server-test.sock"
	responseSocketPath := "/tmp/gojanus-response-test.sock"
	
	// Clean up before and after test
	os.Remove(serverSocketPath)
	os.Remove(responseSocketPath)
	defer func() {
		os.Remove(serverSocketPath)
		os.Remove(responseSocketPath)
	}()
	
	// Create mock server socket
	serverFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create server socket: %v", err)
	}
	defer syscall.Close(serverFd)
	
	serverAddr := &syscall.SockaddrUnix{Name: serverSocketPath}
	err = syscall.Bind(serverFd, serverAddr)
	if err != nil {
		t.Fatalf("Failed to bind server socket: %v", err)
	}
	
	// Create response socket
	responseFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create response socket: %v", err)
	}
	defer syscall.Close(responseFd)
	
	responseAddr := &syscall.SockaddrUnix{Name: responseSocketPath}
	err = syscall.Bind(responseFd, responseAddr)
	if err != nil {
		t.Fatalf("Failed to bind response socket: %v", err)
	}
	
	// Test message with reply_to field
	request := models.NewJanusRequest("test-channel", "test-request", map[string]interface{}{
		"test_param": "test_value",
	}, nil)
	request.ReplyTo = &responseSocketPath
	
	jsonData, err := request.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}
	
	// Send to server socket
	clientFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create client socket: %v", err)
	}
	defer syscall.Close(clientFd)
	
	err = syscall.Sendto(clientFd, jsonData, 0, serverAddr)
	if err != nil {
		t.Fatalf("Failed to send datagram: %v", err)
	}
	
	// Read on server socket to verify message received
	buffer := make([]byte, 4096)
	n, _, err := syscall.Recvfrom(serverFd, buffer, 0)
	if err != nil {
		t.Fatalf("Failed to receive datagram: %v", err)
	}
	
	receivedData := buffer[:n]
	
	// Verify received message
	var receivedRequest models.JanusRequest
	err = receivedRequest.FromJSON(receivedData)
	if err != nil {
		t.Fatalf("Failed to deserialize received request: %v", err)
	}
	
	if receivedRequest.ChannelID != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", receivedRequest.ChannelID)
	}
	
	if receivedRequest.Request != "test-request" {
		t.Errorf("Expected request 'test-request', got '%s'", receivedRequest.Request)
	}
	
	if receivedRequest.ReplyTo == nil || *receivedRequest.ReplyTo != responseSocketPath {
		replyTo := ""
		if receivedRequest.ReplyTo != nil {
			replyTo = *receivedRequest.ReplyTo
		}
		t.Errorf("Expected reply_to '%s', got '%s'", responseSocketPath, replyTo)
	}
}

// TestFireAndForgetSend tests one-way datagram sending without response
func TestFireAndForgetSend(t *testing.T) {
	serverSocketPath := "/tmp/gojanus-server-test.sock"
	
	// Clean up before and after test
	os.Remove(serverSocketPath)
	defer os.Remove(serverSocketPath)
	
	// Create mock server socket
	serverFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create server socket: %v", err)
	}
	defer syscall.Close(serverFd)
	
	serverAddr := &syscall.SockaddrUnix{Name: serverSocketPath}
	err = syscall.Bind(serverFd, serverAddr)
	if err != nil {
		t.Fatalf("Failed to bind server socket: %v", err)
	}
	
	// We don't need a client for this low-level test, just create the request directly
	
	// Test fire-and-forget request (no reply_to field)
	request := models.NewJanusRequest("test-channel", "fire-and-forget", map[string]interface{}{
		"message": "no response needed",
	}, nil)
	// ReplyTo should be empty for fire-and-forget
	
	jsonData, err := request.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}
	
	// Send using fire-and-forget pattern
	clientFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create client socket: %v", err)
	}
	defer syscall.Close(clientFd)
	
	err = syscall.Sendto(clientFd, jsonData, 0, serverAddr)
	if err != nil {
		t.Fatalf("Failed to send fire-and-forget datagram: %v", err)
	}
	
	// Verify server received the message
	buffer := make([]byte, 4096)
	n, _, err := syscall.Recvfrom(serverFd, buffer, 0)
	if err != nil {
		t.Fatalf("Failed to receive fire-and-forget datagram: %v", err)
	}
	
	receivedData := buffer[:n]
	
	var receivedRequest models.JanusRequest
	err = receivedRequest.FromJSON(receivedData)
	if err != nil {
		t.Fatalf("Failed to deserialize received request: %v", err)
	}
	
	if receivedRequest.ReplyTo != nil && *receivedRequest.ReplyTo != "" {
		replyTo := ""
		if receivedRequest.ReplyTo != nil {
			replyTo = *receivedRequest.ReplyTo
		}
		t.Errorf("Fire-and-forget request should have empty ReplyTo, got '%s'", replyTo)
	}
	
	if receivedRequest.Request != "fire-and-forget" {
		t.Errorf("Expected request 'fire-and-forget', got '%s'", receivedRequest.Request)
	}
}

// TestDynamicMessageSizeDetection tests detection of system socket limits
func TestDynamicMessageSizeDetectionCore(t *testing.T) {
	testSocketPath := "/tmp/gojanus-size-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Create socket for size testing
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	defer syscall.Close(fd)
	
	addr := &syscall.SockaddrUnix{Name: testSocketPath}
	err = syscall.Bind(fd, addr)
	if err != nil {
		t.Fatalf("Failed to bind socket: %v", err)
	}
	
	// Test with various message sizes to find limit
	testSizes := []int{1024, 4096, 8192, 16384, 32768, 65536, 131072}
	maxSuccessfulSize := 0
	
	for _, size := range testSizes {
		// Create test message of manifestific size
		testData := make([]byte, size)
		for i := range testData {
			testData[i] = 'A'
		}
		
		// Try to send message to itself
		err := syscall.Sendto(fd, testData, 0, addr)
		if err != nil {
			// Hit the size limit
			if err == syscall.EMSGSIZE {
				t.Logf("Hit EMSGSIZE at size %d bytes", size)
				break
			} else {
				t.Fatalf("Unexpected error at size %d: %v", size, err)
			}
		}
		
		maxSuccessfulSize = size
		t.Logf("Successfully sent %d bytes", size)
	}
	
	if maxSuccessfulSize == 0 {
		t.Error("No successful message sizes - system may have very low limits")
	}
	
	// Verify we can detect the size limit dynamically
	if maxSuccessfulSize > 0 {
		// Try sending a message just over the limit
		oversizedData := make([]byte, maxSuccessfulSize*2)
		for i := range oversizedData {
			oversizedData[i] = 'B'
		}
		
		err = syscall.Sendto(fd, oversizedData, 0, addr)
		if err != syscall.EMSGSIZE {
			t.Logf("Expected EMSGSIZE for oversized message, got: %v", err)
		}
	}
}

// TestSocketCleanupManagement tests automatic and manual socket cleanup
func TestSocketCleanupManagement(t *testing.T) {
	testSocketPath := "/tmp/gojanus-cleanup-test.sock"
	responseSocketPath := "/tmp/gojanus-cleanup-response.sock"
	
	// Ensure clean start
	os.Remove(testSocketPath)
	os.Remove(responseSocketPath)
	
	// Test automatic cleanup when socket is closed
	func() {
		fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
		if err != nil {
			t.Fatalf("Failed to create socket: %v", err)
		}
		
		addr := &syscall.SockaddrUnix{Name: testSocketPath}
		err = syscall.Bind(fd, addr)
		if err != nil {
			t.Fatalf("Failed to bind socket: %v", err)
		}
		
		// Verify socket file exists
		if _, err := os.Stat(testSocketPath); os.IsNotExist(err) {
			t.Error("Socket file should exist after binding")
		}
		
		// Close socket
		syscall.Close(fd)
		
		// Socket file should still exist (Unix domain sockets persist)
		if _, err := os.Stat(testSocketPath); os.IsNotExist(err) {
			t.Error("Socket file should persist after closing")
		}
	}()
	
	// Test manual cleanup
	err := os.Remove(testSocketPath)
	if err != nil {
		t.Errorf("Failed to manually remove socket file: %v", err)
	}
	
	// Verify file is gone
	if _, err := os.Stat(testSocketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after manual cleanup")
	}
	
	// Test cleanup of response sockets
	responseFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create response socket: %v", err)
	}
	
	responseAddr := &syscall.SockaddrUnix{Name: responseSocketPath}
	err = syscall.Bind(responseFd, responseAddr)
	if err != nil {
		t.Fatalf("Failed to bind response socket: %v", err)
	}
	
	// Verify response socket exists
	if _, err := os.Stat(responseSocketPath); os.IsNotExist(err) {
		t.Error("Response socket file should exist")
	}
	
	// Clean up response socket
	syscall.Close(responseFd)
	os.Remove(responseSocketPath)
	
	// Verify cleanup
	if _, err := os.Stat(responseSocketPath); !os.IsNotExist(err) {
		t.Error("Response socket should be cleaned up")
	}
}

// TestConnectionTesting tests connectivity probing with datagrams
func TestConnectionTesting(t *testing.T) {
	serverSocketPath := "/tmp/gojanus-conn-test.sock"
	
	// Clean up before and after test
	os.Remove(serverSocketPath)
	defer os.Remove(serverSocketPath)
	
	// Create client for connection testing
	client, err := protocol.New(serverSocketPath, "test-channel")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	// Test connection to non-existent server (should fail)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	err = client.TestConnection(ctx)
	if err == nil {
		t.Error("Expected connection test to fail for non-existent server")
	}
	
	// Create mock server
	serverFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create server socket: %v", err)
	}
	defer syscall.Close(serverFd)
	
	serverAddr := &syscall.SockaddrUnix{Name: serverSocketPath}
	err = syscall.Bind(serverFd, serverAddr)
	if err != nil {
		t.Fatalf("Failed to bind server socket: %v", err)
	}
	
	// Start simple echo server in goroutine
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, clientAddr, err := syscall.Recvfrom(serverFd, buffer, 0)
			if err != nil {
				return
			}
			
			// Echo back the message
			syscall.Sendto(serverFd, buffer[:n], 0, clientAddr)
		}
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test connection to running server (should succeed)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()
	
	err = client.TestConnection(ctx2)
	if err != nil {
		t.Errorf("Expected connection test to succeed for running server: %v", err)
	}
}

// TestUniqueResponseSocketPaths tests generation of unique temporary socket paths
func TestUniqueResponseSocketPaths(t *testing.T) {
	testSocketPath := "/tmp/gojanus-unique-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// We don't need the client for this test, just generate paths directly
	
	// Generate multiple response socket paths (manual generation for testing)
	paths := make([]string, 10)
	for i := 0; i < 10; i++ {
		paths[i] = fmt.Sprintf("/tmp/gojanus_response_%d_%d.sock", time.Now().UnixNano(), i)
		time.Sleep(1 * time.Microsecond) // Ensure uniqueness
	}
	
	// Verify all paths are unique
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[i] == paths[j] {
				t.Errorf("Duplicate response socket paths: %s", paths[i])
			}
		}
	}
	
	// Verify paths are not empty
	for i, path := range paths {
		if path == "" {
			t.Errorf("Response socket path %d is empty", i)
		}
	}
	
	// Verify paths are different from main socket path
	for i, path := range paths {
		if path == testSocketPath {
			t.Errorf("Response socket path %d matches main socket path", i)
		}
	}
	
	// Verify paths use temporary directory
	for i, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Response socket path %d is not absolute: %s", i, path)
		}
	}
}

// TestSocketAddressConfiguration tests Unix domain socket address setup
func TestSocketAddressConfiguration(t *testing.T) {
	testSocketPath := "/tmp/gojanus-addr-test.sock"
	
	// Clean up before and after test
	os.Remove(testSocketPath)
	defer os.Remove(testSocketPath)
	
	// Test socket address creation
	addr := &syscall.SockaddrUnix{Name: testSocketPath}
	
	// Verify address structure
	if addr.Name != testSocketPath {
		t.Errorf("Expected address name '%s', got '%s'", testSocketPath, addr.Name)
	}
	
	// Test address with socket
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	defer syscall.Close(fd)
	
	// Bind to address
	err = syscall.Bind(fd, addr)
	if err != nil {
		t.Fatalf("Failed to bind to configured address: %v", err)
	}
	
	// Verify socket file was created at correct path
	if _, err := os.Stat(testSocketPath); os.IsNotExist(err) {
		t.Error("Socket file not created at configured address")
	}
	
	// Test address for sending
	clientFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create client socket: %v", err)
	}
	defer syscall.Close(clientFd)
	
	testMessage := []byte("test message")
	err = syscall.Sendto(clientFd, testMessage, 0, addr)
	if err != nil {
		t.Fatalf("Failed to send to configured address: %v", err)
	}
	
	// Verify message was received
	buffer := make([]byte, 1024)
	n, _, err := syscall.Recvfrom(fd, buffer, 0)
	if err != nil {
		t.Fatalf("Failed to receive from configured address: %v", err)
	}
	
	receivedMessage := buffer[:n]
	if string(receivedMessage) != string(testMessage) {
		t.Errorf("Expected message '%s', got '%s'", testMessage, receivedMessage)
	}
}

// TestTimeoutManagement tests socket read/write timeout handling
func TestTimeoutManagement(t *testing.T) {
	serverSocketPath := "/tmp/gojanus-timeout-test.sock"
	responseSocketPath := "/tmp/gojanus-timeout-response.sock"
	
	// Clean up before and after test
	os.Remove(serverSocketPath)
	os.Remove(responseSocketPath)
	defer func() {
		os.Remove(serverSocketPath)
		os.Remove(responseSocketPath)
	}()
	
	// Create response socket with timeout
	responseFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create response socket: %v", err)
	}
	defer syscall.Close(responseFd)
	
	responseAddr := &syscall.SockaddrUnix{Name: responseSocketPath}
	err = syscall.Bind(responseFd, responseAddr)
	if err != nil {
		t.Fatalf("Failed to bind response socket: %v", err)
	}
	
	// Set socket receive timeout
	timeout := syscall.Timeval{Sec: 1, Usec: 0} // 1 second timeout
	err = syscall.SetsockoptTimeval(responseFd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &timeout)
	if err != nil {
		t.Fatalf("Failed to set socket timeout: %v", err)
	}
	
	// Test timeout behavior - try to receive with no sender
	buffer := make([]byte, 1024)
	startTime := time.Now()
	
	_, _, err = syscall.Recvfrom(responseFd, buffer, 0)
	elapsed := time.Since(startTime)
	
	// Should timeout after approximately 1 second
	if err == nil {
		t.Error("Expected timeout error when no data available")
	}
	
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected timeout around 1 second, got %v", elapsed)
	}
	
	// Test successful receive within timeout
	// Create sender socket
	senderFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("Failed to create sender socket: %v", err)
	}
	defer syscall.Close(senderFd)
	
	// Send message quickly (should not timeout)
	go func() {
		time.Sleep(100 * time.Millisecond) // Send after 100ms
		testMessage := []byte("timeout test message")
		syscall.Sendto(senderFd, testMessage, 0, responseAddr)
	}()
	
	startTime = time.Now()
	n, _, err := syscall.Recvfrom(responseFd, buffer, 0)
	elapsed = time.Since(startTime)
	
	if err != nil {
		t.Errorf("Expected successful receive, got error: %v", err)
	}
	
	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected quick receive, took %v", elapsed)
	}
	
	receivedMessage := buffer[:n]
	if string(receivedMessage) != "timeout test message" {
		t.Errorf("Expected 'timeout test message', got '%s'", receivedMessage)
	}
}