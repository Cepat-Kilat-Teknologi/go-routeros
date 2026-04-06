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
