package api

import "fmt"

// DeviceError represents a !trap response from RouterOS.
type DeviceError struct {
	Category int
	Message  string
}

// Error implements the error interface.
func (e *DeviceError) Error() string {
	return fmt.Sprintf("routeros: trap: %s (category %d)", e.Message, e.Category)
}

// FatalError represents a !fatal response from RouterOS.
// The connection is closed by the router after a fatal error.
type FatalError struct {
	Message string
}

// Error implements the error interface.
func (e *FatalError) Error() string {
	return fmt.Sprintf("routeros: fatal: %s", e.Message)
}
