// Package api provides a client for the MikroTik RouterOS API Protocol.
//
// The API Protocol is a binary TCP-based protocol supported by all RouterOS
// versions (v6 and v7). It communicates over port 8728 (plain TCP) or
// port 8729 (TLS-encrypted).
//
// This package handles authentication automatically, supporting both the
// modern plaintext login (RouterOS 6.43+) and the legacy MD5 challenge-response
// login (RouterOS < 6.43).
//
// Usage:
//
//	client, err := api.Dial("192.168.88.1", "admin", "password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	reply, err := client.Print(ctx, "/ip/address",
//	    api.WithProplist("address", "interface"),
//	)
package api

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
)

// ClientOption configures the Client via functional options pattern.
// Pass one or more ClientOption values to Dial to customize the connection.
type ClientOption func(*clientConfig)

// clientConfig holds the configuration for establishing a connection.
type clientConfig struct {
	useTLS    bool          // whether to use TLS encryption
	tlsConfig *tls.Config   // custom TLS configuration (nil = use defaults)
	timeout   time.Duration // connection and login timeout
}

// Client holds a connection to a RouterOS device.
// A Client is NOT safe for concurrent use by multiple goroutines.
// Each Client maintains a single TCP connection; concurrent commands
// require separate Client instances.
type Client struct {
	conn   net.Conn      // underlying TCP or TLS connection
	reader *proto.Reader // reads API sentences from the connection
	writer *proto.Writer // writes API sentences to the connection
}

// WithTLS enables TLS-encrypted connection on port 8729 (by default).
// When enabled, the client connects using TLS with default configuration.
// For custom TLS settings (e.g., InsecureSkipVerify), use WithTLSConfig instead.
func WithTLS(useTLS bool) ClientOption {
	return func(c *clientConfig) {
		c.useTLS = useTLS
	}
}

// WithTLSConfig sets a custom TLS configuration for the connection.
// This implicitly enables TLS (no need to also call WithTLS).
// Use this for self-signed certificates:
//
//	api.WithTLSConfig(&tls.Config{InsecureSkipVerify: true})
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *clientConfig) {
		c.tlsConfig = config
		c.useTLS = true // TLS is always enabled when a custom config is provided
	}
}

// WithTimeout sets the connection and login timeout.
// This timeout applies to both the TCP dial and the login handshake.
// After successful login, the timeout is cleared.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// resolveAddress ensures the address includes a port number.
// If no port is specified, the default API port is appended:
//   - Plain TCP: 8728
//   - TLS: 8729
func resolveAddress(address string, useTLS bool) string {
	// net.SplitHostPort returns an error if no port is present
	if _, _, err := net.SplitHostPort(address); err != nil {
		if useTLS {
			return address + ":" + DefaultTLSPort
		}
		return address + ":" + DefaultPort
	}
	return address
}

// newClientFromConn creates a Client from an existing connection.
// This is used internally for unit testing with mock connections.
func newClientFromConn(conn net.Conn) *Client {
	return &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}
}

// Dial connects to a RouterOS device and authenticates.
// It establishes a TCP (or TLS) connection, performs the login handshake,
// and returns a ready-to-use Client.
//
// The address can be "host", "host:port", or "ip:port".
// If no port is specified, the default port is used (8728 for TCP, 8729 for TLS).
//
// Example:
//
//	// Plain TCP connection
//	client, err := api.Dial("192.168.88.1", "admin", "password")
//
//	// TLS connection with self-signed certificate
//	client, err := api.Dial("192.168.88.1", "admin", "password",
//	    api.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
//	)
func Dial(address, username, password string, opts ...ClientOption) (*Client, error) {
	// Apply all functional options to the configuration.
	cfg := &clientConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Append default port if not specified in the address.
	addr := resolveAddress(address, cfg.useTLS)

	var conn net.Conn
	var err error

	// Create a dialer with the configured timeout.
	dialer := net.Dialer{Timeout: cfg.timeout}

	// Establish the TCP or TLS connection.
	if cfg.useTLS {
		tlsCfg := cfg.tlsConfig
		if tlsCfg == nil {
			tlsCfg = &tls.Config{} // use default TLS config if none provided
		}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("routeros: dial %s: %w", addr, err)
	}

	// Initialize the client with protocol reader and writer.
	c := &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}

	// Apply login timeout to prevent hanging if the router doesn't respond.
	if cfg.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(cfg.timeout))
	}

	// Perform authentication. On failure, close the connection immediately.
	if err := c.login(username, password); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Clear the deadline after successful login so future commands aren't affected.
	_ = conn.SetDeadline(time.Time{})

	return c, nil
}

// Close closes the underlying TCP connection to the RouterOS device.
func (c *Client) Close() error {
	return c.conn.Close()
}

// login handles authentication, auto-detecting the login method.
//
// RouterOS 6.43+ uses plaintext login: the client sends username and password
// directly, and the router replies with !done on success.
//
// RouterOS < 6.43 uses MD5 challenge-response: the router replies with a
// challenge token in the "ret" field of !done, and the client must compute
// an MD5 hash and send it back. This is handled by loginLegacy.
func (c *Client) login(username, password string) error {
	// Send the login command with credentials.
	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=password=" + password)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	// Read the router's response.
	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	// !trap means authentication failed (bad credentials or permission denied).
	// Read and discard the following !done sentence before returning the error.
	if sen.Word == replyTrap {
		_, _ = c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	// !fatal means a critical error (e.g., too many sessions).
	// The connection is closed by the router after a fatal error.
	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	// If !done contains a "ret" field, the router is using the legacy
	// challenge-response login method (pre-6.43).
	if challenge, ok := sen.Get("ret"); ok {
		return c.loginLegacy(username, password, challenge)
	}

	// !done without "ret" means successful modern login (post-6.43).
	return nil
}

// loginLegacy handles the pre-6.43 MD5 challenge-response login.
//
// The router sends a hex-encoded challenge. The client computes:
//
//	MD5(0x00 + password + challenge_bytes)
//
// and sends the result as a hex string prefixed with "00".
func (c *Client) loginLegacy(username, password, challenge string) error {
	// Decode the hex challenge from the router.
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return fmt.Errorf("routeros: decode challenge: %w", err)
	}

	// Compute the MD5 response: MD5(0x00 + password + challenge).
	h := md5.New()
	h.Write([]byte{0})        // null byte prefix (required by the protocol)
	h.Write([]byte(password)) // user's password
	h.Write(challengeBytes)   // challenge from the router
	response := fmt.Sprintf("00%x", h.Sum(nil))

	// Send the second login command with the computed response.
	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=response=" + response)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	// Read the router's final response.
	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	// !trap means the MD5 response was incorrect.
	if sen.Word == replyTrap {
		_, _ = c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	// !fatal means a critical error occurred.
	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	// !done means successful legacy login.
	return nil
}

// sendCommand writes a complete command sentence to the router.
// The sentence includes the command word, key-value parameters,
// proplist filter, and query words.
func (c *Client) sendCommand(command string, params map[string]string, opts *requestOptions) error {
	c.writer.BeginSentence()
	c.writer.WriteWord(command)

	// Write each parameter as an API attribute word (=key=value).
	for k, v := range params {
		c.writer.WriteWord("=" + k + "=" + v)
	}

	// Write the proplist word to limit returned properties.
	if len(opts.proplist) > 0 {
		c.writer.WriteWord(proplistWord(opts.proplist))
	}

	// Write each query word for filtering.
	for _, q := range opts.query {
		c.writer.WriteWord(q)
	}

	// Flush the sentence to the network.
	return c.writer.EndSentence()
}

// readReply reads API sentences from the router until the response is complete.
// It collects !re (data) sentences into Reply.Re and the !done sentence into Reply.Done.
// If a !trap sentence is received, the error is deferred until !done is read.
// A !fatal sentence causes an immediate return with an error.
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
			// !re: a data sentence containing record fields.
			reply.Re = append(reply.Re, sen)
		case replyDone:
			// !done: marks the end of the response.
			// If a trap error was collected earlier, return it now.
			reply.Done = sen
			if trapErr != nil {
				return nil, trapErr
			}
			return reply, nil
		case replyTrap:
			// !trap: an error occurred, but more sentences may follow.
			// Defer the error until !done is received.
			trapErr = parseTrapError(sen)
		case replyFatal:
			// !fatal: a critical error. The connection is now closed.
			return nil, parseFatalError(sen)
		case replyEmpty:
			// !empty: RouterOS 7.18+ sends this when no data matches.
			// Ignored; continue reading until !done.
		}
	}
}

// execute sends a command to the router and reads the complete reply.
// It applies context deadlines to the underlying connection, ensuring
// that operations respect context cancellation and timeouts.
func (c *Client) execute(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	reqOpts := collectRequestOptions(opts...)

	// Apply the context deadline to the TCP connection.
	// This ensures that both the write and read operations will time out
	// if the context deadline is exceeded.
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
		defer func() { _ = c.conn.SetDeadline(time.Time{}) }()
	}

	// Send the command sentence to the router.
	if err := c.sendCommand(command, params, reqOpts); err != nil {
		return nil, fmt.Errorf("routeros: send: %w", err)
	}

	// Read and return the complete reply.
	return c.readReply()
}

// Auth verifies the connection by querying system resource info.
// Returns system information including board-name, version, platform, and uptime.
func (c *Client) Auth(ctx context.Context) (*Reply, error) {
	return c.execute(ctx, "/system/resource/print", nil)
}

// Print retrieves data from RouterOS by sending a /command/print request.
// Use WithProplist to limit returned properties and WithQuery for filtering.
//
// Example:
//
//	reply, err := client.Print(ctx, "/ip/address",
//	    api.WithProplist("address", "interface"),
//	)
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/print", nil, opts...)
}

// Add creates a new record in RouterOS by sending a /command/add request.
// The params map contains the fields for the new record.
// Returns the created record's ID in reply.Done.Get("ret").
//
// Example:
//
//	reply, err := client.Add(ctx, "/ip/address", map[string]string{
//	    "address": "10.0.0.1/24", "interface": "ether1",
//	})
//	id, _ := reply.Done.Get("ret")
func (c *Client) Add(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/add", params, opts...)
}

// Set updates an existing record in RouterOS by sending a /command/set request.
// The params map must include ".id" to identify the record to update.
//
// Example:
//
//	_, err := client.Set(ctx, "/ip/address", map[string]string{
//	    ".id": "*1", "comment": "Updated via API",
//	})
func (c *Client) Set(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/set", params, opts...)
}

// Remove deletes a record from RouterOS by sending a /command/remove request.
// The id parameter is the record's internal ID (e.g., "*1").
//
// Example:
//
//	_, err := client.Remove(ctx, "/ip/address", "*1")
func (c *Client) Remove(ctx context.Context, command string, id string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/remove", map[string]string{".id": id}, opts...)
}

// Run executes an arbitrary RouterOS command with optional parameters.
// Unlike Print/Add/Set/Remove, Run does not append a sub-command suffix.
//
// Example:
//
//	reply, err := client.Run(ctx, "/system/reboot", nil)
func (c *Client) Run(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command, params, opts...)
}

// parseTrapError creates a DeviceError from a !trap sentence.
// Extracts the "category" (0-7) and "message" fields from the sentence.
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
// Fatal errors indicate the connection has been closed by the router.
func parseFatalError(sen *proto.Sentence) *FatalError {
	msg, _ := sen.Get("message")
	return &FatalError{Message: msg}
}
