package api

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
)

// ClientOption configures the Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	useTLS    bool
	tlsConfig *tls.Config
	timeout   time.Duration
}

// Client holds a connection to a RouterOS device.
type Client struct {
	conn   io.ReadWriteCloser
	reader *proto.Reader
	writer *proto.Writer
}

// WithTLS enables TLS connection (port 8729 by default).
func WithTLS(useTLS bool) ClientOption {
	return func(c *clientConfig) {
		c.useTLS = useTLS
	}
}

// WithTLSConfig sets custom TLS configuration.
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *clientConfig) {
		c.tlsConfig = config
		c.useTLS = true
	}
}

// WithTimeout sets the connection timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// resolveAddress ensures address has a port. Uses default API port if missing.
func resolveAddress(address string, useTLS bool) string {
	if _, _, err := net.SplitHostPort(address); err != nil {
		if useTLS {
			return address + ":" + DefaultTLSPort
		}
		return address + ":" + DefaultPort
	}
	return address
}

// newClientFromConn creates a Client from an existing connection (for testing).
func newClientFromConn(conn io.ReadWriteCloser, username, password string) *Client {
	return &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}
}

// Dial connects to a RouterOS device and authenticates.
func Dial(address, username, password string, opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	addr := resolveAddress(address, cfg.useTLS)

	var conn net.Conn
	var err error

	dialer := net.Dialer{Timeout: cfg.timeout}

	if cfg.useTLS {
		tlsCfg := cfg.tlsConfig
		if tlsCfg == nil {
			tlsCfg = &tls.Config{}
		}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("routeros: dial %s: %w", addr, err)
	}

	c := &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}

	if err := c.login(username, password); err != nil {
		conn.Close()
		return nil, err
	}

	return c, nil
}

// Close closes the TCP connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// login handles authentication, auto-detecting pre/post-6.43.
func (c *Client) login(username, password string) error {
	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=password=" + password)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	if sen.Word == replyTrap {
		c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	if challenge, ok := sen.Get("ret"); ok {
		return c.loginLegacy(username, password, challenge)
	}

	return nil
}

// loginLegacy handles pre-6.43 MD5 challenge-response login.
func (c *Client) loginLegacy(username, password, challenge string) error {
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return fmt.Errorf("routeros: decode challenge: %w", err)
	}

	h := md5.New()
	h.Write([]byte{0})
	h.Write([]byte(password))
	h.Write(challengeBytes)
	response := fmt.Sprintf("00%x", h.Sum(nil))

	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=response=" + response)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	if sen.Word == replyTrap {
		c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	return nil
}

// sendCommand sends a command sentence with optional params and options.
func (c *Client) sendCommand(command string, params map[string]string, opts *requestOptions) error {
	c.writer.BeginSentence()
	c.writer.WriteWord(command)

	for k, v := range params {
		c.writer.WriteWord("=" + k + "=" + v)
	}

	if len(opts.proplist) > 0 {
		c.writer.WriteWord(proplistWord(opts.proplist))
	}

	for _, q := range opts.query {
		c.writer.WriteWord(q)
	}

	return c.writer.EndSentence()
}

// readReply reads sentences until !done or !fatal, collecting !re sentences.
func (c *Client) readReply() (*Reply, error) {
	reply := &Reply{}
	var trapErr error

	for {
		sen, err := c.reader.ReadSentence()
		if err != nil {
			return nil, err
		}

		switch sen.Word {
		case replyRe:
			reply.Re = append(reply.Re, sen)
		case replyDone:
			reply.Done = sen
			if trapErr != nil {
				return nil, trapErr
			}
			return reply, nil
		case replyTrap:
			trapErr = parseTrapError(sen)
		case replyFatal:
			return nil, parseFatalError(sen)
		case replyEmpty:
			// RouterOS 7.18+: no data to return, continue to !done
		}
	}
}

// execute sends a command and reads the reply.
func (c *Client) execute(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	reqOpts := collectRequestOptions(opts...)

	if err := c.sendCommand(command, params, reqOpts); err != nil {
		return nil, fmt.Errorf("routeros: send: %w", err)
	}

	return c.readReply()
}

// Auth verifies the connection by querying system resource info.
func (c *Client) Auth(ctx context.Context) (*Reply, error) {
	return c.execute(ctx, "/system/resource/print", nil)
}

// Print retrieves data (sends /command/print).
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/print", nil, opts...)
}

// Add creates a new record (sends /command/add).
func (c *Client) Add(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/add", params, opts...)
}

// Set updates a record (sends /command/set).
func (c *Client) Set(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/set", params, opts...)
}

// Remove deletes a record (sends /command/remove with =.id=).
func (c *Client) Remove(ctx context.Context, command string, id string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/remove", map[string]string{".id": id}, opts...)
}

// Run executes an arbitrary command with optional parameters.
func (c *Client) Run(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command, params, opts...)
}

// parseTrapError creates a DeviceError from a !trap sentence.
func parseTrapError(sen *proto.Sentence) *DeviceError {
	category := 0
	if catStr, ok := sen.Get("category"); ok {
		category, _ = strconv.Atoi(catStr)
	}
	msg, _ := sen.Get("message")
	return &DeviceError{
		Category: category,
		Message:  msg,
	}
}

// parseFatalError creates a FatalError from a !fatal sentence.
func parseFatalError(sen *proto.Sentence) *FatalError {
	msg, _ := sen.Get("message")
	return &FatalError{Message: msg}
}
