package api

const (
	DefaultPort    = "8728"
	DefaultTLSPort = "8729"

	replyRe    = "!re"
	replyDone  = "!done"
	replyTrap  = "!trap"
	replyFatal = "!fatal"
	replyEmpty = "!empty" // RouterOS 7.18+
)
