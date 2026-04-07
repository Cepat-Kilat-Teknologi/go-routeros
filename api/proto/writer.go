package proto

import (
	"bufio"
	"io"
	"sync"
)

// Writer writes API sentences to a connection.
// It handles the variable-length encoding used by the RouterOS API Protocol.
// Writer is safe for concurrent use via its internal mutex.
type Writer struct {
	w  *bufio.Writer // buffered writer for efficient network writes
	mu sync.Mutex    // protects concurrent sentence writes
}

// NewWriter creates a new Writer wrapping the given io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// encodeLength encodes a word length as a 1-5 byte variable-length prefix.
// The encoding scheme matches readLength in reader.go:
//
//	0-127:         1 byte   (0xxxxxxx)
//	128-16383:     2 bytes  (10xxxxxx xxxxxxxx)
//	16384-2097151: 3 bytes  (110xxxxx xxxxxxxx xxxxxxxx)
//	2097152-268M:  4 bytes  (1110xxxx xxxxxxxx xxxxxxxx xxxxxxxx)
//	268M+:         5 bytes  (11110000 xxxxxxxx xxxxxxxx xxxxxxxx xxxxxxxx)
func encodeLength(l int) []byte {
	switch {
	case l < 0x80:
		return []byte{byte(l)}
	case l < 0x4000:
		return []byte{byte(l>>8) | 0x80, byte(l)}
	case l < 0x200000:
		return []byte{byte(l>>16) | 0xC0, byte(l >> 8), byte(l)}
	case l < 0x10000000:
		return []byte{byte(l>>24) | 0xE0, byte(l >> 16), byte(l >> 8), byte(l)}
	default:
		return []byte{0xF0, byte(l >> 24), byte(l >> 16), byte(l >> 8), byte(l)}
	}
}

// BeginSentence acquires the write lock and prepares for writing words.
// Must be paired with EndSentence to release the lock and flush the buffer.
// Returns the Writer for method chaining.
func (w *Writer) BeginSentence() *Writer {
	w.mu.Lock()
	return w
}

// WriteWord writes a single length-prefixed word to the buffer.
// The word is not flushed to the network until EndSentence is called.
// Returns the Writer for method chaining.
func (w *Writer) WriteWord(word string) *Writer {
	b := encodeLength(len(word)) // encode the word length as a variable-length prefix
	_, _ = w.w.Write(b)          // write the length prefix
	_, _ = w.w.WriteString(word) // write the word content
	return w
}

// EndSentence writes the zero-length sentence terminator, flushes all
// buffered data to the network, and releases the write lock.
// Returns any error from the flush operation.
func (w *Writer) EndSentence() error {
	_ = w.w.WriteByte(0x00) // write zero-length terminator
	err := w.w.Flush()      // flush all buffered data to the network
	w.mu.Unlock()           // release the write lock
	return err
}
