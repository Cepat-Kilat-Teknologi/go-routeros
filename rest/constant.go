package rest

const (
	// httpProtocol is the plain HTTP scheme.
	httpProtocol = "http"
	// httpsProtocol is the TLS-secured HTTPS scheme.
	httpsProtocol = "https"

	// MethodGet is the HTTP GET method.
	MethodGet = "GET"
	// MethodPost is the HTTP POST method.
	MethodPost = "POST"
	// MethodPut is the HTTP PUT method.
	MethodPut = "PUT"
	// MethodPatch is the HTTP PATCH method.
	MethodPatch = "PATCH"
	// MethodDelete is the HTTP DELETE method.
	MethodDelete = "DELETE"

	// tlsHandshakeFailure is the error substring indicating a TLS handshake failure.
	tlsHandshakeFailure = "tls: handshake failure"
)
