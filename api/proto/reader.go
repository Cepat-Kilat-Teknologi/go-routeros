// api/proto/reader.go
package proto

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Reader reads API sentences from a connection.
type Reader struct {
	r *bufio.Reader
}

// NewReader creates a new Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(r)}
}

// readLength decodes a variable-length prefix (1-5 bytes) from the stream.
func (r *Reader) readLength() (int, error) {
	b, err := r.r.ReadByte()
	if err != nil {
		return 0, err
	}

	switch {
	case b&0x80 == 0x00:
		return int(b), nil
	case b&0xC0 == 0x80:
		b2, err := r.r.ReadByte()
		if err != nil {
			return 0, err
		}
		return int(b&^0xC0)<<8 | int(b2), nil
	case b&0xE0 == 0xC0:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xE0)<<16 | int(buf[0])<<8 | int(buf[1]), nil
	case b&0xF0 == 0xE0:
		buf := make([]byte, 3)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xF0)<<24 | int(buf[0])<<16 | int(buf[1])<<8 | int(buf[2]), nil
	case b&0xF8 == 0xF0:
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
func (r *Reader) readWord() (string, error) {
	l, err := r.readLength()
	if err != nil {
		return "", err
	}
	if l == 0 {
		return "", nil
	}
	buf := make([]byte, l)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadSentence reads one complete sentence from the connection.
func (r *Reader) ReadSentence() (*Sentence, error) {
	s := &Sentence{
		Map: make(map[string]string),
	}

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

	s.Word = first

	for {
		word, err := r.readWord()
		if err != nil {
			return nil, err
		}
		if word == "" {
			break
		}

		if strings.HasPrefix(word, ".tag=") {
			s.Tag = word[5:]
		} else if strings.HasPrefix(word, "=") || strings.HasPrefix(word, ".") {
			k, v := ParseWord(word)
			s.List = append(s.List, Pair{Key: k, Value: v})
			s.Map[k] = v
		}
	}

	return s, nil
}
