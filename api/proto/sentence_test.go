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

func TestParseWord_EqualPrefixNoValue(t *testing.T) {
	// "=key" without a second "=" should return (key, "")
	k, v := ParseWord("=key")
	assert.Equal(t, "key", k)
	assert.Equal(t, "", v)
}

func TestParseWord_DotPrefixNoEquals(t *testing.T) {
	// ".proplist" without "=" should return (".proplist", "")
	k, v := ParseWord(".proplist")
	assert.Equal(t, ".proplist", k)
	assert.Equal(t, "", v)
}
