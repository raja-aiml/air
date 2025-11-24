package errors

import (
	"errors"
	"fmt"
)

// AppError represents a structured application error with context
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	underlying error
}

// Error codes for consistent error handling
const (
	// Protocol errors
	ErrCodeInvalidEnvelope = "protocol.invalid_envelope"
	ErrCodeInvalidEvent    = "protocol.invalid_event"
	ErrCodeUnknownEvent    = "protocol.unknown_event"
	ErrCodeUnimplemented   = "protocol.unimplemented"
	ErrCodeInvalidPayload  = "protocol.invalid_payload"
	ErrCodePayloadTooLarge = "protocol.payload_too_large"

	// Auth errors
	ErrCodeAuthFailed   = "auth.failed"
	ErrCodeInvalidToken = "auth.invalid_token"
	ErrCodeTokenExpired = "auth.token_expired"
	ErrCodeTokenMissing = "auth.token_missing"
	ErrCodeUnauthorized = "auth.unauthorized"

	// Database errors
	ErrCodeDatabaseUnavailable = "db.unavailable"
	ErrCodeDatabaseQuery       = "db.query_failed"
	ErrCodeDatabaseConstraint  = "db.constraint_violation"
	ErrCodeNotFound            = "db.not_found"

	// Rate limiting
	ErrCodeRateLimited = "rate_limit.exceeded"

	// Internal errors
	ErrCodeInternal = "internal.error"
)

// Error implements the error interface
func (e *AppError) Error() string {
	if e.underlying != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.underlying)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As
func (e *AppError) Unwrap() error {
	return e.underlying
}

// WithDetail adds contextual details to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds request ID for correlation
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// New creates a new AppError
func New(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with application context
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		underlying: err,
	}
}

// Is checks if an error matches a specific code
func Is(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// Common error constructors

func InvalidEnvelope(message string) *AppError {
	return New(ErrCodeInvalidEnvelope, message)
}

func InvalidEvent(eventName string) *AppError {
	return New(ErrCodeInvalidEvent, fmt.Sprintf("invalid event: %s", eventName))
}

func UnknownEvent(eventName string) *AppError {
	return New(ErrCodeUnknownEvent, fmt.Sprintf("unknown event: %s", eventName))
}

func Unimplemented(feature string) *AppError {
	return New(ErrCodeUnimplemented, fmt.Sprintf("%s not implemented", feature))
}

func InvalidPayload(message string) *AppError {
	return New(ErrCodeInvalidPayload, message)
}

func PayloadTooLarge(size, maxSize int) *AppError {
	return New(ErrCodePayloadTooLarge, "payload exceeds maximum size").
		WithDetail("size", size).
		WithDetail("max_size", maxSize)
}

func AuthFailed(message string) *AppError {
	return New(ErrCodeAuthFailed, message)
}

func InvalidToken(message string) *AppError {
	return New(ErrCodeInvalidToken, message)
}

func TokenMissing() *AppError {
	return New(ErrCodeTokenMissing, "authentication token required")
}

func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

func DatabaseUnavailable(err error) *AppError {
	return Wrap(err, ErrCodeDatabaseUnavailable, "database unavailable")
}

func DatabaseQuery(err error, query string) *AppError {
	return Wrap(err, ErrCodeDatabaseQuery, "database query failed").
		WithDetail("query", query)
}

func NotFound(resource string) *AppError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

func RateLimited(resource string) *AppError {
	return New(ErrCodeRateLimited, fmt.Sprintf("rate limit exceeded for %s", resource))
}

func Internal(err error) *AppError {
	return Wrap(err, ErrCodeInternal, "internal server error")
}
