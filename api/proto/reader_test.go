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

func TestReadLength_Truncated2Byte(t *testing.T) {
	// First byte indicates 2-byte length (0x80-0xBF), but no second byte
	r := NewReader(bytes.NewReader([]byte{0x80}))
	_, err := r.readLength()
	assert.Error(t, err)
}

func TestReadLength_Truncated3Byte(t *testing.T) {
	// First byte indicates 3-byte length (0xC0-0xDF), but only 1 extra byte follows
	r := NewReader(bytes.NewReader([]byte{0xC0, 0x40}))
	_, err := r.readLength()
	assert.Error(t, err)
}

func TestReadLength_Truncated4Byte(t *testing.T) {
	// First byte indicates 4-byte length (0xE0-0xEF), but only 2 extra bytes follow
	r := NewReader(bytes.NewReader([]byte{0xE0, 0x20, 0x00}))
	_, err := r.readLength()
	assert.Error(t, err)
}

func TestReadLength_Truncated5Byte(t *testing.T) {
	// First byte indicates 5-byte length (0xF0), but only 3 extra bytes follow
	r := NewReader(bytes.NewReader([]byte{0xF0, 0x10, 0x00, 0x00}))
	_, err := r.readLength()
	assert.Error(t, err)
}

func TestReadLength_UnexpectedByte(t *testing.T) {
	// 0xF8 falls into the default case (reserved/unexpected)
	r := NewReader(bytes.NewReader([]byte{0xF8}))
	_, err := r.readLength()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected length byte")
}

func TestReadWord_TruncatedContent(t *testing.T) {
	// Length says 10 bytes but only 3 bytes of content follow
	data := []byte{0x0A, 'a', 'b', 'c'}
	r := NewReader(bytes.NewReader(data))
	_, err := r.readWord()
	assert.Error(t, err)
}

func TestReadSentence_ErrorMidWord(t *testing.T) {
	// Build a sentence that has the first word ok, then a word whose content is truncated
	// First word: "!re" (length 3, then 3 bytes, then we need another word with error)
	var buf bytes.Buffer
	buf.WriteByte(0x03) // length 3
	buf.WriteString("!re")
	buf.WriteByte(0x0A) // length 10 but we provide only 3 bytes
	buf.WriteString("abc")
	r := NewReader(&buf)
	_, err := r.ReadSentence()
	assert.Error(t, err)
}

func TestReadSentence_ErrorWhileSkippingEmpty(t *testing.T) {
	// First word is empty (zero-length), then EOF when trying to read next word
	var buf bytes.Buffer
	buf.WriteByte(0x00) // empty first word
	// No more data — next readWord will hit EOF
	r := NewReader(&buf)
	_, err := r.ReadSentence()
	assert.Error(t, err)
}

func TestReadSentence_SkipEmptySentences(t *testing.T) {
	// Write empty words (zero-length terminators) before the actual sentence
	var buf bytes.Buffer
	buf.WriteByte(0x00) // empty word (causes skip in first-word loop)
	buf.WriteByte(0x00) // another empty word
	buf.Write(buildSentence("!done"))
	r := NewReader(&buf)
	s, err := r.ReadSentence()
	require.NoError(t, err)
	assert.Equal(t, "!done", s.Word)
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
