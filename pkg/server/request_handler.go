package server

import (
	"fmt"
	"GoJanus/pkg/models"
)

// HandlerResult represents the result of a handler execution
type HandlerResult struct {
	Value interface{}
	Error *models.JSONRPCError
}

// RequestHandler supports direct value responses and flexible error handling
type RequestHandler interface {
	Handle(*models.JanusRequest) HandlerResult
}

// SyncHandler wraps a synchronous handler function
type SyncHandler func(*models.JanusRequest) HandlerResult

func (h SyncHandler) Handle(cmd *models.JanusRequest) HandlerResult {
	return h(cmd)
}

// AsyncHandler wraps an asynchronous handler function using goroutines
type AsyncHandler func(*models.JanusRequest, chan<- HandlerResult)

func (h AsyncHandler) Handle(cmd *models.JanusRequest) HandlerResult {
	resultChan := make(chan HandlerResult, 1)
	go h(cmd, resultChan)
	return <-resultChan
}

// Direct response handlers for common types
type BoolHandler func(*models.JanusRequest) (bool, error)
type StringHandler func(*models.JanusRequest) (string, error)
type IntHandler func(*models.JanusRequest) (int, error)
type FloatHandler func(*models.JanusRequest) (float64, error)
type ArrayHandler func(*models.JanusRequest) ([]interface{}, error)
type ObjectHandler func(*models.JanusRequest) (map[string]interface{}, error)

// CustomHandler for any JSON-serializable type
type CustomHandler[T any] func(*models.JanusRequest) (T, error)

// Convenience constructors for direct value handlers

func NewBoolHandler(fn BoolHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

func NewStringHandler(fn StringHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

func NewIntHandler(fn IntHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

func NewFloatHandler(fn FloatHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

func NewArrayHandler(fn ArrayHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

func NewObjectHandler(fn ObjectHandler) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

// NewCustomHandler creates a handler for any JSON-serializable type
func NewCustomHandler[T any](fn CustomHandler[T]) RequestHandler {
	return SyncHandler(func(cmd *models.JanusRequest) HandlerResult {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				return HandlerResult{Error: jsonRPCErr}
			}
			return HandlerResult{Error: &models.JSONRPCError{
				Code:    models.InternalError,
				Message: err.Error(),
			}}
		}
		return HandlerResult{Value: value}
	})
}

// NewAsyncBoolHandler creates an async boolean handler
func NewAsyncBoolHandler(fn func(*models.JanusRequest) (bool, error)) RequestHandler {
	return AsyncHandler(func(cmd *models.JanusRequest, result chan<- HandlerResult) {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				result <- HandlerResult{Error: jsonRPCErr}
			} else {
				result <- HandlerResult{Error: &models.JSONRPCError{
					Code:    models.InternalError,
					Message: err.Error(),
				}}
			}
		} else {
			result <- HandlerResult{Value: value}
		}
	})
}

// NewAsyncStringHandler creates an async string handler
func NewAsyncStringHandler(fn func(*models.JanusRequest) (string, error)) RequestHandler {
	return AsyncHandler(func(cmd *models.JanusRequest, result chan<- HandlerResult) {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				result <- HandlerResult{Error: jsonRPCErr}
			} else {
				result <- HandlerResult{Error: &models.JSONRPCError{
					Code:    models.InternalError,
					Message: err.Error(),
				}}
			}
		} else {
			result <- HandlerResult{Value: value}
		}
	})
}

// NewAsyncCustomHandler creates an async handler for any JSON-serializable type
func NewAsyncCustomHandler[T any](fn func(*models.JanusRequest) (T, error)) RequestHandler {
	return AsyncHandler(func(cmd *models.JanusRequest, result chan<- HandlerResult) {
		value, err := fn(cmd)
		if err != nil {
			// If error is already a JSONRPCError, preserve it
			if jsonRPCErr, ok := err.(*models.JSONRPCError); ok {
				result <- HandlerResult{Error: jsonRPCErr}
			} else {
				result <- HandlerResult{Error: &models.JSONRPCError{
					Code:    models.InternalError,
					Message: err.Error(),
				}}
			}
		} else {
			result <- HandlerResult{Value: value}
		}
	})
}

// SerializeResponse converts a HandlerResult to a JSON-serializable response
func SerializeResponse(result HandlerResult) (interface{}, *models.JSONRPCError) {
	if result.Error != nil {
		return nil, result.Error
	}
	
	// Direct serialization without dictionary wrapping
	return result.Value, nil
}

// Enhanced handler registry with type safety
type HandlerRegistry struct {
	handlers map[string]RequestHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]RequestHandler),
	}
}

func (r *HandlerRegistry) RegisterHandler(request string, handler RequestHandler) error {
	// List of reserved built-in requests that cannot be overridden
	builtinRequests := []string{"ping", "echo", "get_info", "manifest", "validate", "slow_process"}
	
	// Check if trying to override a built-in request
	for _, builtin := range builtinRequests {
		if request == builtin {
			return fmt.Errorf("cannot override built-in request: %s", request)
		}
	}
	
	r.handlers[request] = handler
	return nil
}

func (r *HandlerRegistry) UnregisterHandler(request string) {
	delete(r.handlers, request)
}

func (r *HandlerRegistry) GetHandler(request string) (RequestHandler, bool) {
	handler, exists := r.handlers[request]
	return handler, exists
}

func (r *HandlerRegistry) HasHandler(request string) bool {
	_, exists := r.handlers[request]
	return exists
}

func (r *HandlerRegistry) ExecuteHandler(request string, cmd *models.JanusRequest) (interface{}, *models.JSONRPCError) {
	handler, exists := r.GetHandler(request)
	if !exists {
		return nil, &models.JSONRPCError{
			Code:    models.MethodNotFound,
			Message: "Method not found",
			Data: &models.JSONRPCErrorData{
				Details: "Request not found: " + request,
				Context: map[string]interface{}{"method": request},
			},
		}
	}
	
	result := handler.Handle(cmd)
	return SerializeResponse(result)
}