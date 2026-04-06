// rest/errors.go
package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a structured error response from RouterOS REST API.
type APIError struct {
	StatusCode int    `json:"error"`
	Message    string `json:"message"`
	Detail     string `json:"detail"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("routeros: %d %s: %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("routeros: %d %s", e.StatusCode, e.Message)
}

// parseAPIError reads the response body and returns an *APIError.
// If the body is valid RouterOS JSON error, fields are populated from it.
// If not, Message is set to the raw body text.
func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read response body: %v", err),
		}
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}

	if len(body) == 0 {
		apiErr.Message = http.StatusText(resp.StatusCode)
		return apiErr
	}

	if err := json.Unmarshal(body, apiErr); err != nil {
		apiErr.Message = string(body)
	}

	return apiErr
}
