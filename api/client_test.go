package api

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockServer struct {
	conn   net.Conn
	reader *proto.Reader
	writer *proto.Writer
}

func newMockServer(t *testing.T) (*mockServer, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	return &mockServer{
		conn:   server,
		reader: proto.NewReader(server),
		writer: proto.NewWriter(server),
	}, client
}

func (m *mockServer) readSentence() (*proto.Sentence, error) {
	return m.reader.ReadSentence()
}

func (m *mockServer) writeSentence(words ...string) error {
	m.writer.BeginSentence()
	for _, w := range words {
		m.writer.WriteWord(w)
	}
	return m.writer.EndSentence()
}

func (m *mockServer) close() {
	m.conn.Close()
}

func (m *mockServer) handleLogin() {
	m.readSentence()
	m.writeSentence("!done")
}

func (m *mockServer) handleLoginLegacy(challenge string) {
	m.readSentence()
	m.writeSentence("!done", "=ret="+challenge)
	m.readSentence()
	m.writeSentence("!done")
}

func TestDial_PostV643(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go srv.handleLogin()

	client := newClientFromConn(clientConn, "admin", "")
	err := client.login("admin", "")
	require.NoError(t, err)
	client.Close()
}

func TestDial_PreV643(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go srv.handleLoginLegacy("abcdef0123456789")

	client := newClientFromConn(clientConn, "admin", "secret")
	err := client.login("admin", "secret")
	require.NoError(t, err)
	client.Close()
}

func TestDial_LoginFailure(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		srv.writeSentence("!trap", "=category=5", "=message=invalid user name or password")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "wrong")
	err := client.login("admin", "wrong")
	require.Error(t, err)

	de, ok := err.(*DeviceError)
	require.True(t, ok)
	assert.Equal(t, 5, de.Category)
	assert.Contains(t, de.Message, "invalid user")
	client.Close()
}

func TestClient_Close(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	client := newClientFromConn(clientConn, "", "")
	err := client.Close()
	assert.NoError(t, err)
}

func TestClient_Print(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!re", "=.id=*1", "=address=10.0.0.1/24", "=interface=ether1")
		srv.writeSentence("!re", "=.id=*2", "=address=192.168.1.1/24", "=interface=ether2")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Len(t, reply.Re, 2)
	assert.Equal(t, "10.0.0.1/24", reply.Re[0].Map["address"])
	assert.Equal(t, "192.168.1.1/24", reply.Re[1].Map["address"])
	client.Close()
}

func TestClient_Print_WithProplist(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/print", s.Word)
		assert.Equal(t, "address,interface", s.Map[".proplist"])

		srv.writeSentence("!re", "=address=10.0.0.1/24", "=interface=ether1")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address",
		WithProplist("address", "interface"),
	)
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	client.Close()
}

func TestClient_Print_WithQuery(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()
		srv.readSentence()
		srv.writeSentence("!re", "=.id=*1", "=type=ether")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/interface",
		WithQuery("?type=ether"),
	)
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	client.Close()
}

func TestClient_Add(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/add", s.Word)
		assert.Equal(t, "10.0.0.1/24", s.Map["address"])
		assert.Equal(t, "ether1", s.Map["interface"])

		srv.writeSentence("!done", "=ret=*A")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Add(context.Background(), "/ip/address", map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	require.NoError(t, err)
	assert.Equal(t, "*A", reply.Done.Map["ret"])
	client.Close()
}

func TestClient_Set(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/set", s.Word)
		assert.Equal(t, "*1", s.Map[".id"])
		assert.Equal(t, "updated", s.Map["comment"])

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Set(context.Background(), "/ip/address", map[string]string{
		".id":     "*1",
		"comment": "updated",
	})
	require.NoError(t, err)
	client.Close()
}

func TestClient_Remove(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/remove", s.Word)
		assert.Equal(t, "*1", s.Map[".id"])

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Remove(context.Background(), "/ip/address", "*1")
	require.NoError(t, err)
	client.Close()
}

func TestClient_Run(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/system/reboot", s.Word)

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Run(context.Background(), "/system/reboot", nil)
	require.NoError(t, err)
	client.Close()
}

func TestClient_TrapError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!trap", "=category=0", "=message=no such command")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Print(context.Background(), "/nonexistent")
	require.Error(t, err)
	de, ok := err.(*DeviceError)
	require.True(t, ok)
	assert.Equal(t, 0, de.Category)
	assert.Equal(t, "no such command", de.Message)
	client.Close()
}

func TestClient_FatalError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!fatal", "=message=session terminated")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Print(context.Background(), "/ip/address")
	require.Error(t, err)
	fe, ok := err.(*FatalError)
	require.True(t, ok)
	assert.Equal(t, "session terminated", fe.Message)
	client.Close()
}

func TestClient_Auth(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!re", "=board-name=hAP", "=version=7.14", "=platform=MikroTik")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Auth(context.Background())
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	assert.Equal(t, "MikroTik", reply.Re[0].Map["platform"])
	client.Close()
}

func TestClientOption_WithTimeout(t *testing.T) {
	cfg := &clientConfig{}
	WithTimeout(30 * time.Second)(cfg)
	assert.Equal(t, 30*time.Second, cfg.timeout)
}

func TestClientOption_WithTLS(t *testing.T) {
	cfg := &clientConfig{}
	WithTLS(true)(cfg)
	assert.True(t, cfg.useTLS)
}

func TestClientOption_WithTLSConfig(t *testing.T) {
	tc := &tls.Config{InsecureSkipVerify: true}
	cfg := &clientConfig{}
	WithTLSConfig(tc)(cfg)
	assert.Equal(t, tc, cfg.tlsConfig)
	assert.True(t, cfg.useTLS)
}

func TestResolveAddress_DefaultPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1", false)
	assert.Equal(t, "192.168.88.1:8728", addr)
}

func TestResolveAddress_DefaultTLSPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1", true)
	assert.Equal(t, "192.168.88.1:8729", addr)
}

func TestResolveAddress_CustomPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1:9000", false)
	assert.Equal(t, "192.168.88.1:9000", addr)
}

func TestClient_EmptyReply(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Empty(t, reply.Re)
	assert.NotNil(t, reply.Done)
	client.Close()
}
