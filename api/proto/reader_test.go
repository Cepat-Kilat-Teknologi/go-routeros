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

// Helper: build a wire-format sentence from words using Writer
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
	assert.Len(t, s.List, 1) // only =.proplist= is an attribute
}
