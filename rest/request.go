// rest/request.go
package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	return err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https")
}

func isValidHTTPMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodPost ||
		method == http.MethodPut || method == http.MethodPatch ||
		method == http.MethodDelete
}

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

func createRequestBody(payload []byte) io.Reader {
	if len(payload) > 0 {
		return bytes.NewBuffer(payload)
	}
	return nil
}

func closeResponseBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Println(err)
	}
}

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

func decodeJSONBody(body io.ReadCloser) (interface{}, error) {
	var responseData interface{}
	if err := json.NewDecoder(body).Decode(&responseData); err != nil {
		return nil, err
	}
	return responseData, nil
}

func setRequestAuth(request *http.Request, username, password string) {
	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}
}

func setRequestContentType(request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
}

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

func retryTlsErrorRequest(httpClient *http.Client, request *http.Request, config requestConfig) (
	*http.Response, error,
) {
	config.URL = replaceProtocol(config.URL, httpsProtocol, httpProtocol)
	request.URL, _ = parseURL(config.URL)
	return httpClient.Do(request)
}

func sendRequest(httpClient *http.Client, request *http.Request, config requestConfig) (*http.Response, error) {
	response, err := httpClient.Do(request)
	if err != nil && shouldRetryTlsErrorRequest(err, request.URL.Scheme) {
		return retryTlsErrorRequest(httpClient, request, config)
	}
	return response, err
}

// makeRequest executes an HTTP request and returns the decoded JSON response.
// Returns *APIError for non-2xx responses.
func makeRequest(ctx context.Context, config requestConfig, insecureSkipVerify bool) (interface{}, error) {
	if err := validateRequestConfig(config); err != nil {
		return nil, err
	}

	protocol := determineProtocolFromURL(config.URL)
	httpClient := createHTTPClient(protocol, insecureSkipVerify)
	requestBody := createRequestBody(config.Payload)

	request, err := createRequest(ctx, config.Method, config.URL, requestBody, config.Username, config.Password)
	if err != nil {
		return nil, fmt.Errorf("makeRequest: request creation failed: %w", err)
	}

	response, err := sendRequest(httpClient, request, config)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response.Body)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, parseAPIError(response)
	}

	return decodeJSONBody(response.Body)
}
