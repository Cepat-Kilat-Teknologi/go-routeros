// rest/errors_test.go
package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error_WithDetail(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Not Found",
		Detail:     "no such command or directory (remove)",
	}
	expected := "routeros: 404 Not Found: no such command or directory (remove)"
	assert.Equal(t, expected, err.Error())
}

func TestAPIError_Error_WithoutDetail(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "Bad Request",
	}
	expected := "routeros: 400 Bad Request"
	assert.Equal(t, expected, err.Error())
}

func TestAPIError_TypeAssertion(t *testing.T) {
	var err error = &APIError{StatusCode: 404, Message: "Not Found"}
	apiErr, ok := err.(*APIError)
	assert.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
}

func TestParseAPIError_ValidJSON(t *testing.T) {
	body := `{"error": 404, "message": "Not Found", "detail": "no such command"}`
	resp := &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	err := parseAPIError(resp)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "Not Found", apiErr.Message)
	assert.Equal(t, "no such command", apiErr.Detail)
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	body := `plain text error`
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	err := parseAPIError(resp)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 500, apiErr.StatusCode)
	assert.Equal(t, "plain text error", apiErr.Message)
}

func TestParseAPIError_EmptyBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader("")),
	}
	err := parseAPIError(resp)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 500, apiErr.StatusCode)
}

func TestAPIError_JSONRoundTrip(t *testing.T) {
	raw := `{"error":406,"message":"Not Acceptable","detail":"no such command or directory (remove)"}`
	var parsed struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	err := json.Unmarshal([]byte(raw), &parsed)
	require.NoError(t, err)
	assert.Equal(t, 406, parsed.Error)
	assert.Equal(t, "Not Acceptable", parsed.Message)
	assert.Equal(t, "no such command or directory (remove)", parsed.Detail)
}
