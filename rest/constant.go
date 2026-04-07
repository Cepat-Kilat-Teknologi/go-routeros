package rest

const (
	// httpProtocol is the plain HTTP scheme used for unencrypted connections (port 80).
	httpProtocol = "http"
	// httpsProtocol is the TLS-secured HTTPS scheme used for encrypted connections (port 443).
	httpsProtocol = "https"

	// MethodGet is the HTTP GET method, used for retrieving data (Print).
	MethodGet = "GET"
	// MethodPost is the HTTP POST method, used for executing commands (Run).
	MethodPost = "POST"
	// MethodPut is the HTTP PUT method, used for creating records (Add).
	MethodPut = "PUT"
	// MethodPatch is the HTTP PATCH method, used for updating records (Set).
	MethodPatch = "PATCH"
	// MethodDelete is the HTTP DELETE method, used for deleting records (Remove).
	MethodDelete = "DELETE"

	// tlsHandshakeFailure is the error substring that indicates a TLS handshake failure.
	// Used to detect when a connection should be retried over plain HTTP.
	tlsHandshakeFailure = "tls: handshake failure"
)
