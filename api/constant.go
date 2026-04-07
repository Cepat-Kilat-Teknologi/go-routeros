package api

const (
	// DefaultPort is the default RouterOS API port for plain TCP connections.
	DefaultPort = "8728"
	// DefaultTLSPort is the default RouterOS API port for TLS-encrypted connections.
	DefaultTLSPort = "8729"

	// replyRe is the data reply word. Each !re sentence contains one record.
	replyRe = "!re"
	// replyDone is the completion reply word. Marks the end of a response.
	replyDone = "!done"
	// replyTrap is the error reply word. Contains error details (category + message).
	replyTrap = "!trap"
	// replyFatal is the fatal error reply word. The connection is closed after this.
	replyFatal = "!fatal"
	// replyEmpty is the empty reply word introduced in RouterOS 7.18+.
	// Sent when a command returns no data (e.g., empty print result).
	replyEmpty = "!empty"
)
