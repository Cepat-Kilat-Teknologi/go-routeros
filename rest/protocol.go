package rest

import (
	"net"
	"strings"
)

// closeConnection closes a network connection, discarding any error.
func closeConnection(conn net.Conn) {
	_ = conn.Close()
}

// determineProtocolFromURL extracts the protocol from a URL string.
// Returns "https" if the URL starts with "https", otherwise "http".
func determineProtocolFromURL(url string) string {
	if strings.HasPrefix(url, httpsProtocol) {
		return httpsProtocol
	}
	return httpProtocol
}

// replaceProtocol replaces the protocol in a URL with a new protocol.
func replaceProtocol(url, oldProtocol, newProtocol string) string {
	return strings.Replace(url, oldProtocol, newProtocol, 1)
}

// shouldRetryTlsErrorRequest checks if a request should be retried
// due to a TLS handshake failure on an HTTPS connection.
func shouldRetryTlsErrorRequest(err error, protocol string) bool {
	return strings.Contains(err.Error(), tlsHandshakeFailure) && protocol == httpsProtocol
}

// isHostAvailableOnPort checks if a host is available on a given port.
func isHostAvailableOnPort(host, port string) bool {
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		return false
	}
	defer closeConnection(conn)
	return true
}
