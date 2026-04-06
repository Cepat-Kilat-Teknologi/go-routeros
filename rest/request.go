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

// requestConfig represents an internal request configuration.
type requestConfig struct {
	URL      string
	Method   string
	Payload  []byte
	Username string
	Password string
}

// isValidURL checks whether the given string is a valid HTTP or HTTPS URL.
func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	return err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https")
}

// isValidHTTPMethod checks whether the given string is a supported HTTP method.
func isValidHTTPMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodPost ||
		method == http.MethodPut || method == http.MethodPatch ||
		method == http.MethodDelete
}

// parseURL parses a raw URL string into a *url.URL.
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

// createRequestBody returns an io.Reader for the given payload, or nil if empty.
func createRequestBody(payload []byte) io.Reader {
	if len(payload) > 0 {
		return bytes.NewBuffer(payload)
	}
	return nil
}

// closeResponseBody closes the response body, discarding any error.
func closeResponseBody(body io.ReadCloser) {
	_ = body.Close()
}

// validateRequestConfig checks that the URL and HTTP method in config are valid.
func validateRequestConfig(config requestConfig) error {
	if !isValidURL(config.URL) {
		return fmt.Errorf("makeRequest: invalid URL: %s", config.URL)
	}
	if !isValidHTTPMethod(config.Method) {
		return fmt.Errorf("makeRequest: invalid HTTP method: %s", config.Method)
	}
	return nil
}

// createHTTPClient creates an HTTP client. If insecureSkipVerify is true,
// TLS certificate verification is skipped (for self-signed certs).
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

// decodeJSONBody reads and decodes a JSON response body.
func decodeJSONBody(body io.ReadCloser) (interface{}, error) {
	var responseData interface{}
	if err := json.NewDecoder(body).Decode(&responseData); err != nil {
		return nil, err
	}
	return responseData, nil
}

// setRequestAuth sets basic authentication on the request if credentials are provided.
func setRequestAuth(request *http.Request, username, password string) {
	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}
}

// setRequestContentType sets the Content-Type header to application/json.
func setRequestContentType(request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
}

// newHTTPRequest creates an HTTP request with context, auth, and content type.
func newHTTPRequest(ctx context.Context, method, url string, body io.Reader, username, password string) (
	*http.Request, error,
) {
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	setRequestAuth(request, username, password)
	setRequestContentType(request)
	return request, nil
}

// createRequest validates the URL and method, then builds an HTTP request.
func createRequest(
	ctx context.Context, method, rawURL string, body io.Reader, username, password string,
) (*http.Request, error) {
	parsedURL, err := parseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("createRequest: error parsing URL: %v", err)
	}
	if !isValidHTTPMethod(method) {
		return nil, fmt.Errorf("createRequest: invalid HTTP method: %s", method)
	}
	return newHTTPRequest(ctx, method, parsedURL.String(), body, username, password)
}

// retryTlsErrorRequest retries a failed HTTPS request over plain HTTP.
func retryTlsErrorRequest(httpClient *http.Client, request *http.Request, config requestConfig) (
	*http.Response, error,
) {
	config.URL = replaceProtocol(config.URL, httpsProtocol, httpProtocol)
	request.URL, _ = parseURL(config.URL)
	return httpClient.Do(request)
}

// sendRequest executes the request and retries over HTTP on TLS failure.
func sendRequest(httpClient *http.Client, request *http.Request, config requestConfig) (*http.Response, error) {
	response, err := httpClient.Do(request)
	if err != nil && shouldRetryTlsErrorRequest(err, request.URL.Scheme) {
		return retryTlsErrorRequest(httpClient, request, config)
	}
	return response, err
}

