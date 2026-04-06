# go-routeros/api Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `api` package implementing the MikroTik RouterOS API Protocol (TCP-based, binary wire format) with sync CRUD operations, auto-detect authentication, and 100% test coverage.

**Architecture:** Subpackage `api/proto/` handles low-level wire protocol (length-prefix encoding, sentence parsing). Package `api/` provides `Client` with `Dial`/`Close`, login, and CRUD methods. All tests use in-memory byte buffers or mock TCP connections — no real network required.

**Tech Stack:** Go 1.21+, testify v1.8.4, crypto/md5 for legacy auth, crypto/tls for API-SSL

---

## File Map

| File | Responsibility |
|---|---|
| `api/proto/sentence.go` | `Pair` and `Sentence` structs, word parsing helpers |
| `api/proto/sentence_test.go` | Tests for Sentence |
| `api/proto/writer.go` | Length-prefix encoding, word/sentence writing |
| `api/proto/writer_test.go` | Tests for Writer |
| `api/proto/reader.go` | Length-prefix decoding, word/sentence reading |
| `api/proto/reader_test.go` | Tests for Reader |
| `api/constant.go` | Port and reply word constants |
| `api/errors.go` | `DeviceError`, `FatalError` |
| `api/errors_test.go` | Tests for errors |
| `api/options.go` | `RequestOption`, `WithProplist`, `WithQuery` |
| `api/options_test.go` | Tests for options |
| `api/reply.go` | `Reply` struct |
| `api/reply_test.go` | Tests for Reply |
| `api/client.go` | `Client`, `Dial`, `Close`, `ClientOption`, login, CRUD methods |
| `api/client_test.go` | Tests for Client |
| `api/doc.go` | Package documentation |
| `example/api/basic/main.go` | Basic usage example |
| `example/api/query/main.go` | Query filtering example |
| `README.md` | Update with api/ section |

---

### Task 1: Sentence Struct (api/proto/sentence.go + test)

**Files:**
- Create: `api/proto/sentence.go`
- Create: `api/proto/sentence_test.go`

- [ ] **Step 1: Write failing tests**

```go
// api/proto/sentence_test.go
package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPair(t *testing.T) {
	p := Pair{Key: "address", Value: "10.0.0.1/24"}
	assert.Equal(t, "address", p.Key)
	assert.Equal(t, "10.0.0.1/24", p.Value)
}

func TestParseWord_Attribute(t *testing.T) {
	tests := []struct {
		name  string
		word  string
		key   string
		value string
	}{
		{"simple", "=address=10.0.0.1/24", "address", "10.0.0.1/24"},
		{"dotid", "=.id=*1", ".id", "*1"},
		{"empty value", "=disabled=", "disabled", ""},
		{"equals in value", "=comment=a=b", "comment", "a=b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, v := ParseWord(tt.word)
			assert.Equal(t, tt.key, k)
			assert.Equal(t, tt.value, v)
		})
	}
}

func TestParseWord_Tag(t *testing.T) {
	k, v := ParseWord(".tag=cmd1")
	assert.Equal(t, ".tag", k)
	assert.Equal(t, "cmd1", v)
}

func TestParseWord_Query(t *testing.T) {
	// Query words are not parsed into key-value, they stay raw
	k, v := ParseWord("?=interface=ether1")
	assert.Equal(t, "?=interface=ether1", k)
	assert.Equal(t, "", v)
}

func TestParseWord_Command(t *testing.T) {
	k, v := ParseWord("!done")
	assert.Equal(t, "!done", k)
	assert.Equal(t, "", v)
}

func TestSentence_String(t *testing.T) {
	s := &Sentence{
		Word: "!re",
		List: []Pair{
			{Key: ".id", Value: "*1"},
			{Key: "address", Value: "10.0.0.1/24"},
		},
		Map: map[string]string{
			".id":     "*1",
			"address": "10.0.0.1/24",
		},
	}
	assert.Equal(t, "!re", s.Word)
	assert.Len(t, s.List, 2)
	assert.Equal(t, "*1", s.Map[".id"])
	assert.Equal(t, "10.0.0.1/24", s.Map["address"])
}

func TestSentence_GetPair(t *testing.T) {
	s := &Sentence{
		Word: "!done",
		List: []Pair{{Key: "ret", Value: "abc123"}},
		Map:  map[string]string{"ret": "abc123"},
	}
	val, ok := s.Get("ret")
	assert.True(t, ok)
	assert.Equal(t, "abc123", val)

	_, ok = s.Get("nonexistent")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v -run "TestPair|TestParseWord|TestSentence"
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Implement Sentence**

```go
// api/proto/sentence.go
package proto

import "strings"

// Pair represents a key-value pair from an API attribute word.
type Pair struct {
	Key   string
	Value string
}

// Sentence represents a single API sentence (sequence of words).
type Sentence struct {
	Word string            // First word: reply type ("!re", "!done", "!trap", "!fatal")
	List []Pair            // Ordered key-value pairs
	Map  map[string]string // Same data indexed by key
	Tag  string            // Value of .tag if present
}

// Get returns the value for a key and whether it exists.
func (s *Sentence) Get(key string) (string, bool) {
	v, ok := s.Map[key]
	return v, ok
}

// ParseWord parses an API word into a key-value pair.
// Attribute words (=key=value) return (key, value).
// Tag words (.tag=value) return (".tag", value).
// Other words (commands, queries, replies) return (word, "").
func ParseWord(word string) (string, string) {
	// Attribute word: =key=value
	if strings.HasPrefix(word, "=") {
		word = word[1:] // strip leading =
		idx := strings.Index(word, "=")
		if idx >= 0 {
			return word[:idx], word[idx+1:]
		}
		return word, ""
	}

	// Tag word: .tag=value
	if strings.HasPrefix(word, ".") {
		idx := strings.Index(word, "=")
		if idx >= 0 {
			return word[:idx], word[idx+1:]
		}
		return word, ""
	}

	// Reply words (!re, !done, !trap, !fatal) and query words (?...)
	return word, ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v -run "TestPair|TestParseWord|TestSentence"
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/proto/sentence.go api/proto/sentence_test.go
git commit -m "feat(api/proto): add Sentence and Pair structs with word parsing"
```

---

### Task 2: Writer — Length Encoding + Sentence Writing (api/proto/writer.go + test)

**Files:**
- Create: `api/proto/writer.go`
- Create: `api/proto/writer_test.go`

- [ ] **Step 1: Write failing tests**

```go
// api/proto/writer_test.go
package proto

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeLength(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		expected []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"max 1-byte (127)", 0x7F, []byte{0x7F}},
		{"min 2-byte (128)", 0x80, []byte{0x80, 0x80}},
		{"max 2-byte (16383)", 0x3FFF, []byte{0xBF, 0xFF}},
		{"min 3-byte (16384)", 0x4000, []byte{0xC0, 0x40, 0x00}},
		{"max 3-byte (2097151)", 0x1FFFFF, []byte{0xDF, 0xFF, 0xFF}},
		{"min 4-byte (2097152)", 0x200000, []byte{0xE0, 0x20, 0x00, 0x00}},
		{"max 4-byte (268435455)", 0xFFFFFFF, []byte{0xEF, 0xFF, 0xFF, 0xFF}},
		{"min 5-byte (268435456)", 0x10000000, []byte{0xF0, 0x10, 0x00, 0x00, 0x00}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeLength(tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriter_WriteWord(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.BeginSentence()
	w.WriteWord("/login")
	err := w.EndSentence()
	require.NoError(t, err)

	data := buf.Bytes()
	// 6 bytes for "/login" + 1 byte length prefix + 1 byte zero terminator
	assert.Equal(t, byte(6), data[0])          // length of "/login"
	assert.Equal(t, []byte("/login"), data[1:7]) // word content
	assert.Equal(t, byte(0), data[7])           // zero-length terminator
}

func TestWriter_WriteSentence_MultipleWords(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.BeginSentence()
	w.WriteWord("/login")
	w.WriteWord("=name=admin")
	w.WriteWord("=password=secret")
	err := w.EndSentence()
	require.NoError(t, err)

	// Verify we can read it back by checking the buffer is non-empty
	// and ends with a zero byte
	data := buf.Bytes()
	assert.Greater(t, len(data), 3)
	assert.Equal(t, byte(0), data[len(data)-1]) // terminator
}

func TestWriter_WriteWord_LongWord(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Create a word longer than 127 bytes (requires 2-byte length encoding)
	longWord := strings.Repeat("a", 200)

	w.BeginSentence()
	w.WriteWord(longWord)
	err := w.EndSentence()
	require.NoError(t, err)

	data := buf.Bytes()
	// First two bytes should be 2-byte length encoding for 200
	assert.Equal(t, byte(0x80), data[0]&0xC0) // top 2 bits = 10
	// Word content starts at offset 2
	assert.Equal(t, longWord, string(data[2:202]))
}

func TestWriter_EmptySentence(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.BeginSentence()
	err := w.EndSentence()
	require.NoError(t, err)

	data := buf.Bytes()
	assert.Equal(t, []byte{0x00}, data) // just the terminator
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v -run "TestEncodeLength|TestWriter"
```

Expected: FAIL — `encodeLength`, `Writer` not defined.

- [ ] **Step 3: Implement Writer**

```go
// api/proto/writer.go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v -run "TestEncodeLength|TestWriter"
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/proto/writer.go api/proto/writer_test.go
git commit -m "feat(api/proto): add Writer with length-prefix encoding"
```

---

### Task 3: Reader — Length Decoding + Sentence Reading (api/proto/reader.go + test)

**Files:**
- Create: `api/proto/reader.go`
- Create: `api/proto/reader_test.go`

- [ ] **Step 1: Write failing tests**

```go
// api/proto/reader_test.go
package proto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadLength(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int
	}{
		{"zero", []byte{0x00}, 0},
		{"one", []byte{0x01}, 1},
		{"max 1-byte", []byte{0x7F}, 0x7F},
		{"min 2-byte", []byte{0x80, 0x80}, 0x80},
		{"max 2-byte", []byte{0xBF, 0xFF}, 0x3FFF},
		{"min 3-byte", []byte{0xC0, 0x40, 0x00}, 0x4000},
		{"max 3-byte", []byte{0xDF, 0xFF, 0xFF}, 0x1FFFFF},
		{"min 4-byte", []byte{0xE0, 0x20, 0x00, 0x00}, 0x200000},
		{"max 4-byte", []byte{0xEF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFF},
		{"min 5-byte", []byte{0xF0, 0x10, 0x00, 0x00, 0x00}, 0x10000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReader(bytes.NewReader(tt.input))
			length, err := r.readLength()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, length)
		})
	}
}

func TestReadLength_EOF(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte{}))
	_, err := r.readLength()
	assert.Error(t, err)
}

func TestReadWord(t *testing.T) {
	// Encode "/login" manually: length 6 (0x06) + "/login"
	data := append([]byte{0x06}, []byte("/login")...)
	r := NewReader(bytes.NewReader(data))

	word, err := r.readWord()
	require.NoError(t, err)
	assert.Equal(t, "/login", word)
}

func TestReadWord_ZeroLength(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte{0x00}))
	word, err := r.readWord()
	require.NoError(t, err)
	assert.Equal(t, "", word)
}

// Helper: build a wire-format sentence from words
func buildSentence(words ...string) []byte {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	w.BeginSentence()
	for _, word := range words {
		w.WriteWord(word)
	}
	w.EndSentence()
	return buf.Bytes()
}

func TestReadSentence_Reply(t *testing.T) {
	data := buildSentence("!re", "=.id=*1", "=address=10.0.0.1/24")
	r := NewReader(bytes.NewReader(data))

	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!re", s.Word)
	assert.Equal(t, "*1", s.Map[".id"])
	assert.Equal(t, "10.0.0.1/24", s.Map["address"])
	assert.Len(t, s.List, 2)
}

func TestReadSentence_Done(t *testing.T) {
	data := buildSentence("!done")
	r := NewReader(bytes.NewReader(data))

	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!done", s.Word)
	assert.Empty(t, s.List)
}

func TestReadSentence_DoneWithRet(t *testing.T) {
	data := buildSentence("!done", "=ret=abc123")
	r := NewReader(bytes.NewReader(data))

	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!done", s.Word)
	assert.Equal(t, "abc123", s.Map["ret"])
}

func TestReadSentence_Trap(t *testing.T) {
	data := buildSentence("!trap", "=category=1", "=message=bad argument")
	r := NewReader(bytes.NewReader(data))

	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!trap", s.Word)
	assert.Equal(t, "1", s.Map["category"])
	assert.Equal(t, "bad argument", s.Map["message"])
}

func TestReadSentence_WithTag(t *testing.T) {
	data := buildSentence("!re", "=name=ether1", ".tag=cmd1")
	r := NewReader(bytes.NewReader(data))

	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!re", s.Word)
	assert.Equal(t, "cmd1", s.Tag)
	assert.Equal(t, "ether1", s.Map["name"])
}

func TestReadSentence_MultipleSentences(t *testing.T) {
	// Two sentences back to back
	var buf bytes.Buffer
	buf.Write(buildSentence("!re", "=.id=*1"))
	buf.Write(buildSentence("!done"))

	r := NewReader(&buf)

	s1, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!re", s1.Word)

	s2, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!done", s2.Word)
}

func TestReadSentence_EOF(t *testing.T) {
	r := NewReader(bytes.NewReader([]byte{}))
	_, err := r.ReadSentence()
	assert.ErrorIs(t, err, io.EOF)
}

func TestRoundTrip_WriteThenRead(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.BeginSentence()
	w.WriteWord("/ip/address/print")
	w.WriteWord("=.proplist=address,interface")
	w.WriteWord("?=interface=ether1")
	w.WriteWord(".tag=test1")
	w.EndSentence()

	r := NewReader(&buf)
	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "/ip/address/print", s.Word)
	assert.Equal(t, "address,interface", s.Map[".proplist"])
	assert.Equal(t, "test1", s.Tag)
	// Query words are not parsed as key-value
	assert.Len(t, s.List, 1) // only =.proplist= is an attribute
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v -run "TestRead|TestRoundTrip"
```

Expected: FAIL — `Reader`, `readLength`, `readWord`, `ReadSentence` not defined.

- [ ] **Step 3: Implement Reader**

```go
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
		// 1 byte: 0xxxxxxx
		return int(b), nil
	case b&0xC0 == 0x80:
		// 2 bytes: 10xxxxxx
		b2, err := r.r.ReadByte()
		if err != nil {
			return 0, err
		}
		return int(b&^0xC0)<<8 | int(b2), nil
	case b&0xE0 == 0xC0:
		// 3 bytes: 110xxxxx
		buf := make([]byte, 2)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xE0)<<16 | int(buf[0])<<8 | int(buf[1]), nil
	case b&0xF0 == 0xE0:
		// 4 bytes: 1110xxxx
		buf := make([]byte, 3)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return 0, err
		}
		return int(b&^0xF0)<<24 | int(buf[0])<<16 | int(buf[1])<<8 | int(buf[2]), nil
	case b&0xF8 == 0xF0:
		// 5 bytes: 11110xxx + 4 bytes
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
// A sentence is a sequence of words terminated by a zero-length word.
// Returns the parsed Sentence.
func (r *Reader) ReadSentence() (*Sentence, error) {
	s := &Sentence{
		Map: make(map[string]string),
	}

	// Read first word (command/reply word)
	first, err := r.readWord()
	if err != nil {
		return nil, err
	}

	// Skip empty sentences (just a zero-length word)
	for first == "" {
		first, err = r.readWord()
		if err != nil {
			return nil, err
		}
	}

	s.Word = first

	// Read remaining words until zero-length terminator
	for {
		word, err := r.readWord()
		if err != nil {
			return nil, err
		}
		if word == "" {
			break // sentence terminator
		}

		// Parse the word
		if strings.HasPrefix(word, ".tag=") {
			s.Tag = word[5:]
		} else if strings.HasPrefix(word, "=") || strings.HasPrefix(word, ".") {
			k, v := ParseWord(word)
			s.List = append(s.List, Pair{Key: k, Value: v})
			s.Map[k] = v
		}
		// Query words (?...) and other words are not stored in List/Map
	}

	return s, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/proto/ -v
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/proto/reader.go api/proto/reader_test.go
git commit -m "feat(api/proto): add Reader with length-prefix decoding and sentence parsing"
```

---

### Task 4: Constants + Errors + Options (api/)

**Files:**
- Create: `api/constant.go`
- Create: `api/errors.go`
- Create: `api/errors_test.go`
- Create: `api/options.go`
- Create: `api/options_test.go`

- [ ] **Step 1: Write failing tests for errors**

```go
// api/errors_test.go
package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceError_Error(t *testing.T) {
	err := &DeviceError{Category: 1, Message: "bad argument"}
	assert.Equal(t, "routeros: trap: bad argument (category 1)", err.Error())
}

func TestDeviceError_TypeAssertion(t *testing.T) {
	var err error = &DeviceError{Category: 0, Message: "missing"}
	de, ok := err.(*DeviceError)
	assert.True(t, ok)
	assert.Equal(t, 0, de.Category)
}

func TestFatalError_Error(t *testing.T) {
	err := &FatalError{Message: "session terminated"}
	assert.Equal(t, "routeros: fatal: session terminated", err.Error())
}

func TestFatalError_TypeAssertion(t *testing.T) {
	var err error = &FatalError{Message: "gone"}
	fe, ok := err.(*FatalError)
	assert.True(t, ok)
	assert.Equal(t, "gone", fe.Message)
}
```

- [ ] **Step 2: Write failing tests for options**

```go
// api/options_test.go
package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithProplist(t *testing.T) {
	opts := collectRequestOptions(WithProplist("address", "interface"))
	assert.Equal(t, []string{"address", "interface"}, opts.proplist)
}

func TestWithQuery(t *testing.T) {
	opts := collectRequestOptions(WithQuery("?=interface=ether1", "?type=ether", "?#|"))
	assert.Equal(t, []string{"?=interface=ether1", "?type=ether", "?#|"}, opts.query)
}

func TestCollectRequestOptions_Empty(t *testing.T) {
	opts := collectRequestOptions()
	assert.Nil(t, opts.proplist)
	assert.Nil(t, opts.query)
}

func TestCollectRequestOptions_Combined(t *testing.T) {
	opts := collectRequestOptions(
		WithProplist("name", "type"),
		WithQuery("?type=ether"),
	)
	assert.Equal(t, []string{"name", "type"}, opts.proplist)
	assert.Equal(t, []string{"?type=ether"}, opts.query)
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/ -v
```

Expected: FAIL — types not defined.

- [ ] **Step 4: Implement constants, errors, options**

```go
// api/constant.go
package api

const (
	DefaultPort    = "8728"
	DefaultTLSPort = "8729"

	replyRe    = "!re"
	replyDone  = "!done"
	replyTrap  = "!trap"
	replyFatal = "!fatal"
)
```

```go
// api/errors.go
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
```

```go
// api/options.go
package api

import "strings"

// RequestOption configures a single API request.
type RequestOption func(*requestOptions)

type requestOptions struct {
	proplist []string
	query    []string
}

// WithProplist limits which properties are returned.
// Sent as ".proplist=address,interface" API attribute word.
func WithProplist(props ...string) RequestOption {
	return func(o *requestOptions) {
		o.proplist = props
	}
}

// WithQuery adds query words for filtering.
// Each entry is sent as a separate query word.
// User provides raw API Protocol query syntax including the ? prefix.
func WithQuery(query ...string) RequestOption {
	return func(o *requestOptions) {
		o.query = query
	}
}

// collectRequestOptions applies all options and returns the result.
func collectRequestOptions(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// proplistWord builds the .proplist API attribute word.
func proplistWord(props []string) string {
	return ".proplist=" + strings.Join(props, ",")
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/ -v
```

Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/constant.go api/errors.go api/errors_test.go api/options.go api/options_test.go
git commit -m "feat(api): add constants, error types, and request options"
```

---

### Task 5: Reply Struct (api/reply.go + test)

**Files:**
- Create: `api/reply.go`
- Create: `api/reply_test.go`

- [ ] **Step 1: Write failing tests**

```go
// api/reply_test.go
package api

import (
	"testing"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
	"github.com/stretchr/testify/assert"
)

func TestReply_Empty(t *testing.T) {
	r := &Reply{
		Done: &proto.Sentence{Word: "!done", Map: map[string]string{}},
	}
	assert.Empty(t, r.Re)
	assert.NotNil(t, r.Done)
}

func TestReply_WithData(t *testing.T) {
	r := &Reply{
		Re: []*proto.Sentence{
			{Word: "!re", Map: map[string]string{".id": "*1", "address": "10.0.0.1/24"}},
			{Word: "!re", Map: map[string]string{".id": "*2", "address": "192.168.1.1/24"}},
		},
		Done: &proto.Sentence{Word: "!done", Map: map[string]string{}},
	}
	assert.Len(t, r.Re, 2)
	assert.Equal(t, "10.0.0.1/24", r.Re[0].Map["address"])
	assert.Equal(t, "192.168.1.1/24", r.Re[1].Map["address"])
}

func TestReply_DoneWithRet(t *testing.T) {
	r := &Reply{
		Done: &proto.Sentence{
			Word: "!done",
			Map:  map[string]string{"ret": "*A"},
			List: []proto.Pair{{Key: "ret", Value: "*A"}},
		},
	}
	val, ok := r.Done.Get("ret")
	assert.True(t, ok)
	assert.Equal(t, "*A", val)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/ -v -run "TestReply"
```

Expected: FAIL — `Reply` not defined.

- [ ] **Step 3: Implement Reply**

```go
// api/reply.go
package api

import "github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"

// Reply holds the complete response to a command.
type Reply struct {
	Re   []*proto.Sentence // Data sentences (!re)
	Done *proto.Sentence   // Completion sentence (!done)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/ -v -run "TestReply"
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/reply.go api/reply_test.go
git commit -m "feat(api): add Reply struct"
```

---

### Task 6: Client — Dial, Close, Login, CRUD (api/client.go + test)

**Files:**
- Create: `api/client.go`
- Create: `api/client_test.go`
- Create: `api/doc.go`

This is the largest task. The Client ties everything together. Tests use a mock TCP connection (in-memory pipe) to simulate RouterOS responses.

- [ ] **Step 1: Write failing tests**

```go
// api/client_test.go
package api

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockServer simulates a RouterOS API server using an in-memory pipe.
// It reads sentences from the client and writes pre-configured responses.
type mockServer struct {
	conn   net.Conn
	reader *proto.Reader
	writer *proto.Writer
}

func newMockServer(t *testing.T) (*mockServer, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	return &mockServer{
		conn:   server,
		reader: proto.NewReader(server),
		writer: proto.NewWriter(server),
	}, client
}

func (m *mockServer) readSentence() (*proto.Sentence, error) {
	return m.reader.ReadSentence()
}

func (m *mockServer) writeSentence(words ...string) error {
	m.writer.BeginSentence()
	for _, w := range words {
		m.writer.WriteWord(w)
	}
	return m.writer.EndSentence()
}

func (m *mockServer) close() {
	m.conn.Close()
}

// handleLogin simulates post-6.43 login (success)
func (m *mockServer) handleLogin() {
	m.readSentence() // read /login sentence
	m.writeSentence("!done")
}

// handleLoginLegacy simulates pre-6.43 login (MD5 challenge-response)
func (m *mockServer) handleLoginLegacy(challenge string) {
	m.readSentence() // read /login with name+password
	m.writeSentence("!done", "=ret="+challenge)
	m.readSentence() // read /login with name+response
	m.writeSentence("!done")
}

func TestDial_PostV643(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go srv.handleLogin()

	client := newClientFromConn(clientConn, "admin", "")
	err := client.login("admin", "")
	require.NoError(t, err)
	client.Close()
}

func TestDial_PreV643(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	// Hex-encoded challenge bytes
	go srv.handleLoginLegacy("abcdef0123456789")

	client := newClientFromConn(clientConn, "admin", "secret")
	err := client.login("admin", "secret")
	require.NoError(t, err)
	client.Close()
}

func TestDial_LoginFailure(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	go func() {
		srv.readSentence()
		srv.writeSentence("!trap", "=category=5", "=message=invalid user name or password")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "wrong")
	err := client.login("admin", "wrong")
	require.Error(t, err)

	de, ok := err.(*DeviceError)
	require.True(t, ok)
	assert.Equal(t, 5, de.Category)
	assert.Contains(t, de.Message, "invalid user")
	client.Close()
}

func TestClient_Close(t *testing.T) {
	srv, clientConn := newMockServer(t)
	defer srv.close()

	client := newClientFromConn(clientConn, "", "")
	err := client.Close()
	assert.NoError(t, err)
}

func TestClient_Print(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence() // read /ip/address/print
		srv.writeSentence("!re", "=.id=*1", "=address=10.0.0.1/24", "=interface=ether1")
		srv.writeSentence("!re", "=.id=*2", "=address=192.168.1.1/24", "=interface=ether2")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Len(t, reply.Re, 2)
	assert.Equal(t, "10.0.0.1/24", reply.Re[0].Map["address"])
	assert.Equal(t, "192.168.1.1/24", reply.Re[1].Map["address"])
	client.Close()
}

func TestClient_Print_WithProplist(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		// Verify proplist word was sent
		assert.Equal(t, "/ip/address/print", s.Word)
		assert.Equal(t, "address,interface", s.Map[".proplist"])

		srv.writeSentence("!re", "=address=10.0.0.1/24", "=interface=ether1")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address",
		WithProplist("address", "interface"),
	)
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	client.Close()
}

func TestClient_Print_WithQuery(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()
		srv.readSentence()
		srv.writeSentence("!re", "=.id=*1", "=type=ether")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/interface",
		WithQuery("?type=ether"),
	)
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	client.Close()
}

func TestClient_Add(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/add", s.Word)
		assert.Equal(t, "10.0.0.1/24", s.Map["address"])
		assert.Equal(t, "ether1", s.Map["interface"])

		srv.writeSentence("!done", "=ret=*A")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Add(context.Background(), "/ip/address", map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	require.NoError(t, err)
	assert.Equal(t, "*A", reply.Done.Map["ret"])
	client.Close()
}

func TestClient_Set(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/set", s.Word)
		assert.Equal(t, "*1", s.Map[".id"])
		assert.Equal(t, "updated", s.Map["comment"])

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Set(context.Background(), "/ip/address", map[string]string{
		".id":     "*1",
		"comment": "updated",
	})
	require.NoError(t, err)
	client.Close()
}

func TestClient_Remove(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/ip/address/remove", s.Word)
		assert.Equal(t, "*1", s.Map[".id"])

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Remove(context.Background(), "/ip/address", "*1")
	require.NoError(t, err)
	client.Close()
}

func TestClient_Run(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		s, _ := srv.readSentence()
		assert.Equal(t, "/system/reboot", s.Word)

		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Run(context.Background(), "/system/reboot", nil)
	require.NoError(t, err)
	client.Close()
}

func TestClient_TrapError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!trap", "=category=0", "=message=no such command")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Print(context.Background(), "/nonexistent")
	require.Error(t, err)
	de, ok := err.(*DeviceError)
	require.True(t, ok)
	assert.Equal(t, 0, de.Category)
	assert.Equal(t, "no such command", de.Message)
	client.Close()
}

func TestClient_FatalError(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!fatal", "=message=session terminated")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	_, err := client.Print(context.Background(), "/ip/address")
	require.Error(t, err)
	fe, ok := err.(*FatalError)
	require.True(t, ok)
	assert.Equal(t, "session terminated", fe.Message)
	client.Close()
}

func TestClient_Auth(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence() // /system/resource/print
		srv.writeSentence("!re", "=board-name=hAP", "=version=7.14", "=platform=MikroTik")
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Auth(context.Background())
	require.NoError(t, err)
	assert.Len(t, reply.Re, 1)
	assert.Equal(t, "MikroTik", reply.Re[0].Map["platform"])
	client.Close()
}

func TestClientOption_WithTimeout(t *testing.T) {
	cfg := &clientConfig{}
	WithTimeout(30 * time.Second)(cfg)
	assert.Equal(t, 30*time.Second, cfg.timeout)
}

func TestClientOption_WithTLS(t *testing.T) {
	cfg := &clientConfig{}
	WithTLS(true)(cfg)
	assert.True(t, cfg.useTLS)
}

func TestClientOption_WithTLSConfig(t *testing.T) {
	tc := &tls.Config{InsecureSkipVerify: true}
	cfg := &clientConfig{}
	WithTLSConfig(tc)(cfg)
	assert.Equal(t, tc, cfg.tlsConfig)
}

func TestResolveAddress_DefaultPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1", false)
	assert.Equal(t, "192.168.88.1:8728", addr)
}

func TestResolveAddress_DefaultTLSPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1", true)
	assert.Equal(t, "192.168.88.1:8729", addr)
}

func TestResolveAddress_CustomPort(t *testing.T) {
	addr := resolveAddress("192.168.88.1:9000", false)
	assert.Equal(t, "192.168.88.1:9000", addr)
}

func TestClient_EmptyReply(t *testing.T) {
	srv, clientConn := newMockServer(t)

	go func() {
		defer srv.close()
		srv.handleLogin()

		srv.readSentence()
		srv.writeSentence("!done")
	}()

	client := newClientFromConn(clientConn, "admin", "")
	require.NoError(t, client.login("admin", ""))

	reply, err := client.Print(context.Background(), "/ip/address")
	require.NoError(t, err)
	assert.Empty(t, reply.Re)
	assert.NotNil(t, reply.Done)
	client.Close()
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/ -v
```

Expected: FAIL — `Client`, `Dial`, etc. not defined.

- [ ] **Step 3: Implement Client**

```go
// api/doc.go

// Package api implements the MikroTik RouterOS API Protocol (TCP, port 8728/8729).
// It works with all RouterOS versions (v6 and v7).
package api
```

```go
// api/client.go
package api

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"
)

// ClientOption configures the Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	useTLS    bool
	tlsConfig *tls.Config
	timeout   time.Duration
}

// Client holds a connection to a RouterOS device.
type Client struct {
	conn   io.ReadWriteCloser
	reader *proto.Reader
	writer *proto.Writer
}

// WithTLS enables TLS connection (port 8729 by default).
func WithTLS(useTLS bool) ClientOption {
	return func(c *clientConfig) {
		c.useTLS = useTLS
	}
}

// WithTLSConfig sets custom TLS configuration.
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *clientConfig) {
		c.tlsConfig = config
		c.useTLS = true
	}
}

// WithTimeout sets the connection timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// resolveAddress ensures address has a port. Uses default API port if missing.
func resolveAddress(address string, useTLS bool) string {
	if _, _, err := net.SplitHostPort(address); err != nil {
		if useTLS {
			return address + ":" + DefaultTLSPort
		}
		return address + ":" + DefaultPort
	}
	return address
}

// newClientFromConn creates a Client from an existing connection (for testing).
func newClientFromConn(conn io.ReadWriteCloser, username, password string) *Client {
	return &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}
}

// Dial connects to a RouterOS device and authenticates.
func Dial(address, username, password string, opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	addr := resolveAddress(address, cfg.useTLS)

	var conn net.Conn
	var err error

	dialer := net.Dialer{Timeout: cfg.timeout}

	if cfg.useTLS {
		tlsCfg := cfg.tlsConfig
		if tlsCfg == nil {
			tlsCfg = &tls.Config{}
		}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("routeros: dial %s: %w", addr, err)
	}

	c := &Client{
		conn:   conn,
		reader: proto.NewReader(conn),
		writer: proto.NewWriter(conn),
	}

	if err := c.login(username, password); err != nil {
		conn.Close()
		return nil, err
	}

	return c, nil
}

// Close closes the TCP connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// login handles authentication, auto-detecting pre/post-6.43.
func (c *Client) login(username, password string) error {
	// Send /login with name and password (post-6.43 style)
	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=password=" + password)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	// Read response
	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	// Handle !trap (login failed)
	if sen.Word == replyTrap {
		// Read the !done that follows !trap
		c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	// Handle !fatal
	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	// Check for pre-6.43 challenge-response
	if challenge, ok := sen.Get("ret"); ok {
		return c.loginLegacy(username, password, challenge)
	}

	// Post-6.43: !done without ret means success
	return nil
}

// loginLegacy handles pre-6.43 MD5 challenge-response login.
func (c *Client) loginLegacy(username, password, challenge string) error {
	// Decode hex challenge
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return fmt.Errorf("routeros: decode challenge: %w", err)
	}

	// Compute MD5(0x00 + password + challenge)
	h := md5.New()
	h.Write([]byte{0})
	h.Write([]byte(password))
	h.Write(challengeBytes)
	response := fmt.Sprintf("00%x", h.Sum(nil))

	// Send /login with name and response
	c.writer.BeginSentence()
	c.writer.WriteWord("/login")
	c.writer.WriteWord("=name=" + username)
	c.writer.WriteWord("=response=" + response)
	if err := c.writer.EndSentence(); err != nil {
		return fmt.Errorf("routeros: login write: %w", err)
	}

	// Read response
	sen, err := c.reader.ReadSentence()
	if err != nil {
		return fmt.Errorf("routeros: login read: %w", err)
	}

	if sen.Word == replyTrap {
		c.reader.ReadSentence()
		return parseTrapError(sen)
	}

	if sen.Word == replyFatal {
		return parseFatalError(sen)
	}

	return nil
}

// sendCommand sends a command sentence with optional params and options.
func (c *Client) sendCommand(command string, params map[string]string, opts *requestOptions) error {
	c.writer.BeginSentence()
	c.writer.WriteWord(command)

	// Write attribute words
	for k, v := range params {
		c.writer.WriteWord("=" + k + "=" + v)
	}

	// Write .proplist if specified
	if len(opts.proplist) > 0 {
		c.writer.WriteWord(proplistWord(opts.proplist))
	}

	// Write query words
	for _, q := range opts.query {
		c.writer.WriteWord(q)
	}

	return c.writer.EndSentence()
}

// readReply reads sentences until !done or !fatal, collecting !re sentences.
func (c *Client) readReply() (*Reply, error) {
	reply := &Reply{}
	var trapErr error

	for {
		sen, err := c.reader.ReadSentence()
		if err != nil {
			return nil, err
		}

		switch sen.Word {
		case replyRe:
			reply.Re = append(reply.Re, sen)
		case replyDone:
			reply.Done = sen
			if trapErr != nil {
				return nil, trapErr
			}
			return reply, nil
		case replyTrap:
			trapErr = parseTrapError(sen)
		case replyFatal:
			return nil, parseFatalError(sen)
		}
	}
}

// execute sends a command and reads the reply.
func (c *Client) execute(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	reqOpts := collectRequestOptions(opts...)

	if err := c.sendCommand(command, params, reqOpts); err != nil {
		return nil, fmt.Errorf("routeros: send: %w", err)
	}

	return c.readReply()
}

// Auth verifies the connection by querying system resource info.
func (c *Client) Auth(ctx context.Context) (*Reply, error) {
	return c.execute(ctx, "/system/resource/print", nil)
}

// Print retrieves data (sends /command/print).
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/print", nil, opts...)
}

// Add creates a new record (sends /command/add).
func (c *Client) Add(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/add", params, opts...)
}

// Set updates a record (sends /command/set).
func (c *Client) Set(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/set", params, opts...)
}

// Remove deletes a record (sends /command/remove with =.id=).
func (c *Client) Remove(ctx context.Context, command string, id string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command+"/remove", map[string]string{".id": id}, opts...)
}

// Run executes an arbitrary command with optional parameters.
func (c *Client) Run(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error) {
	return c.execute(ctx, command, params, opts...)
}

// parseTrapError creates a DeviceError from a !trap sentence.
func parseTrapError(sen *proto.Sentence) *DeviceError {
	category := 0
	if catStr, ok := sen.Get("category"); ok {
		category, _ = strconv.Atoi(catStr)
	}
	msg, _ := sen.Get("message")
	return &DeviceError{
		Category: category,
		Message:  msg,
	}
}

// parseFatalError creates a FatalError from a !fatal sentence.
func parseFatalError(sen *proto.Sentence) *FatalError {
	msg, _ := sen.Get("message")
	return &FatalError{Message: msg}
}
```

- [ ] **Step 4: Run ALL tests**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/... -v
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add api/client.go api/client_test.go api/doc.go
git commit -m "feat(api): add Client with Dial, login, and CRUD methods"
```

---

### Task 7: Examples and README Update

**Files:**
- Create: `example/api/basic/main.go`
- Create: `example/api/query/main.go`
- Modify: `README.md`

- [ ] **Step 1: Create basic example**

```go
// example/api/basic/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Auth
	reply, err := client.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Platform:", reply.Re[0].Map["platform"])

	// Print IP addresses
	reply, err = client.Print(ctx, "/ip/address",
		api.WithProplist("address", "interface"),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, re := range reply.Re {
		fmt.Printf("  %s on %s\n", re.Map["address"], re.Map["interface"])
	}

	// Add
	reply, err = client.Add(ctx, "/ip/address", map[string]string{
		"address":   "10.0.0.1/24",
		"interface": "ether1",
	})
	if err != nil {
		log.Fatal(err)
	}
	id, _ := reply.Done.Get("ret")
	fmt.Println("Added:", id)

	// Remove
	_, err = client.Remove(ctx, "/ip/address", id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed:", id)
}
```

- [ ] **Step 2: Create query example**

```go
// example/api/query/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
	client, err := api.Dial("192.168.88.1", "admin", "")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Query: find ether OR vlan interfaces
	reply, err := client.Print(ctx, "/interface",
		api.WithProplist("name", "type"),
		api.WithQuery("?type=ether", "?type=vlan", "?#|"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range reply.Re {
		fmt.Printf("  %s (%s)\n", re.Map["name"], re.Map["type"])
	}
}
```

- [ ] **Step 3: Add api/ section to README.md**

Read the current README.md and add an "API Protocol" section after the existing REST API content. Add it before the "## Migration" section. The section should include:

```markdown
## API Protocol (v6 & v7)

For the TCP-based API Protocol (port 8728/8729), use the `api` package:

` ` `go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/api"

client, err := api.Dial("192.168.88.1", "admin", "")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

reply, err := client.Print(context.Background(), "/ip/address",
    api.WithProplist("address", "interface"),
)
` ` `

### Which package to use?

| Your RouterOS version | Package | Why |
|---|---|---|
| v6 | `api` | REST API not available in v6 |
| v7 (simple CRUD) | `rest` | Simpler HTTP-based API |
| v7 (advanced) | `api` | More powerful filtering, future streaming support |

### API Client Options

| Option | Description | Default |
|---|---|---|
| `WithTLS(true)` | Enable TLS (port 8729) | `false` |
| `WithTLSConfig(cfg)` | Custom TLS config | `nil` |
| `WithTimeout(d)` | Connection timeout | No timeout |

### API Error Handling

` ` `go
reply, err := client.Print(ctx, "/ip/address")
if err != nil {
    if de, ok := err.(*api.DeviceError); ok {
        fmt.Printf("Trap category %d: %s\n", de.Category, de.Message)
    }
    if fe, ok := err.(*api.FatalError); ok {
        fmt.Printf("Fatal: %s (connection closed)\n", fe.Message)
    }
}
` ` `
```

Note: Replace `` ` ` ` `` with actual triple backticks.

- [ ] **Step 4: Verify build**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
git add example/api/ README.md
git commit -m "docs: add API protocol examples and update README"
```

---

### Task 8: Coverage to 100% and Final Cleanup

**Files:**
- Modify: test files as needed

- [ ] **Step 1: Run coverage report**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./api/... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

- [ ] **Step 2: Add missing tests for any functions below 100%**

Identify functions below 100% from the coverage report. Common gaps:
- `encodeLength` edge cases
- `readLength` error branches (partial reads)
- `Dial` (hard to test without real network — test resolveAddress and newClientFromConn instead)
- `loginLegacy` hex decode error
- `parseTrapError` / `parseFatalError` edge cases

Add tests to cover all branches. Test file to modify depends on coverage output.

- [ ] **Step 3: Run go vet**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go vet ./...
```

- [ ] **Step 4: Run final test suite**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
go test ./... -cover
```

Expected: ALL PASS, 100% coverage for api/ and api/proto/, rest/ stays at 100%

- [ ] **Step 5: Commit**

```bash
cd /Users/sumitroajiprabowo/Projects/go-routeros
rm -f coverage.out
git add api/ 
git commit -m "test: achieve 100% coverage for api package"
```
