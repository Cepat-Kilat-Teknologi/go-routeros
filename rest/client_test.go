// rest/client_test.go
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "")
	assert.Equal(t, "192.168.88.1", c.host)
	assert.Equal(t, "admin", c.username)
	assert.Equal(t, "", c.password)
	assert.False(t, c.insecureSkipVerify)
	assert.Nil(t, c.httpClient)
}

func TestNewClient_WithInsecureSkipVerify(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "", WithInsecureSkipVerify(true))
	assert.True(t, c.insecureSkipVerify)
}

func TestNewClient_WithTimeout(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "", WithTimeout(30*time.Second))
	assert.Equal(t, 30*time.Second, c.timeout)
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 10 * time.Second}
	c := NewClient("192.168.88.1", "admin", "", WithHTTPClient(custom))
	assert.Equal(t, custom, c.httpClient)
}

func TestClient_buildURL(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "")
	url := c.buildURL("ip/address", nil)
	assert.Equal(t, "http://192.168.88.1/rest/ip/address", url)
}

func TestClient_buildURL_WithHTTPS(t *testing.T) {
	c := NewClient("https://192.168.88.1", "admin", "")
	url := c.buildURL("ip/address", nil)
	assert.Equal(t, "https://192.168.88.1/rest/ip/address", url)
}

func TestClient_buildURL_WithFilter(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "")
	opts := &requestOptions{
		proplist: []string{"address", "interface"},
		filter:   map[string]string{"dynamic": "true"},
	}
	url := c.buildURL("ip/address", opts)
	assert.Contains(t, url, ".proplist=address%2Cinterface")
	assert.Contains(t, url, "dynamic=true")
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func jsonHandler(statusCode int, body interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(body)
	}
}

func TestClient_Auth(t *testing.T) {
	server := newTestServer(t, jsonHandler(200, map[string]string{
		"board-name": "hAP ac2",
		"platform":   "MikroTik",
		"version":    "7.14.3",
	}))
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	result, err := c.Auth(context.Background())

	require.NoError(t, err)
	data := result.(map[string]interface{})
	assert.Equal(t, "MikroTik", data["platform"])
}

func TestClient_Print(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/rest/ip/address")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"address": "10.0.0.1/24", ".id": "*1"},
		})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	result, err := c.Print(context.Background(), "ip/address")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Print_WithProplist(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "address,interface", r.URL.Query().Get(".proplist"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	_, err := c.Print(context.Background(), "ip/address",
		WithProplist("address", "interface"),
	)
	require.NoError(t, err)
}

func TestClient_Print_WithFilter(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("dynamic"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	_, err := c.Print(context.Background(), "ip/address",
		WithFilter(map[string]string{"dynamic": "true"}),
	)
	require.NoError(t, err)
}

func TestClient_Add(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{".id": "*1"})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	payload, _ := json.Marshal(map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	result, err := c.Add(context.Background(), "ip/address", payload)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Set(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{".id": "*1"})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	payload, _ := json.Marshal(map[string]string{"comment": "updated"})
	result, err := c.Set(context.Background(), "ip/address/*1", payload)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Remove(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	_, err := c.Remove(context.Background(), "ip/address/*1")
	require.NoError(t, err)
}

func TestClient_Remove_NoContent(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	result, err := c.Remove(context.Background(), "ip/address/*1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestClient_Run(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{{"name": "ether1"}})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	payload, _ := json.Marshal(map[string]string{".proplist": "name"})
	result, err := c.Run(context.Background(), "interface/print", payload)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Run_WithProplistAndQuery(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		assert.NotNil(t, body[".proplist"])
		assert.NotNil(t, body[".query"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	_, err := c.Run(context.Background(), "interface/print", nil,
		WithProplist("name", "type"),
		WithQuery("type=ether", "type=vlan", "#|"),
	)
	require.NoError(t, err)
}

func TestClient_ErrorResponse(t *testing.T) {
	server := newTestServer(t, jsonHandler(404, map[string]interface{}{
		"error":   404,
		"message": "Not Found",
		"detail":  "no such command",
	}))
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	_, err := c.Print(context.Background(), "nonexistent/path")

	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "no such command", apiErr.Detail)
}

func TestClient_BasicAuth(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", user)
		assert.Equal(t, "secret", pass)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "secret")
	_, err := c.Auth(context.Background())
	require.NoError(t, err)
}

func TestClient_getHTTPClient_CustomClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := NewClient("192.168.88.1", "admin", "", WithHTTPClient(custom))
	assert.Equal(t, custom, c.getHTTPClient("https"))
}

func TestClient_getHTTPClient_InsecureSkipVerify(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "", WithInsecureSkipVerify(true))
	httpClient := c.getHTTPClient("https")
	transport := httpClient.Transport.(*http.Transport)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestClient_getHTTPClient_WithTimeout(t *testing.T) {
	c := NewClient("192.168.88.1", "admin", "", WithTimeout(15*time.Second))
	httpClient := c.getHTTPClient("http")
	assert.Equal(t, 15*time.Second, httpClient.Timeout)
}

func TestClient_Execute_PostWithInvalidJSONPayload(t *testing.T) {
	server := newTestServer(t, jsonHandler(200, map[string]string{"ok": "true"}))
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	// Run uses POST; pass invalid JSON payload with a proplist option to trigger mergePayloadWithOptions error
	_, err := c.Run(context.Background(), "test/command", []byte("not-valid-json"),
		WithProplist("name"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build request payload")
}

func TestClient_Execute_PutMethod(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	payload, _ := json.Marshal(map[string]string{"key": "value"})
	result, err := c.Add(context.Background(), "test/command", payload)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Execute_PatchMethod(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	defer server.Close()

	c := NewClient(server.URL, "admin", "")
	payload, _ := json.Marshal(map[string]string{"key": "value"})
	result, err := c.Set(context.Background(), "test/command", payload)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_Execute_CreateRequestError(t *testing.T) {
	// Use an invalid HTTP method to trigger createRequest error path
	c := NewClient("http://127.0.0.1", "admin", "")
	_, err := c.execute(context.Background(), "INVALID METHOD", "test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request creation failed")
}

func TestClient_Execute_SendRequestError(t *testing.T) {
	// Point to a server that is not running to trigger sendRequest error
	c := NewClient("http://127.0.0.1:1", "admin", "")
	_, err := c.execute(context.Background(), MethodGet, "test", nil)
	require.Error(t, err)
}

func TestDecode_Struct(t *testing.T) {
	src := map[string]interface{}{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	}
	var dst struct {
		Address   string `json:"address"`
		Interface string `json:"interface"`
	}
	err := Decode(src, &dst)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1/24", dst.Address)
	assert.Equal(t, "ether1", dst.Interface)
}

func TestDecode_Slice(t *testing.T) {
	src := []interface{}{
		map[string]interface{}{"name": "ether1"},
		map[string]interface{}{"name": "ether2"},
	}
	var dst []struct {
		Name string `json:"name"`
	}
	err := Decode(src, &dst)
	require.NoError(t, err)
	assert.Len(t, dst, 2)
	assert.Equal(t, "ether1", dst[0].Name)
}

func TestDecode_Nil(t *testing.T) {
	var dst struct{}
	err := Decode(nil, &dst)
	require.NoError(t, err)
}

func TestDecode_InvalidDst(t *testing.T) {
	src := map[string]interface{}{"key": "value"}
	// Decode into a non-pointer to trigger unmarshal error
	var dst int
	err := Decode(src, &dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rest: decode response")
}

func TestDecode_MarshalError(t *testing.T) {
	// channels cannot be marshaled to JSON
	src := make(chan int)
	var dst struct{}
	err := Decode(src, &dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rest: encode response")
}

func TestClient_cleanHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://192.168.88.1", "192.168.88.1"},
		{"http://192.168.88.1", "192.168.88.1"},
		{"192.168.88.1", "192.168.88.1"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, cleanHost(tt.input))
		})
	}
}
