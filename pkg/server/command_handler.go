package server

import (
	"github.com/user/GoJanus/pkg/models"
)

// HandlerResult represents the result of a handler execution
type HandlerResult struct {
	Value interface{}
	Error *models.JSONRPCError
}

// CommandHandler supports direct value responses and flexible error handling
type CommandHandler interface {
	Handle(*models.JanusCommand) HandlerResult
}

// SyncHandler wraps a synchronous handler function
type SyncHandler func(*models.JanusCommand) HandlerResult

func (h SyncHandler) Handle(cmd *models.JanusCommand) HandlerResult {
	return h(cmd)
}

// AsyncHandler wraps an asynchronous handler function using goroutines
type AsyncHandler func(*models.JanusCommand, chan<- HandlerResult)

func (h AsyncHandler) Handle(cmd *models.JanusCommand) HandlerResult {
	resultChan := make(chan HandlerResult, 1)
	go h(cmd, resultChan)
	return <-resultChan
}

// Direct response handlers for common types
type BoolHandler func(*models.JanusCommand) (bool, error)
type StringHandler func(*models.JanusCommand) (string, error)
type IntHandler func(*models.JanusCommand) (int, error)
type FloatHandler func(*models.JanusCommand) (float64, error)
type ArrayHandler func(*models.JanusCommand) ([]interface{}, error)
type ObjectHandler func(*models.JanusCommand) (map[string]interface{}, error)

// CustomHandler for any JSON-serializable type
type CustomHandler[T any] func(*models.JanusCommand) (T, error)

// Convenience constructors for direct value handlers

func NewBoolHandler(fn BoolHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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

func NewStringHandler(fn StringHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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

func NewIntHandler(fn IntHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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

func NewFloatHandler(fn FloatHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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

func NewArrayHandler(fn ArrayHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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

func NewObjectHandler(fn ObjectHandler) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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
func NewCustomHandler[T any](fn CustomHandler[T]) CommandHandler {
	return SyncHandler(func(cmd *models.JanusCommand) HandlerResult {
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
func NewAsyncBoolHandler(fn func(*models.JanusCommand) (bool, error)) CommandHandler {
	return AsyncHandler(func(cmd *models.JanusCommand, result chan<- HandlerResult) {
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
func NewAsyncStringHandler(fn func(*models.JanusCommand) (string, error)) CommandHandler {
	return AsyncHandler(func(cmd *models.JanusCommand, result chan<- HandlerResult) {
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
func NewAsyncCustomHandler[T any](fn func(*models.JanusCommand) (T, error)) CommandHandler {
	return AsyncHandler(func(cmd *models.JanusCommand, result chan<- HandlerResult) {
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
	handlers map[string]CommandHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]CommandHandler),
	}
}

func (r *HandlerRegistry) RegisterHandler(command string, handler CommandHandler) {
	r.handlers[command] = handler
}

func (r *HandlerRegistry) UnregisterHandler(command string) {
	delete(r.handlers, command)
}

func (r *HandlerRegistry) GetHandler(command string) (CommandHandler, bool) {
	handler, exists := r.handlers[command]
	return handler, exists
}

func (r *HandlerRegistry) ExecuteHandler(command string, cmd *models.JanusCommand) (interface{}, *models.JSONRPCError) {
	handler, exists := r.GetHandler(command)
	if !exists {
		return nil, &models.JSONRPCError{
			Code:    models.MethodNotFound,
			Message: "Method not found",
			Data: &models.JSONRPCErrorData{
				Details: "Command not found: " + command,
				Context: map[string]interface{}{"method": command},
			},
		}
	}
	
	result := handler.Handle(cmd)
	return SerializeResponse(result)
}