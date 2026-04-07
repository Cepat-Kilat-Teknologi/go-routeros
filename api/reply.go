package api

import "github.com/Cepat-Kilat-Teknologi/go-routeros/api/proto"

// Reply holds the complete response to a RouterOS API command.
//
// A typical response consists of zero or more !re (data) sentences
// followed by one !done (completion) sentence:
//
//	reply.Re   — slice of data sentences, each containing key-value pairs
//	reply.Done — the completion sentence, may contain return values (e.g., "ret" after Add)
//
// Example:
//
//	reply, err := client.Print(ctx, "/ip/address")
//	for _, re := range reply.Re {
//	    fmt.Println(re.Map["address"], re.Map["interface"])
//	}
//	if id, ok := reply.Done.Get("ret"); ok {
//	    fmt.Println("Created ID:", id)
//	}
type Reply struct {
	Re   []*proto.Sentence // data sentences from !re responses
	Done *proto.Sentence   // completion sentence from !done response
}
