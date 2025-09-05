package apperrors

import "fmt"

// ErrorCode defines common application error codes.
// Use these to standardize user-facing API response statuses.
type ErrorCode string

const (
	CodeNotFound     ErrorCode = "NOT_FOUND"       // 404
	CodeInvalidInput ErrorCode = "INVALID_INPUT"   // 400
	CodeUnauthorized ErrorCode = "UNAUTHORIZED"    // 401
	CodeForbidden    ErrorCode = "FORBIDDEN"       // 403
	CodeConflict     ErrorCode = "CONFLICT"        // 409
	CodeInternal     ErrorCode = "INTERNAL_SERVER" // 500
	CodeUnknown      ErrorCode = "UNKNOWN"         // 500
)

// Error captures details about an application or domain-specific error.
type Error struct {
	Code    ErrorCode      // type/category of the error
	Message string         // user-readable message
	Cause   error          // optional root cause (not exposed in API responses)
	Body    map[string]any // optional structured payload (e.g., validation fields)
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WrapError creates a new Error with an optional cause.
func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// GetErrorCode extracts an ErrorCode from an error if it's an *Error; otherwise UNKNOWN.
func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*Error); ok {
		return appErr.Code
	}
	return CodeUnknown
}
