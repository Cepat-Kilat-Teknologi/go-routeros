// Package rest provides a client for the MikroTik RouterOS v7 REST API.
package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ClientOption configures the Client.
type ClientOption func(*Client)

// Client holds connection configuration and credentials for a RouterOS device.
// A Client is safe for concurrent use by multiple goroutines.
type Client struct {
	host               string
	username           string
	password           string
	insecureSkipVerify bool
	timeout            time.Duration
	httpClient         *http.Client
}

// NewClient creates a new RouterOS REST API client.
func NewClient(host, username, password string, opts ...ClientOption) *Client {
	c := &Client{
		host:     host,
		username: username,
		password: password,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithInsecureSkipVerify skips TLS certificate verification.
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		c.insecureSkipVerify = skip
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithHTTPClient sets a custom http.Client, overriding InsecureSkipVerify and Timeout.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// cleanHost strips the protocol scheme from a host string.
func cleanHost(host string) string {
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	return host
}

// getHTTPClient returns the configured http.Client.
func (c *Client) getHTTPClient(protocol string) *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}

	client := &http.Client{}
	if c.timeout > 0 {
		client.Timeout = c.timeout
	}
	if protocol == httpsProtocol {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.insecureSkipVerify,
			},
		}
	}
	return client
}

// resolveProtocol determines the protocol from the host string.
func (c *Client) resolveProtocol() string {
	return determineProtocolFromURL(c.host)
}

// buildURL constructs the full REST API URL for a command.
func (c *Client) buildURL(command string, opts *requestOptions) string {
	protocol := c.resolveProtocol()
	host := cleanHost(c.host)
	baseURL := fmt.Sprintf("%s://%s/rest/%s", protocol, host, command)

	if opts == nil {
		return baseURL
	}

	q := buildURLQuery(opts)
	if encoded := q.Encode(); encoded != "" {
		return baseURL + "?" + encoded
	}
	return baseURL
}

// execute runs a request with the given method, command, payload, and options.
func (c *Client) execute(ctx context.Context, method, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	reqOpts := collectRequestOptions(opts...)

	var finalURL string
	var finalPayload []byte

	switch method {
	case MethodGet, MethodDelete:
		finalURL = c.buildURL(command, reqOpts)
		finalPayload = payload
	case MethodPost:
		finalURL = c.buildURL(command, nil)
		merged, err := mergePayloadWithOptions(payload, reqOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to build request payload: %w", err)
		}
		finalPayload = merged
	default:
		finalURL = c.buildURL(command, nil)
		finalPayload = payload
	}

	protocol := c.resolveProtocol()
	httpClient := c.getHTTPClient(protocol)
	requestBody := createRequestBody(finalPayload)

	request, err := createRequest(ctx, method, finalURL, requestBody, c.username, c.password)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	config := requestConfig{
		URL:      finalURL,
		Method:   method,
		Payload:  finalPayload,
		Username: c.username,
		Password: c.password,
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

// Auth verifies the connection to the RouterOS device.
func (c *Client) Auth(ctx context.Context) (interface{}, error) {
	return c.execute(ctx, MethodGet, "system/resource", nil)
}

// Print retrieves data from RouterOS (GET request).
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodGet, command, nil, opts...)
}

// Add creates a new record in RouterOS (PUT request).
func (c *Client) Add(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPut, command, payload, opts...)
}

// Set updates an existing record in RouterOS (PATCH request).
func (c *Client) Set(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPatch, command, payload, opts...)
}

// Remove deletes a record from RouterOS (DELETE request).
func (c *Client) Remove(ctx context.Context, command string, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodDelete, command, nil, opts...)
}

// Run executes an arbitrary command via POST request.
func (c *Client) Run(ctx context.Context, command string, payload []byte, opts ...RequestOption) (interface{}, error) {
	return c.execute(ctx, MethodPost, command, payload, opts...)
}
