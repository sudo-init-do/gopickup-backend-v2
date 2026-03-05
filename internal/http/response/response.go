package response

import (
	"github.com/gin-gonic/gin"
)

// ErrorDetails holds the structure for error details
type ErrorDetails struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorResponse is the top-level error response structure
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

// JSONError sends a standardized error response
func JSONError(c *gin.Context, status int, code string, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// JSONSuccess sends a standardized success response
func JSONSuccess(c *gin.Context, status int, data interface{}) {
	if data == nil {
		c.Status(status)
		return
	}
	c.JSON(status, data)
}

// Common error codes used in API responses
const (
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)
