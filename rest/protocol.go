package rest

import (
	"net"
	"strings"
)

// closeConnection closes a network connection, discarding any error.
// Used in defer statements where the error is not actionable.
func closeConnection(conn net.Conn) {
	_ = conn.Close()
}

// determineProtocolFromURL extracts the protocol scheme from a URL string.
// Returns "https" if the URL starts with "https", otherwise returns "http".
// This is used to determine whether TLS should be used for the connection.
func determineProtocolFromURL(url string) string {
	if strings.HasPrefix(url, httpsProtocol) {
		return httpsProtocol
	}
	return httpProtocol
}

// replaceProtocol replaces the protocol scheme in a URL.
// Used by the TLS retry logic to convert "https://..." to "http://...".
func replaceProtocol(url, oldProtocol, newProtocol string) string {
	return strings.Replace(url, oldProtocol, newProtocol, 1)
}

// shouldRetryTLSErrorRequest checks if a failed request should be retried
// over plain HTTP due to a TLS handshake failure.
// Only retries if the error is a TLS handshake failure AND the original
// request was made over HTTPS. This prevents infinite retry loops.
func shouldRetryTLSErrorRequest(err error, protocol string) bool {
	return strings.Contains(err.Error(), tlsHandshakeFailure) && protocol == httpsProtocol
}

// isHostAvailableOnPort checks if a host is reachable on a given TCP port.
// Used to verify service availability before making API requests.
func isHostAvailableOnPort(host, port string) bool {
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		return false
	}
	defer closeConnection(conn)
	return true
}
