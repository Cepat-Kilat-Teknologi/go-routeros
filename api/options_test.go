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

func TestProplistWord(t *testing.T) {
	result := proplistWord([]string{"address", "interface"})
	assert.Equal(t, ".proplist=address,interface", result)
}
