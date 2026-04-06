// rest/request_test.go
package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Valid HTTP URL", "http://example.com", true},
		{"Valid HTTPS URL", "https://example.com", true},
		{"Invalid URL", "invalid_url", false},
		{"Empty URL", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidURL(tt.url))
		})
	}
}

func TestIsValidHTTPMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"Valid GET", "GET", true},
		{"Valid POST", "POST", true},
		{"Valid PUT", "PUT", true},
		{"Valid PATCH", "PATCH", true},
		{"Valid DELETE", "DELETE", true},
		{"Invalid method", "INVALID", false},
		{"Empty method", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidHTTPMethod(tt.method))
		})
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{"Valid URL", "https://example.com/path", false},
		{"Invalid URL", "invalid_url", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseURL(tt.rawURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateRequestBody(t *testing.T) {
	body := createRequestBody([]byte(`{"key":"value"}`))
	assert.NotNil(t, body)

	emptyBody := createRequestBody(nil)
	assert.Nil(t, emptyBody)
}

func TestValidateRequestConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    requestConfig
		expectErr bool
	}{
		{"Valid", requestConfig{URL: "https://example.com", Method: "GET"}, false},
		{"Invalid URL", requestConfig{URL: "invalid_url", Method: "GET"}, true},
		{"Invalid Method", requestConfig{URL: "https://example.com", Method: "INVALID"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequestConfig(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateHTTPClient_HTTPS_Secure(t *testing.T) {
	client := createHTTPClient(httpsProtocol, false)
	transport := client.Transport.(*http.Transport)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestCreateHTTPClient_HTTPS_Insecure(t *testing.T) {
	client := createHTTPClient(httpsProtocol, true)
	transport := client.Transport.(*http.Transport)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestCreateHTTPClient_HTTP(t *testing.T) {
	client := createHTTPClient(httpProtocol, false)
	assert.Nil(t, client.Transport)
}

type mockErrorReadCloser struct{}

func (m *mockErrorReadCloser) Read(_ []byte) (int, error) {
	return 0, errors.New("mocked read error")
}
func (m *mockErrorReadCloser) Close() error {
	return errors.New("mocked close error")
}

func TestCloseResponseBody_Error(t *testing.T) {
	closeResponseBody(&mockErrorReadCloser{})
}

func TestDecodeJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"key":"value"}`)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	result, err := decodeJSONBody(resp.Body)
	require.NoError(t, err)
	data := result.(map[string]interface{})
	assert.Equal(t, "value", data["key"])
}

func TestSetRequestAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	setRequestAuth(req, "admin", "pass")
	user, pass, ok := req.BasicAuth()
	assert.True(t, ok)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pass", pass)
}

func TestSetRequestAuth_Empty(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	setRequestAuth(req, "", "")
	_, _, ok := req.BasicAuth()
	assert.False(t, ok)
}

func TestCreateRequest_InvalidMethod(t *testing.T) {
	_, err := createRequest(context.Background(), "INVALID", "http://example.com", nil, "", "")
	assert.Error(t, err)
}

func mustParseURL(rawURL string) *url.URL {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return parsedURL
}
