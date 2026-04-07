package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
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

	client := newClientFromConn(clientConn)
	err := client.login("admin", "")
	require.NoError(t, err)
	client.Close()
}

func TestDial_PreV643(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go srv.handleLoginLegacy("abcdef0123456789")

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

	client := newClientFromConn(clientConn)
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

func TestDial_RealTCPServer(t *testing.T) {
	// Start a local TCP listener that simulates RouterOS login
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := proto.NewReader(conn)
		w := proto.NewWriter(conn)
		// Read login sentence
		r.ReadSentence()
		// Reply with !done (post-6.43 login)
		w.BeginSentence()
		w.WriteWord("!done")
		w.EndSentence()
	}()

	client, err := Dial(ln.Addr().String(), "admin", "")
	require.NoError(t, err)
	client.Close()
}

func TestDial_RealTCPServer_WithTLS(t *testing.T) {
	cert := generateSelfSignedCert(t)

	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := proto.NewReader(conn)
		w := proto.NewWriter(conn)
		r.ReadSentence()
		w.BeginSentence()
		w.WriteWord("!done")
		w.EndSentence()
	}()

	clientTLSCfg := &tls.Config{InsecureSkipVerify: true}
	client, err := Dial(ln.Addr().String(), "admin", "", WithTLSConfig(clientTLSCfg))
	require.NoError(t, err)
	client.Close()
}

func TestDial_ConnectionRefused(t *testing.T) {
	// Try to dial a port that is not listening
	_, err := Dial("127.0.0.1:1", "admin", "", WithTimeout(100*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routeros: dial")
}

func TestDial_TLS_NoConfig(t *testing.T) {
	// Test Dial with useTLS=true but no tlsConfig (uses default empty tls.Config)
	// This will fail to connect, but exercises the code path
	_, err := Dial("127.0.0.1:1", "admin", "", WithTLS(true), WithTimeout(100*time.Millisecond))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "routeros: dial")
}

func TestLogin_FatalError(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		srv.writeSentence("!fatal", "=message=too many sessions")
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "")
	require.Error(t, err)
	_, ok := err.(*FatalError)
	assert.True(t, ok)
}

func TestLogin_WriteError(t *testing.T) {
	srv, clientConn := newMockServer(t)
	// Close the connection before login to cause a write error
	clientConn.Close()
	srv.close()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login write")
}

func TestLogin_ReadError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		// Read the login sentence, then close without replying
		srv.readSentence()
		srv.close()
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login read")
}

func TestLoginLegacy_InvalidHexChallenge(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		// Send a challenge with invalid hex characters
		srv.writeSentence("!done", "=ret=ZZZZ_not_hex")
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode challenge")
}

func TestLoginLegacy_FatalError(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		// First reply with challenge to trigger legacy login
		srv.writeSentence("!done", "=ret=abcdef0123456789")
		// Read the legacy login response
		srv.readSentence()
		// Reply with fatal
		srv.writeSentence("!fatal", "=message=session limit reached")
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "secret")
	require.Error(t, err)
	_, ok := err.(*FatalError)
	assert.True(t, ok)
}

func TestLoginLegacy_TrapError(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		srv.writeSentence("!done", "=ret=abcdef0123456789")
		srv.readSentence()
		srv.writeSentence("!trap", "=category=2", "=message=wrong password")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "secret")
	require.Error(t, err)
	de, ok := err.(*DeviceError)
	require.True(t, ok)
	assert.Equal(t, 2, de.Category)
}

func TestLoginLegacy_WriteError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		srv.readSentence()
		srv.writeSentence("!done", "=ret=abcdef0123456789")
		// Close the server side so the legacy login write fails
		time.Sleep(10 * time.Millisecond)
		srv.close()
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "secret")
	require.Error(t, err)
}

func TestLoginLegacy_ReadError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		srv.readSentence()
		srv.writeSentence("!done", "=ret=abcdef0123456789")
		srv.readSentence()
		// Close without replying to cause read error
		srv.close()
	}()

	client := newClientFromConn(clientConn)
	err := client.login("admin", "secret")
	require.Error(t, err)
}

func TestExecute_SendCommandError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		srv.handleLogin()
		srv.close()
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	// Wait for server to close
	time.Sleep(10 * time.Millisecond)

	_, err := client.Print(context.Background(), "/ip/address")
	require.Error(t, err)
}

func TestReadReply_ReadError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		srv.handleLogin()
		srv.readSentence()
		// Send one !re, then close mid-reply (before !done)
		srv.writeSentence("!re", "=.id=*1")
		srv.close()
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	_, err := client.Print(context.Background(), "/ip/address")
	require.Error(t, err)
}

func TestDial_LoginFailureCloseConn(t *testing.T) {
	// Test that Dial closes connection on login failure
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := proto.NewReader(conn)
		w := proto.NewWriter(conn)
		r.ReadSentence()
		w.BeginSentence()
		w.WriteWord("!trap")
		w.WriteWord("=category=5")
		w.WriteWord("=message=bad creds")
		w.EndSentence()
		w.BeginSentence()
		w.WriteWord("!done")
		w.EndSentence()
	}()

	_, err = Dial(ln.Addr().String(), "admin", "wrong")
	require.Error(t, err)
}

// generateSelfSignedCert creates a self-signed TLS certificate for testing.
func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func TestClient_EmptyReply(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Empty(t, reply.Re)
	assert.NotNil(t, reply.Done)
	client.Close()
}

func TestClient_EmptyReply_V718(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		// RouterOS 7.18+ sends !empty before !done when no data
		srv.writeSentence("!empty")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Empty(t, reply.Re)
	assert.NotNil(t, reply.Done)
	client.Close()
}

func TestDial_LoginTimeout(t *testing.T) {
	// Start a TCP server that accepts but never responds to login
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Read login sentence but never respond — simulate a hanging login
		r := proto.NewReader(conn)
		r.ReadSentence()
		time.Sleep(5 * time.Second)
	}()

	start := time.Now()
	_, err = Dial(ln.Addr().String(), "admin", "", WithTimeout(200*time.Millisecond))
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "login read")
	assert.Less(t, elapsed, 2*time.Second, "login should timeout quickly")
}

func TestClient_ContextCancel(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()
		// Don't respond — simulate a hanging request
		// The context cancel should unblock the client
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Print(ctx, "/ip/address")
	require.Error(t, err)
	client.Close()
}

func TestClient_ContextDeadline(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		// Respond normally
		srv.writeSentence("!re", "=.id=*1")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn)
	require.NoError(t, client.login("admin", ""))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := client.Print(ctx, "/ip/address")
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	client.Close()
}
