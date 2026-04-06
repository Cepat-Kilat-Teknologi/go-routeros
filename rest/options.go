package rest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// RequestOption configures a single API request.
type RequestOption func(*requestOptions)

type requestOptions struct {
	proplist []string
	query    []string
	filter   map[string]string
}

// WithProplist limits which properties are returned in the response.
func WithProplist(props ...string) RequestOption {
	return func(o *requestOptions) {
		o.proplist = props
	}
}

// WithQuery sets the query stack for complex filtering (POST only).
func WithQuery(query ...string) RequestOption {
	return func(o *requestOptions) {
		o.query = query
	}
}

// WithFilter sets URL query parameter filters for GET requests.
func WithFilter(filter map[string]string) RequestOption {
	return func(o *requestOptions) {
		o.filter = filter
	}
}

// collectRequestOptions applies all RequestOption functions and returns the result.
func collectRequestOptions(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// buildURLQuery builds URL query parameters from requestOptions.
func buildURLQuery(opts *requestOptions) url.Values {
	q := url.Values{}
	if len(opts.proplist) > 0 {
		q.Set(".proplist", strings.Join(opts.proplist, ","))
	}
	for k, v := range opts.filter {
		q.Set(k, v)
	}
	return q
}

// mergePayloadWithOptions merges .proplist and .query into a JSON payload.
func mergePayloadWithOptions(payload []byte, opts *requestOptions) ([]byte, error) {
	if len(opts.proplist) == 0 && len(opts.query) == 0 {
		return payload, nil
	}

	var data map[string]interface{}

	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse existing payload: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	if len(opts.proplist) > 0 {
		data[".proplist"] = opts.proplist
	}
	if len(opts.query) > 0 {
		data[".query"] = opts.query
	}

	return json.Marshal(data)
}
