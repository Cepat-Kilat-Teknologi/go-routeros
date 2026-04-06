package proto

import (
	"bufio"
	"io"
	"sync"
)

// Writer writes API sentences to a connection.
type Writer struct {
	w  *bufio.Writer
	mu sync.Mutex
}

// NewWriter creates a new Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// encodeLength encodes a word length as a 1-5 byte prefix.
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

// BeginSentence acquires the write lock.
func (w *Writer) BeginSentence() *Writer {
	w.mu.Lock()
	return w
}

// WriteWord writes a single length-prefixed word.
func (w *Writer) WriteWord(word string) *Writer {
	b := encodeLength(len(word))
	w.w.Write(b)
	w.w.WriteString(word)
	return w
}

// EndSentence writes the zero-length terminator, flushes, and releases the lock.
func (w *Writer) EndSentence() error {
	w.w.WriteByte(0x00)
	err := w.w.Flush()
	w.mu.Unlock()
	return err
}
