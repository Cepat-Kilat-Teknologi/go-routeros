package api

import "fmt"

// DeviceError represents a !trap response from RouterOS.
// Trap errors occur when a command fails (e.g., bad parameters,
// missing record, invalid command). The connection remains open.
//
// Category codes:
//   - 0: Missing item or command
//   - 1: Argument value failure
//   - 2: Command interrupted
//   - 3: Scripting failure
//   - 4: General failure
//   - 5: API-related failure
//   - 6: TTY-related failure
//   - 7: Value from :return
type DeviceError struct {
	Category int    // trap category code (0-7)
	Message  string // human-readable error message from the router
}

// Error implements the error interface.
// Returns a formatted string with the message and category code.
func (e *DeviceError) Error() string {
	return fmt.Sprintf("routeros: trap: %s (category %d)", e.Message, e.Category)
}

// FatalError represents a !fatal response from RouterOS.
// Fatal errors are unrecoverable; the router closes the connection
// immediately after sending a !fatal reply. The client must
// reconnect by calling Dial again.
type FatalError struct {
	Message string // human-readable error message from the router
}

// Error implements the error interface.
func (e *FatalError) Error() string {
	return fmt.Sprintf("routeros: fatal: %s", e.Message)
}
