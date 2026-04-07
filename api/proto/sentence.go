// Package proto implements the MikroTik RouterOS API Protocol wire format.
//
// The wire protocol uses length-prefixed words grouped into sentences.
// Each sentence starts with a command/reply word, followed by attribute
// words (=key=value), and ends with a zero-length terminator.
//
// This package provides Reader and Writer for encoding/decoding sentences,
// and Sentence/Pair types for structured access to parsed data.
package proto

import "strings"

// Pair represents a key-value pair from an API attribute word.
// Attribute words have the format "=key=value" on the wire.
type Pair struct {
	Key   string // attribute name (e.g., "address", ".id")
	Value string // attribute value (e.g., "10.0.0.1/24", "*1")
}

// Sentence represents a single API sentence (a sequence of words).
// The first word indicates the sentence type (reply, command, etc.),
// followed by zero or more attribute words.
type Sentence struct {
	Word string            // first word: reply type ("!re", "!done", "!trap", "!fatal")
	List []Pair            // ordered key-value pairs preserving wire order
	Map  map[string]string // same data indexed by key for quick lookups
	Tag  string            // value of .tag if present (used for async command correlation)
}

// Get returns the value for a key and whether it exists.
// This is a convenience method for accessing Map with existence check.
func (s *Sentence) Get(key string) (string, bool) {
	v, ok := s.Map[key]
	return v, ok
}

// ParseWord parses an API word into a key-value pair.
//
// Word types:
//   - Attribute words "=key=value" → returns (key, value)
//   - Tag words ".tag=value" → returns (".tag", value)
//   - Commands/replies (e.g., "/login", "!done") → returns (word, "")
//   - Query words (e.g., "?type=ether") → returns (word, "")
func ParseWord(word string) (string, string) {
	// Attribute word: starts with "=", format is "=key=value"
	if strings.HasPrefix(word, "=") {
		word = word[1:]                 // strip leading "="
		idx := strings.Index(word, "=") // find the separator between key and value
		if idx >= 0 {
			return word[:idx], word[idx+1:] // return key and value
		}
		return word, "" // "=key" with no value
	}
	// Dot-prefixed word: ".tag=value" or ".proplist=..."
	if strings.HasPrefix(word, ".") {
		idx := strings.Index(word, "=")
		if idx >= 0 {
			return word[:idx], word[idx+1:]
		}
		return word, ""
	}
	// Command, reply, or query word: return as-is with empty value
	return word, ""
}
