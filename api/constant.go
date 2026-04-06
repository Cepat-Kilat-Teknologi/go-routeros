package api

const (
	// DefaultPort is the default RouterOS API port (plain TCP).
	DefaultPort = "8728"
	// DefaultTLSPort is the default RouterOS API port (TLS).
	DefaultTLSPort = "8729"

	// replyRe is the data reply word.
	replyRe = "!re"
	// replyDone is the completion reply word.
	replyDone = "!done"
	// replyTrap is the error reply word.
	replyTrap = "!trap"
	// replyFatal is the fatal error reply word.
	replyFatal = "!fatal"
	// replyEmpty is the empty reply word introduced in RouterOS 7.18+.
	replyEmpty = "!empty"
)
