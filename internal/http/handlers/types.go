package handlers

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error" example:"Internal Server Error"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation successful"`
}
