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
	if strings.HasPrefix(word, "=") {
		word = word[1:]
		idx := strings.Index(word, "=")
		if idx >= 0 {
			return word[:idx], word[idx+1:]
		}
		return word, ""
	}
	if strings.HasPrefix(word, ".") {
		idx := strings.Index(word, "=")
		if idx >= 0 {
			return word[:idx], word[idx+1:]
		}
		return word, ""
	}
	return word, ""
}
