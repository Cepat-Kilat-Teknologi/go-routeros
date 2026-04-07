// Package rest provides a client for the MikroTik RouterOS v7 REST API.
//
// The REST API communicates over HTTP/HTTPS (ports 80/443) and is available
// on RouterOS v7 and later. It uses standard HTTP methods for CRUD operations:
//   - GET for reading data (Print)
//   - PUT for creating records (Add)
//   - PATCH for updating records (Set)
//   - DELETE for removing records (Remove)
//   - POST for executing commands (Run)
//
// Usage:
//
//	client := rest.NewClient("192.168.88.1", "admin", "password",
//	    rest.WithInsecureSkipVerify(true),
//	)
//	result, err := client.Print(ctx, "ip/address")
package rest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ClientOption configures the Client via functional options pattern.
// Pass one or more ClientOption values to NewClient to customize behavior.
type ClientOption func(*Client)

// Client holds connection configuration and credentials for a RouterOS device.
// A Client is safe for concurrent use by multiple goroutines because each
// request creates its own HTTP client and request objects.
type Client struct {
	host               string        // RouterOS device hostname or IP (with optional scheme)
	username           string        // authentication username
	password           string        // authentication password
	insecureSkipVerify bool          // skip TLS certificate verification (for self-signed certs)
	timeout            time.Duration // HTTP client timeout for each request
	httpClient         *http.Client  // custom HTTP client (overrides other settings if set)
}

// NewClient creates a new RouterOS REST API client.
// The host can be an IP address, hostname, or URL with scheme:
//   - "192.168.88.1" — uses HTTP by default
//   - "https://192.168.88.1" — uses HTTPS
//
// Example:
//
//	client := rest.NewClient("192.168.88.1", "admin", "password",
//	    rest.WithInsecureSkipVerify(true),
//	    rest.WithTimeout(30 * time.Second),
//	)
func NewClient(host, username, password string, opts ...ClientOption) *Client {
	c := &Client{
		host:     host,
		username: username,
		password: password,
	}
	// Apply all functional options.
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithInsecureSkipVerify skips TLS certificate verification.
// Use this when connecting to routers with self-signed certificates.
// WARNING: This disables certificate validation and should not be
// used in production without understanding the security implications.
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		c.insecureSkipVerify = skip
	}
}

// WithTimeout sets the HTTP client timeout for each request.
// This is the total time allowed for a single HTTP request/response cycle.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithHTTPClient sets a custom http.Client, overriding InsecureSkipVerify and Timeout.
// Use this for advanced HTTP configuration (custom transport, proxy, etc.).
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// cleanHost strips the protocol scheme (http:// or https://) from a host string.
// The scheme is determined separately by resolveProtocol.
func cleanHost(host string) string {
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	return host
}

// getHTTPClient returns the configured http.Client.
// If a custom client was provided via WithHTTPClient, it is returned as-is.
// Otherwise, a new client is created with the configured timeout and TLS settings.
func (c *Client) getHTTPClient(protocol string) *http.Client {
	// Return custom client if provided.
	if c.httpClient != nil {
		return c.httpClient
	}

	client := &http.Client{}
	// Apply timeout if configured.
	if c.timeout > 0 {
		client.Timeout = c.timeout
	}
	// Configure TLS transport for HTTPS connections.
	if protocol == httpsProtocol {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.insecureSkipVerify,
			},
		}
	}
	return client
}

// resolveProtocol determines the protocol (http or https) from the host string.
// If the host starts with "https://", HTTPS is used; otherwise HTTP.
func (c *Client) resolveProtocol() string {
	return determineProtocolFromURL(c.host)
}

// buildURL constructs the full REST API URL for a command.
// The URL format is: {protocol}://{host}/rest/{command}[?query_params]
//
// For example, "ip/address" becomes "http://192.168.88.1/rest/ip/address".
func (c *Client) buildURL(command string, opts *requestOptions) string {
	protocol := c.resolveProtocol()
	host := cleanHost(c.host)
	baseURL := fmt.Sprintf("%s://%s/rest/%s", protocol, host, command)

	// If no options, return the base URL without query parameters.
	if opts == nil {
		return baseURL
	}

	// Append query parameters (.proplist, filters) if present.
	q := buildURLQuery(opts)
	if encoded := q.Encode(); encoded != "" {
		return baseURL + "?" + encoded
	}
	return baseURL
}

// execute runs an HTTP request with the given method, command, payload, and options.
// This is the core method that all CRUD operations delegate to.
//
// The method determines how options are applied:
//   - GET/DELETE: options are sent as URL query parameters
//   - POST: options are merged into the JSON payload body
//   - PUT/PATCH: options are not applied (payload is sent as-is)
func (c *Client) execute(ctx context.Context, method, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	reqOpts := collectRequestOptions(opts...)

	var finalURL string
	var finalPayload []byte

	// Build the URL and payload based on the HTTP method.
	switch method {
	case MethodGet, MethodDelete:
		// For GET and DELETE, options go into URL query parameters.
		finalURL = c.buildURL(command, reqOpts)
		finalPayload = payload
	case MethodPost:
		// For POST, merge proplist and query into the JSON payload body.
		finalURL = c.buildURL(command, nil)
		merged, err := mergePayloadWithOptions(payload, reqOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to build request payload: %w", err)
		}
		finalPayload = merged
	default:
		// For PUT and PATCH, use the URL without options.
		finalURL = c.buildURL(command, nil)
		finalPayload = payload
	}

	// Create the HTTP client based on the detected protocol.
	protocol := c.resolveProtocol()
	httpClient := c.getHTTPClient(protocol)
	requestBody := createRequestBody(finalPayload)

	// Build the HTTP request with authentication and content type.
	request, err := createRequest(ctx, method, finalURL, requestBody, c.username, c.password)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	// Store request config for potential TLS retry.
	config := requestConfig{
		URL:      finalURL,
		Method:   method,
		Payload:  finalPayload,
		Username: c.username,
		Password: c.password,
	}

	// Execute the HTTP request. On TLS failure, retries over plain HTTP.
	response, err := sendRequest(httpClient, request, config)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response.Body)

	// Check for HTTP error status codes (4xx, 5xx).
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, parseAPIError(response)
	}

	// Handle 204 No Content (returned by RouterOS on successful DELETE).
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	// Decode the JSON response body.
	return decodeJSONBody(response.Body)
}

// Auth verifies the connection to the RouterOS device by querying system resources.
// Returns system information including board-name, version, platform, and uptime.
func (c *Client) Auth(ctx context.Context) (interface{}, error) {
	return c.execute(ctx, MethodGet, "system/resource", nil)
}

// Print retrieves data from RouterOS via GET request.
// Use WithProplist to limit returned properties and WithFilter for simple filtering.
//
// Example:
//
//	result, err := client.Print(ctx, "ip/address",
//	    rest.WithProplist("address", "interface"),
//	)
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodGet, command, nil, opts...)
}

// Add creates a new record in RouterOS via PUT request.
// The payload is a JSON-encoded map of field values.
// Returns the created record including its ".id" field.
//
// Example:
//
//	payload, _ := json.Marshal(map[string]string{
//	    "address": "10.0.0.1/24", "interface": "ether1",
//	})
//	result, err := client.Add(ctx, "ip/address", payload)
func (c *Client) Add(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPut, command, payload, opts...)
}

// Set updates an existing record in RouterOS via PATCH request.
// The command should include the record ID (e.g., "ip/address/*1").
//
// Example:
//
//	payload, _ := json.Marshal(map[string]string{"comment": "Updated"})
//	_, err := client.Set(ctx, "ip/address/*1", payload)
func (c *Client) Set(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPatch, command, payload, opts...)
}

// Remove deletes a record from RouterOS via DELETE request.
// The command should include the record ID (e.g., "ip/address/*1").
// Returns (nil, nil) on success because RouterOS responds with 204 No Content.
//
// Example:
//
//	_, err := client.Remove(ctx, "ip/address/*1")
func (c *Client) Remove(ctx context.Context, command string, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodDelete, command, nil, opts...)
}

// Run executes an arbitrary RouterOS command via POST request.
// Use WithProplist and WithQuery for complex filtering (they are merged into the payload).
//
// Example:
//
//	result, err := client.Run(ctx, "interface/print", nil,
//	    rest.WithProplist("name", "type"),
//	    rest.WithQuery("type=ether", "type=vlan", "#|"),
//	)
func (c *Client) Run(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPost, command, payload, opts...)
}

// Decode converts a REST API response (interface{}) into a typed struct or slice.
// This avoids the manual JSON re-encode/decode pattern when working with responses.
//
// The function marshals the source to JSON, then unmarshals into the destination.
// The destination must be a pointer to the target type.
//
// Example:
//
//	type IPAddress struct {
//	    ID        string `json:".id"`
//	    Address   string `json:"address"`
//	    Interface string `json:"interface"`
//	}
//
//	result, err := client.Print(ctx, "ip/address")
//	var addresses []IPAddress
//	err = rest.Decode(result, &addresses)
func Decode(src interface{}, dst interface{}) error {
	// Marshal the untyped response to JSON bytes.
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("rest: encode response: %w", err)
	}
	// Unmarshal the JSON bytes into the caller's typed destination.
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("rest: decode response: %w", err)
	}
	return nil
}
