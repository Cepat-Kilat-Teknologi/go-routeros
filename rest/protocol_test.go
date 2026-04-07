// rest/protocol_test.go
package rest

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockErrorConn struct{}

func (m *mockErrorConn) Read([]byte) (int, error)         { return 0, nil }
func (m *mockErrorConn) Write([]byte) (int, error)        { return 0, nil }
func (m *mockErrorConn) Close() error                     { return errors.New("mocked close error") }
func (m *mockErrorConn) LocalAddr() net.Addr              { return nil }
func (m *mockErrorConn) RemoteAddr() net.Addr             { return nil }
func (m *mockErrorConn) SetDeadline(time.Time) error      { return nil }
func (m *mockErrorConn) SetReadDeadline(time.Time) error  { return nil }
func (m *mockErrorConn) SetWriteDeadline(time.Time) error { return nil }

func TestIsHostAvailableOnPort_NotAvailable(t *testing.T) {
	available := isHostAvailableOnPort("localhost", "9999")
	assert.False(t, available)
}

func TestIsHostAvailableOnPort_Available(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("Failed to create listener:", err)
	}
	defer listener.Close()

	_, port, _ := net.SplitHostPort(listener.Addr().String())
	available := isHostAvailableOnPort("localhost", port)
	assert.True(t, available)
}

func TestShouldRetryTLSErrorRequest_HTTP(t *testing.T) {
	err := errors.New("tls: handshake failure")
	assert.False(t, shouldRetryTLSErrorRequest(err, httpProtocol))
}

func TestShouldRetryTLSErrorRequest_HTTPS(t *testing.T) {
	err := errors.New("tls: handshake failure")
	assert.True(t, shouldRetryTLSErrorRequest(err, httpsProtocol))
}

func TestShouldRetryTLSErrorRequest_NonTLSError(t *testing.T) {
	err := errors.New("connection refused")
	assert.False(t, shouldRetryTLSErrorRequest(err, httpsProtocol))
}

func TestDetermineProtocolFromURL_HTTP(t *testing.T) {
	assert.Equal(t, httpProtocol, determineProtocolFromURL("http://example.com"))
}

func TestDetermineProtocolFromURL_HTTPS(t *testing.T) {
	assert.Equal(t, httpsProtocol, determineProtocolFromURL("https://example.com"))
}

func TestReplaceProtocol(t *testing.T) {
	result := replaceProtocol("http://example.com", httpProtocol, httpsProtocol)
	assert.Equal(t, "https://example.com", result)
}

func TestCloseConnection_Error(t *testing.T) {
	mockConn := &mockErrorConn{}
	closeConnection(mockConn)
}
