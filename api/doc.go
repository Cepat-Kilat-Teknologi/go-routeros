// Package api implements the MikroTik RouterOS API Protocol (TCP, port 8728/8729).
// It works with all RouterOS versions (v6 and v7).
//
// A Client maintains a single TCP connection and is not safe for concurrent use.
// Create separate clients for concurrent operations.
package api
