package response

import (
	"github.com/gin-gonic/gin"
)

type ErrorDetails struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

func JSONError(c *gin.Context, status int, code string, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func JSONSuccess(c *gin.Context, status int, data interface{}) {
	if data == nil {
		c.Status(status)
		return
	}
	c.JSON(status, data)
}

// Common error codes
const (
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)
