package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithProplist(t *testing.T) {
	opts := collectRequestOptions(WithProplist("address", "interface", "disabled"))
	assert.Equal(t, []string{"address", "interface", "disabled"}, opts.proplist)
}

func TestWithQuery(t *testing.T) {
	opts := collectRequestOptions(WithQuery("type=ether", "type=vlan", "#|"))
	assert.Equal(t, []string{"type=ether", "type=vlan", "#|"}, opts.query)
}

func TestWithFilter(t *testing.T) {
	f := map[string]string{"network": "10.0.0.0", "dynamic": "true"}
	opts := collectRequestOptions(WithFilter(f))
	assert.Equal(t, f, opts.filter)
}

func TestCollectRequestOptions_Empty(t *testing.T) {
	opts := collectRequestOptions()
	assert.Nil(t, opts.proplist)
	assert.Nil(t, opts.query)
	assert.Nil(t, opts.filter)
}

func TestCollectRequestOptions_Combined(t *testing.T) {
	opts := collectRequestOptions(
		WithProplist("name", "type"),
		WithQuery("type=ether"),
		WithFilter(map[string]string{"dynamic": "true"}),
	)
	assert.Equal(t, []string{"name", "type"}, opts.proplist)
	assert.Equal(t, []string{"type=ether"}, opts.query)
	assert.Equal(t, map[string]string{"dynamic": "true"}, opts.filter)
}

func TestBuildURLQuery_ProplistAndFilter(t *testing.T) {
	opts := &requestOptions{
		proplist: []string{"address", "interface"},
		filter:   map[string]string{"dynamic": "true"},
	}
	q := buildURLQuery(opts)
	assert.Equal(t, "address,interface", q.Get(".proplist"))
	assert.Equal(t, "true", q.Get("dynamic"))
}

func TestBuildURLQuery_Empty(t *testing.T) {
	opts := &requestOptions{}
	q := buildURLQuery(opts)
	assert.Equal(t, "", q.Encode())
}

func TestMergePayloadWithOptions_NewPayload(t *testing.T) {
	opts := &requestOptions{
		proplist: []string{"name", "type"},
		query:    []string{"type=ether", "#!"},
	}
	result, err := mergePayloadWithOptions(nil, opts)
	assert.NoError(t, err)
	assert.Contains(t, string(result), `".proplist"`)
	assert.Contains(t, string(result), `".query"`)
}

func TestMergePayloadWithOptions_ExistingPayload(t *testing.T) {
	existing := []byte(`{"address":"10.0.0.1/24"}`)
	opts := &requestOptions{
		proplist: []string{"address"},
	}
	result, err := mergePayloadWithOptions(existing, opts)
	assert.NoError(t, err)
	assert.Contains(t, string(result), `"address"`)
	assert.Contains(t, string(result), `".proplist"`)
}

func TestMergePayloadWithOptions_InvalidJSON(t *testing.T) {
	opts := &requestOptions{
		proplist: []string{"name"},
	}
	_, err := mergePayloadWithOptions([]byte("not-valid-json"), opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse existing payload")
}

func TestMergePayloadWithOptions_NoOptions(t *testing.T) {
	existing := []byte(`{"address":"10.0.0.1/24"}`)
	opts := &requestOptions{}
	result, err := mergePayloadWithOptions(existing, opts)
	assert.NoError(t, err)
	assert.Equal(t, existing, result)
}
