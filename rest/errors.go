package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a structured error response from the RouterOS REST API.
// RouterOS returns errors as JSON objects with the following format:
//
//	{"error": 404, "message": "Not Found", "detail": "no such command or directory"}
//
// Use type assertion to access error details:
//
//	if apiErr, ok := err.(*rest.APIError); ok {
//	    fmt.Printf("Status: %d, Message: %s\n", apiErr.StatusCode, apiErr.Message)
//	}
type APIError struct {
	StatusCode int    `json:"error"`   // HTTP status code (e.g., 400, 404, 500)
	Message    string `json:"message"` // short error description (e.g., "Not Found")
	Detail     string `json:"detail"`  // detailed error explanation from RouterOS
}

// Error implements the error interface.
// If Detail is present, it is included in the error message.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("routeros: %d %s: %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("routeros: %d %s", e.StatusCode, e.Message)
}

// parseAPIError reads the response body and returns an *APIError.
// It attempts to parse the body as a RouterOS JSON error object.
// If the body is not valid JSON, the raw body text is used as the message.
// If the body is empty, the standard HTTP status text is used.
func parseAPIError(resp *http.Response) error {
	// Read the full response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read response body: %v", err),
		}
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}

	// Empty body: use the standard HTTP status text (e.g., "Not Found" for 404).
	if len(body) == 0 {
		apiErr.Message = http.StatusText(resp.StatusCode)
		return apiErr
	}

	// Try to parse as RouterOS JSON error. If parsing fails, use raw body text.
	if err := json.Unmarshal(body, apiErr); err != nil {
		apiErr.Message = string(body)
	}

	return apiErr
}
