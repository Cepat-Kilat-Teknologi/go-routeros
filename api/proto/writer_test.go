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
	assert.Equal(t, byte(6), data[0])
	assert.Equal(t, []byte("/login"), data[1:7])
	assert.Equal(t, byte(0), data[7])
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

	data := buf.Bytes()
	assert.Greater(t, len(data), 3)
	assert.Equal(t, byte(0), data[len(data)-1])
}

func TestWriter_WriteWord_LongWord(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	longWord := strings.Repeat("a", 200)

	w.BeginSentence()
	w.WriteWord(longWord)
	err := w.EndSentence()
	require.NoError(t, err)

	data := buf.Bytes()
	assert.Equal(t, byte(0x80), data[0]&0xC0)
	assert.Equal(t, longWord, string(data[2:202]))
}

func TestWriter_EmptySentence(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	w.BeginSentence()
	err := w.EndSentence()
	require.NoError(t, err)

	data := buf.Bytes()
	assert.Equal(t, []byte{0x00}, data)
}
