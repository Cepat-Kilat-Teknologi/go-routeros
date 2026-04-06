package api

import "github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"

// Reply holds the complete response to a command.
type Reply struct {
	Re   []*proto.Sentence // Data sentences (!re)
	Done *proto.Sentence   // Completion sentence (!done)
}
