package proto

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Reader reads API sentences from a connection.
// It handles the variable-length encoding used by the RouterOS API Protocol.
type Reader struct {
	r *bufio.Reader // buffered reader for efficient byte-level reading
}

// NewReader creates a new Reader wrapping the given io.Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(r)}
}

// readLength decodes a variable-length prefix (1-5 bytes) from the stream.
//
// The RouterOS API Protocol encodes word lengths using a variable-length scheme:
//
//	0xxxxxxx                       → 1 byte,  length 0-127
//	10xxxxxx xxxxxxxx              → 2 bytes, length 128-16383
//	110xxxxx xxxxxxxx xxxxxxxx     → 3 bytes, length 16384-2097151
//	1110xxxx xxxxxxxx xxxxxxxx xx  → 4 bytes, length 2097152-268435455
//	11110000 xxxxxxxx xxxxxxxx xx  → 5 bytes, length 268435456+
//
// A length of 0 indicates the end of a sentence (sentence terminator).
func (r *Reader) readLength() (int, error) {
	// Read the first byte to determine the encoding length.
	b, err := r.r.ReadByte()
	if err != nil {
		return 0, err
	}

	switch {
	case b&0x80 == 0x00:
		// 1-byte encoding: 0xxxxxxx (0-127)
		return int(b), nil
	case b&0xC0 == 0x80:
		// 2-byte encoding: 10xxxxxx (128-16383)
		b2, err := r.r.ReadByte()
		if err != nil {
			return 0, err
		}
		return int(b&^0xC0)<<8 | int(b2), nil
	case b&0xE0 == 0xC0:
		// 3-byte encoding: 110xxxxx (16384-2097151)
		buf := make([]byte, 2)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xE0)<<16 | int(buf[0])<<8 | int(buf[1]), nil
	case b&0xF0 == 0xE0:
		// 4-byte encoding: 1110xxxx (2097152-268435455)
		buf := make([]byte, 3)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xF0)<<24 | int(buf[0])<<16 | int(buf[1])<<8 | int(buf[2]), nil
	case b&0xF8 == 0xF0:
		// 5-byte encoding: 11110000 (268435456+)
		buf := make([]byte, 4)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3]), nil
	default:
		return 0, fmt.Errorf("unexpected length byte: 0x%02x", b)
	}
}

// readWord reads a single length-prefixed word from the stream.
// Returns an empty string if the word length is zero (sentence terminator).
func (r *Reader) readWord() (string, error) {
	l, err := r.readLength()
	if err != nil {
		return "", err
	}
	// A zero-length word marks the end of a sentence.
	if l == 0 {
		return "", nil
	}
	// Read exactly l bytes for the word content.
	buf := make([]byte, l)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadSentence reads one complete sentence from the connection.
//
// A sentence consists of:
//  1. A command/reply word (e.g., "!re", "!done", "/login")
//  2. Zero or more attribute words (e.g., "=address=10.0.0.1/24")
//  3. A zero-length terminator word
//
// Empty sentences (zero-length first word) are skipped automatically.
func (r *Reader) ReadSentence() (*Sentence, error) {
	s := &Sentence{
		Map: make(map[string]string),
	}

	// Read the first word. Skip any empty leading words
	// (some RouterOS versions send empty sentences between responses).
	first, err := r.readWord()
	if err != nil {
		return nil, err
	}

	for first == "" {
		first, err = r.readWord()
		if err != nil {
			return nil, err
		}
	}

	// The first non-empty word is the sentence type (e.g., "!re", "!done").
	s.Word = first

	// Read attribute words until the zero-length terminator.
	for {
		word, err := r.readWord()
		if err != nil {
			return nil, err
		}
		// Zero-length word marks the end of the sentence.
		if word == "" {
			break
		}

		// Parse the word based on its prefix.
		if strings.HasPrefix(word, ".tag=") {
			// .tag is stored separately for async command correlation.
			s.Tag = word[5:]
		} else if strings.HasPrefix(word, "=") || strings.HasPrefix(word, ".") {
			// Attribute words are stored in both List (ordered) and Map (indexed).
			k, v := ParseWord(word)
			s.List = append(s.List, Pair{Key: k, Value: v})
			s.Map[k] = v
		}
	}

	return s, nil
}
