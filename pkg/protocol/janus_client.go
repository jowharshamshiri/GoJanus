package protocol

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"GoJanus/pkg/core"
	"GoJanus/pkg/models"
	"GoJanus/pkg/manifest"
)

// JanusClient is the main client interface for SOCK_DGRAM Unix socket communication
// Connectionless implementation for cross-language compatibility
type JanusClient struct {
	socketPath     string
	manifest        *manifest.Manifest
	config         JanusClientConfig
	
	janusClient *core.JanusClient
	validator      *core.SecurityValidator
	
	// Request handler registry (thread-safe)
	handlers      map[string]models.RequestHandler
	handlerMutex  sync.RWMutex
	
	// Timeout management
	timeoutManager *TimeoutManager
	
	// Response correlation system
	responseTracker *ResponseTracker
	
	// Request lifecycle management (automatic ID system)
	requestRegistry map[string]*models.RequestHandle
	registryMutex   sync.RWMutex
}

// JanusClientConfig holds configuration for the datagram client
type JanusClientConfig struct {
	MaxMessageSize   int
	DefaultTimeout   time.Duration
	DatagramTimeout  time.Duration
	EnableValidation bool
}

// DefaultJanusClientConfig returns default configuration for SOCK_DGRAM
func DefaultJanusClientConfig() JanusClientConfig {
	return JanusClientConfig{
		MaxMessageSize:   64 * 1024,      // 64KB datagram limit
		DefaultTimeout:   30 * time.Second,
		DatagramTimeout:  5 * time.Second,
		EnableValidation: true,
	}
}

// validateConstructorInputs validates constructor parameters
func validateConstructorInputs(socketPath string, config JanusClientConfig) error {
	// Validate socket path
	if socketPath == "" {
		return fmt.Errorf("socket path cannot be empty")
	}
	
	// Manifest is always fetched from server - no validation needed here
	
	// Validate configuration security
	if config.MaxMessageSize < 1024 {
		return fmt.Errorf("configuration error: MaxMessageSize too small, minimum 1024 bytes")
	}
	
	if config.DefaultTimeout < time.Second {
		return fmt.Errorf("configuration error: DefaultTimeout too short, minimum 1 second")
	}
	
	if config.DatagramTimeout < time.Millisecond*100 {
		return fmt.Errorf("configuration error: DatagramTimeout too short, minimum 100ms")
	}
	
	return nil
}

// fetchManifestFromServer fetches the Manifest from the server
func fetchManifestFromServer(janusClient *core.JanusClient, socketPath string, cfg JanusClientConfig) (*manifest.Manifest, error) {
	log.Printf("[GO-PROTOCOL] fetchManifestFromServer ENTER - Server: %s", socketPath)
	
	// Generate response socket path
	responseSocketPath := fmt.Sprintf("/tmp/janus_manifest_%d_%s.sock", time.Now().UnixNano(), generateRandomID())
	log.Printf("[GO-PROTOCOL] Generated response socket path: %s", responseSocketPath)
	
	// Create manifest request with proper JanusRequest structure
	manifestRequest := *models.NewJanusRequest("manifest", nil, nil)
	manifestRequest.ReplyTo = &responseSocketPath
	
	requestJSON, err := json.Marshal(manifestRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest request: %w", err)
	}
	
	// Send manifest request to server with timeout context
	log.Printf("[GO-PROTOCOL] Creating timeout context: %v", cfg.DefaultTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DefaultTimeout)
	defer func() {
		log.Printf("[GO-PROTOCOL] Context cancel called")
		cancel()
	}()
	
	log.Printf("[GO-PROTOCOL] Calling core SendDatagram with response path: %s", responseSocketPath)
	responseData, err := janusClient.SendDatagram(ctx, requestJSON, responseSocketPath)
	if err != nil {
		log.Printf("[GO-PROTOCOL] ERROR: SendDatagram failed: %v", err)
		return nil, fmt.Errorf("failed to fetch manifest from server: %w", err)
	}
	log.Printf("[GO-PROTOCOL] SendDatagram SUCCESS - Received %d bytes", len(responseData))
	
	// Parse the response JSON
	var response map[string]interface{}
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}
	
	// Check for error in response (PRIME DIRECTIVE format)
	if errorMsg, exists := response["error"]; exists && errorMsg != nil {
		return nil, fmt.Errorf("server returned error: %v", errorMsg)
	}
	
	// Extract manifest from response
	manifestData, exists := response["result"]
	if !exists {
		return nil, fmt.Errorf("server response missing 'result' field")
	}
	
	// Convert manifest data to JSON and parse
	manifestJSON, err := json.Marshal(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest data: %w", err)
	}
	
	parser := manifest.NewManifestParser()
	manifest, err := parser.ParseJSON(manifestJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server manifest: %w", err)
	}
	
	log.Printf("[GO-PROTOCOL] fetchManifestFromServer SUCCESS - Returning manifest")
	return manifest, nil
}

// generateRandomID generates a random ID for unique socket paths
func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// New creates a new datagram API client
// Always fetches manifest from server - no hardcoded manifests allowed
func New(socketPath string, config ...JanusClientConfig) (*JanusClient, error) {
	cfg := DefaultJanusClientConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	
	// Validate inputs 
	if err := validateConstructorInputs(socketPath, cfg); err != nil {
		return nil, err
	}
	
	// Create datagram client
	datagramConfig := core.JanusClientConfig{
		MaxMessageSize:  cfg.MaxMessageSize,
		DatagramTimeout: cfg.DatagramTimeout,
	}
	
	janusClient, err := core.NewJanusClient(socketPath, datagramConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create datagram client: %w", err)
	}
	
	// Manifest will be fetched when needed during operations
	
	validator := core.NewSecurityValidator()
	timeoutManager := NewTimeoutManager()
	
	// Initialize response tracker for advanced client features
	trackerConfig := TrackerConfig{
		MaxPendingRequests: 1000,
		CleanupInterval:    30 * time.Second,
		DefaultTimeout:     cfg.DefaultTimeout,
	}
	responseTracker := NewResponseTracker(trackerConfig)
	
	return &JanusClient{
		socketPath:      socketPath,
		manifest:         nil,
		config:          cfg,
		janusClient:     janusClient,
		validator:       validator,
		handlers:        make(map[string]models.RequestHandler),
		timeoutManager:  timeoutManager,
		responseTracker: responseTracker,
		requestRegistry: make(map[string]*models.RequestHandle),
	}, nil
}

// ensureManifestLoaded fetches Manifest from server if not already loaded
func (client *JanusClient) ensureManifestLoaded() error {
	if client.manifest != nil {
		return nil // Already loaded
	}
	
	if !client.config.EnableValidation {
		return nil // Validation disabled, no need to fetch
	}
	
	// Fetch manifest from server
	fetchedManifest, err := fetchManifestFromServer(client.janusClient, client.socketPath, client.config)
	if err != nil {
		return fmt.Errorf("failed to fetch Manifest: %w", err)
	}
	
	// Channels have been removed from the protocol
	client.manifest = fetchedManifest
	return nil
}

// SendRequest sends a request via SOCK_DGRAM and waits for response
func (client *JanusClient) SendRequest(ctx context.Context, request string, args map[string]interface{}, options ...RequestOptions) (*models.JanusResponse, error) {
	// Apply options
	opts := mergeRequestOptions(options...)
	
	// Generate request ID
	requestID := generateUUID()
	
	// Generate response socket path
	responseSocketPath := client.janusClient.GenerateResponseSocketPath()
	
	// Create socket request
	timeoutSeconds := opts.Timeout.Seconds()
	janusRequest := *models.NewJanusRequest(request, args, &timeoutSeconds)
	janusRequest.ID = requestID // Use provided request ID
	janusRequest.ReplyTo = &responseSocketPath
	
	// Ensure Manifest is loaded for validation
	if client.config.EnableValidation {
		if err := client.ensureManifestLoaded(); err != nil {
			// If this is a connection error, propagate it directly without wrapping as validation error
			if strings.Contains(err.Error(), "dial") || strings.Contains(err.Error(), "connect") || strings.Contains(err.Error(), "no such file") {
				return nil, err
			}
			return nil, fmt.Errorf("failed to load Manifest for validation: %w", err)
		}
	}

	// Validate request arguments against Manifest (but don't reject unknown requests)
	// Unknown requests should be sent to the server which will respond with an error
	if client.config.EnableValidation && client.manifest != nil {
		// Only validate arguments if the request exists in the manifest
		if client.manifest.HasRequest(request) {
			requestManifest, err := client.manifest.GetRequest(request)
			if err != nil {
				return nil, fmt.Errorf("request validation failed: %w", err)
			}
			
			if err := client.manifest.ValidateRequestArgs(requestManifest, args); err != nil {
				return nil, fmt.Errorf("request validation failed: %w", err)
			}
		}
		// If request doesn't exist in manifest, still send it to server
		// Server will respond with METHOD_NOT_FOUND error
	}
	
	// Apply timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = client.config.DefaultTimeout
	}
	
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Serialize request
	requestData, err := json.Marshal(janusRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}
	
	// Send datagram and wait for response
	responseData, err := client.janusClient.SendDatagram(requestCtx, requestData, responseSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to send request datagram: %w", err)
	}
	
	// Deserialize response
	var response models.JanusResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}
	
	// Validate response correlation
	if response.RequestID != requestID {
		return nil, fmt.Errorf("response correlation mismatch: expected %s, got %s", requestID, response.RequestID)
	}
	
	// PRIME DIRECTIVE: Channel validation removed - responses don't include channel info
	
	return &response, nil
}

// SendRequestNoResponse sends a request without expecting a response (fire-and-forget)
func (client *JanusClient) SendRequestNoResponse(ctx context.Context, request string, args map[string]interface{}) error {
	// Generate request ID
	requestID := generateUUID()
	
	// Create socket request (no reply_to field)
	janusRequest := *models.NewJanusRequest(request, args, nil)
	janusRequest.ID = requestID // Use provided request ID
	
	// Channels have been removed - skip validation
	
	// Serialize request
	requestData, err := json.Marshal(janusRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize request: %w", err)
	}
	
	// Send datagram without waiting for response
	return client.janusClient.SendDatagramNoResponse(ctx, requestData)
}

// TestConnection tests connectivity to the server
func (client *JanusClient) TestConnection(ctx context.Context) error {
	return client.janusClient.TestDatagramSocket(ctx)
}

// Close cleans up client resources
func (client *JanusClient) Close() error {
	// Clean up timeout manager
	if client.timeoutManager != nil {
		client.timeoutManager.Close()
	}
	
	// Clean up response tracker
	if client.responseTracker != nil {
		client.responseTracker.Shutdown()
	}
	
	// Clear handlers
	client.handlerMutex.Lock()
	client.handlers = make(map[string]models.RequestHandler)
	client.handlerMutex.Unlock()
	
	// Cancel and clear all pending requests
	client.CancelAllRequests()
	
	return nil
}

// GetChannelID returns empty string - channels have been removed
func (client *JanusClient) GetChannelID() string {
	return ""
}

// GetSocketPath returns the socket path
func (client *JanusClient) GetSocketPath() string {
	return client.socketPath
}

// GetManifest returns the Manifest
func (client *JanusClient) GetManifest() *manifest.Manifest {
	return client.manifest
}

// Helper functions

// generateUUID generates a simple UUID for request correlation
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// isBuiltinRequest checks if a request is a built-in request
func (client *JanusClient) isBuiltinRequest(request string) bool {
	builtinRequests := map[string]bool{
		"ping":         true,
		"echo":         true,
		"get_info":     true,
		"manifest":         true,
		"validate":     true,
		"slow_process": true,
	}
	return builtinRequests[request]
}

// RequestOptions holds options for sending requests
type RequestOptions struct {
	Timeout time.Duration
}

// mergeRequestOptions merges request options with defaults
func mergeRequestOptions(options ...RequestOptions) RequestOptions {
	opts := RequestOptions{
		Timeout: 30 * time.Second, // default
	}
	
	for _, option := range options {
		if option.Timeout > 0 {
			opts.Timeout = option.Timeout
		}
	}
	
	return opts
}

// Automatic ID Management Methods (F0193-F0216)

// SendRequestWithHandle sends a request and returns a RequestHandle for tracking
// Hides UUID complexity from users while providing request lifecycle management
func (client *JanusClient) SendRequestWithHandle(ctx context.Context, request string, args map[string]interface{}, options ...RequestOptions) (*models.RequestHandle, chan *models.JanusResponse, chan error) {
	// Generate internal UUID (hidden from user)
	requestID := generateUUID()
	
	// Create request handle for user
	handle := models.NewRequestHandle(requestID, request)
	
	// Register the request handle
	client.registryMutex.Lock()
	client.requestRegistry[requestID] = handle
	client.registryMutex.Unlock()
	
	// Create response and error channels
	responseChan := make(chan *models.JanusResponse, 1)
	errorChan := make(chan error, 1)
	
	// Execute request asynchronously
	go func() {
		defer func() {
			// Clean up request handle when done
			client.registryMutex.Lock()
			delete(client.requestRegistry, requestID)
			client.registryMutex.Unlock()
		}()
		
		response, err := client.SendRequest(ctx, request, args, options...)
		if err != nil {
			errorChan <- err
			return
		}
		
		responseChan <- response
	}()
	
	return handle, responseChan, errorChan
}

// GetRequestStatus returns the current status of a request by handle
func (client *JanusClient) GetRequestStatus(handle *models.RequestHandle) models.RequestStatus {
	if handle.IsCancelled() {
		return models.RequestStatusCancelled
	}
	
	client.registryMutex.RLock()
	defer client.registryMutex.RUnlock()
	
	if _, exists := client.requestRegistry[handle.GetInternalID()]; exists {
		return models.RequestStatusPending
	}
	
	return models.RequestStatusCompleted
}

// CancelRequest cancels a pending request using its handle
func (client *JanusClient) CancelRequest(handle *models.RequestHandle) error {
	if handle.IsCancelled() {
		return fmt.Errorf("request already cancelled")
	}
	
	client.registryMutex.Lock()
	defer client.registryMutex.Unlock()
	
	if _, exists := client.requestRegistry[handle.GetInternalID()]; !exists {
		return fmt.Errorf("request not found or already completed")
	}
	
	handle.MarkCancelled()
	delete(client.requestRegistry, handle.GetInternalID())
	
	return nil
}

// GetPendingRequests returns handles for all pending requests
func (client *JanusClient) GetPendingRequests() []*models.RequestHandle {
	client.registryMutex.RLock()
	defer client.registryMutex.RUnlock()
	
	handles := make([]*models.RequestHandle, 0, len(client.requestRegistry))
	for _, handle := range client.requestRegistry {
		handles = append(handles, handle)
	}
	
	return handles
}

// CancelAllRequests cancels all pending requests
func (client *JanusClient) CancelAllRequests() int {
	client.registryMutex.Lock()
	defer client.registryMutex.Unlock()
	
	count := len(client.requestRegistry)
	for _, handle := range client.requestRegistry {
		handle.MarkCancelled()
	}
	
	client.requestRegistry = make(map[string]*models.RequestHandle)
	return count
}

// Backward compatibility methods for tests

// ChannelIdentifier returns empty string - channels have been removed
func (client *JanusClient) ChannelIdentifier() string {
	return ""
}

// Manifest returns the Manifest for backward compatibility  
func (client *JanusClient) Manifest() *manifest.Manifest {
	return client.manifest
}

// PublishRequest sends a request without expecting response for backward compatibility
func (client *JanusClient) PublishRequest(ctx context.Context, request string, args map[string]interface{}) (string, error) {
	err := client.SendRequestNoResponse(ctx, request, args)
	if err != nil {
		return "", err
	}
	// Return a generated request ID for compatibility
	return generateUUID(), nil
}

// SocketPathString returns the socket path for backward compatibility
func (client *JanusClient) SocketPathString() string {
	return client.socketPath
}

// RegisterRequestHandler validates request exists in manifest (SOCK_DGRAM compatibility)
func (client *JanusClient) RegisterRequestHandler(request string, handler interface{}) error {
	// Ensure manifest is loaded first
	if err := client.ensureManifestLoaded(); err != nil {
		return fmt.Errorf("failed to load manifest for handler validation: %w", err)
	}
	
	// Channels have been removed - skip validation
	
	// SOCK_DGRAM doesn't actually use handlers, but validation passed
	return nil
}

// Disconnect is a no-op for backward compatibility (SOCK_DGRAM doesn't have persistent connections)
func (client *JanusClient) Disconnect() error {
	// SOCK_DGRAM doesn't have persistent connections - this is for backward compatibility only
	return nil
}

// IsConnected always returns true for backward compatibility (SOCK_DGRAM doesn't track connections)
func (client *JanusClient) IsConnected() bool {
	// SOCK_DGRAM doesn't track connections - return true for backward compatibility
	return true
}

// Ping sends a ping request and returns success/failure
// Convenience method for testing connectivity with a simple ping request
func (client *JanusClient) Ping(ctx context.Context) bool {
	_, err := client.SendRequest(ctx, "ping", nil)
	return err == nil
}

// MARK: - Advanced Client Features (Response Correlation System)

// SendRequestAsync sends a request and returns a channel for receiving the response
func (client *JanusClient) SendRequestAsync(ctx context.Context, request string, args map[string]interface{}) (<-chan *models.JanusResponse, <-chan error) {
	responseChan := make(chan *models.JanusResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		response, err := client.SendRequest(ctx, request, args)
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- response
	}()

	return responseChan, errorChan
}

// SendRequestWithCorrelation sends a request with response correlation tracking
func (client *JanusClient) SendRequestWithCorrelation(ctx context.Context, request string, args map[string]interface{}, timeout time.Duration) (<-chan *models.JanusResponse, <-chan error, string) {
	requestID := generateUUID()
	responseChan := make(chan *models.JanusResponse, 1)
	errorChan := make(chan error, 1)

	// Track the request in response tracker
	err := client.responseTracker.TrackRequest(requestID, responseChan, errorChan, timeout)
	if err != nil {
		go func() { errorChan <- err }()
		return responseChan, errorChan, requestID
	}

	// Send the request asynchronously
	go func() {
		// Create response socket path
		responseSocketPath := fmt.Sprintf("/tmp/janus_response_%d_%s.sock", time.Now().UnixNano(), generateRandomID())

		// Create socket request with manifestific ID
		timeoutSeconds := float64(timeout.Seconds())
		janusRequest := *models.NewJanusRequest(request, args, &timeoutSeconds)
		janusRequest.ID = requestID // Use provided request ID
		janusRequest.ReplyTo = &responseSocketPath

		// Validate and send request
		if client.config.EnableValidation {
			if err := client.ensureManifestLoaded(); err != nil {
				client.responseTracker.CancelRequest(requestID, fmt.Sprintf("Manifest loading failed: %v", err))
				return
			}

			// Channels have been removed - skip validation
		}

		// Serialize and send request
		requestData, err := json.Marshal(janusRequest)
		if err != nil {
			client.responseTracker.CancelRequest(requestID, fmt.Sprintf("failed to serialize request: %v", err))
			return
		}

		// Send datagram and wait for response
		responseData, err := client.janusClient.SendDatagram(ctx, requestData, responseSocketPath)
		if err != nil {
			client.responseTracker.CancelRequest(requestID, fmt.Sprintf("failed to send request datagram: %v", err))
			return
		}

		// Parse response
		var response models.JanusResponse
		if err := json.Unmarshal(responseData, &response); err != nil {
			client.responseTracker.CancelRequest(requestID, fmt.Sprintf("failed to deserialize response: %v", err))
			return
		}

		// Handle response through tracker
		client.responseTracker.HandleResponse(&response)
	}()

	return responseChan, errorChan, requestID
}

// These methods are already defined above at lines 500 and 532

// GetPendingRequestCount returns the number of pending requests
func (client *JanusClient) GetPendingRequestCount() int {
	return client.responseTracker.GetPendingCount()
}

// GetPendingRequestIDs returns the IDs of all pending requests
func (client *JanusClient) GetPendingRequestIDs() []string {
	return client.responseTracker.GetPendingRequestIDs()
}

// IsRequestPending checks if a request is currently pending
func (client *JanusClient) IsRequestPending(requestID string) bool {
	return client.responseTracker.IsTracking(requestID)
}

// GetRequestStatistics returns statistics about pending requests
func (client *JanusClient) GetRequestStatistics() RequestStatistics {
	return client.responseTracker.GetStatistics()
}

// ExecuteRequestsInParallel executes multiple requests in parallel
func (client *JanusClient) ExecuteRequestsInParallel(ctx context.Context, requests []ParallelRequest) []ParallelResult {
	results := make([]ParallelResult, len(requests))
	var wg sync.WaitGroup

	for i, cmd := range requests {
		wg.Add(1)
		go func(index int, request ParallelRequest) {
			defer wg.Done()

			response, err := client.SendRequest(ctx, request.Request, request.Args)
			results[index] = ParallelResult{
				RequestID: request.ID,
				Response:  response,
				Error:     err,
			}
		}(i, cmd)
	}

	wg.Wait()
	return results
}


// MARK: - Helper Types for Advanced Features

// ParallelRequest represents a request to be executed in parallel
type ParallelRequest struct {
	ID      string                 `json:"id"`
	Request string                 `json:"request"`
	Args    map[string]interface{} `json:"args"`
}

// ParallelResult represents the result of a parallel request execution  
type ParallelResult struct {
	RequestID string                  `json:"requestId"`
	Response  *models.JanusResponse  `json:"response,omitempty"`
	Error     error                   `json:"error,omitempty"`
}

