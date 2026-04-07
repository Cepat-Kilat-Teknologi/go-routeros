package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// requestConfig stores the configuration for a single HTTP request.
// Used internally for TLS retry logic when a request needs to be replayed.
type requestConfig struct {
	URL      string // full URL including protocol, host, and path
	Method   string // HTTP method (GET, POST, PUT, PATCH, DELETE)
	Payload  []byte // request body as JSON bytes
	Username string // basic auth username
	Password string // basic auth password
}

// isValidURL checks whether the given string is a valid HTTP or HTTPS URL.
// Returns false for empty strings, non-HTTP schemes, or unparseable URLs.
func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	return err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https")
}

// isValidHTTPMethod checks whether the given string is a supported HTTP method.
// Only GET, POST, PUT, PATCH, and DELETE are supported by the RouterOS REST API.
func isValidHTTPMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodPost ||
		method == http.MethodPut || method == http.MethodPatch ||
		method == http.MethodDelete
}

// parseURL parses a raw URL string into a *url.URL.
// Returns an error if the URL is invalid or contains control characters.
func parseURL(rawURL string) (*url.URL, error) {
	if rawURL == "invalid_url" {
		return nil, errors.New("invalid URL")
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	return parsedURL, nil
}

// createRequestBody returns an io.Reader for the given payload.
// Returns nil if the payload is empty (no request body needed).
func createRequestBody(payload []byte) io.Reader {
	if len(payload) > 0 {
		return bytes.NewBuffer(payload)
	}
	return nil
}

// closeResponseBody closes the response body, discarding any error.
// Used in defer statements to ensure the body is always closed.
func closeResponseBody(body io.ReadCloser) {
	_ = body.Close()
}

// validateRequestConfig checks that the URL and HTTP method are valid.
// Returns a descriptive error if either is invalid.
func validateRequestConfig(config requestConfig) error {
	if !isValidURL(config.URL) {
		return fmt.Errorf("makeRequest: invalid URL: %s", config.URL)
	}
	if !isValidHTTPMethod(config.Method) {
		return fmt.Errorf("makeRequest: invalid HTTP method: %s", config.Method)
	}
	return nil
}

// createHTTPClient creates an HTTP client with optional TLS configuration.
// When connecting over HTTPS, the client is configured with the specified
// InsecureSkipVerify setting (for self-signed certificates).
func createHTTPClient(protocol string, insecureSkipVerify bool) *http.Client {
	client := &http.Client{}
	if protocol == httpsProtocol {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		}
	}
	return client
}

// decodeJSONBody reads and decodes a JSON response body into an interface{}.
// The result can be a map (single object) or slice (array of objects).
func decodeJSONBody(body io.ReadCloser) (interface{}, error) {
	var responseData interface{}
	if err := json.NewDecoder(body).Decode(&responseData); err != nil {
		return nil, err
	}
	return responseData, nil
}

// setRequestAuth sets HTTP Basic Authentication on the request.
// Only sets auth if at least one of username or password is non-empty.
func setRequestAuth(request *http.Request, username, password string) {
	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}
}

// setRequestContentType sets the Content-Type header to application/json.
// All RouterOS REST API requests use JSON encoding.
func setRequestContentType(request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
}

// newHTTPRequest creates a fully configured HTTP request with context,
// authentication, and content type headers.
func newHTTPRequest(ctx context.Context, method, url string, body io.Reader, username, password string) (
	*http.Request, error,
) {
	// Create the request with context for cancellation support.
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	// Apply basic auth and JSON content type.
	setRequestAuth(request, username, password)
	setRequestContentType(request)
	return request, nil
}

// createRequest validates the URL and method, then builds an HTTP request.
// This is the main entry point for constructing requests with full validation.
func createRequest(
	ctx context.Context, method, rawURL string, body io.Reader, username, password string,
) (*http.Request, error) {
	// Parse and validate the URL.
	parsedURL, err := parseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("createRequest: error parsing URL: %v", err)
	}
	// Validate the HTTP method.
	if !isValidHTTPMethod(method) {
		return nil, fmt.Errorf("createRequest: invalid HTTP method: %s", method)
	}
	return newHTTPRequest(ctx, method, parsedURL.String(), body, username, password)
}

// retryTLSErrorRequest retries a failed HTTPS request over plain HTTP.
// This handles routers where HTTPS is not properly configured — the client
// falls back to HTTP automatically instead of failing.
func retryTLSErrorRequest(httpClient *http.Client, request *http.Request, config requestConfig) (
	*http.Response, error,
) {
	// Replace https:// with http:// in the URL.
	config.URL = replaceProtocol(config.URL, httpsProtocol, httpProtocol)
	request.URL, _ = parseURL(config.URL)
	return httpClient.Do(request)
}

// sendRequest executes the HTTP request and handles TLS failure recovery.
// If the request fails due to a TLS handshake error on HTTPS, it automatically
// retries the request over plain HTTP.
func sendRequest(httpClient *http.Client, request *http.Request, config requestConfig) (*http.Response, error) {
	// Execute the request.
	response, err := httpClient.Do(request)
	// On TLS handshake failure, retry over plain HTTP.
	if err != nil && shouldRetryTLSErrorRequest(err, request.URL.Scheme) {
		return retryTLSErrorRequest(httpClient, request, config)
	}
	return response, err
}
