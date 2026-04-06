package api

import "strings"

// RequestOption configures a single API request.
type RequestOption func(*requestOptions)

type requestOptions struct {
	proplist []string
	query    []string
}

// WithProplist limits which properties are returned.
// Sent as ".proplist=address,interface" API attribute word.
func WithProplist(props ...string) RequestOption {
	return func(o *requestOptions) {
		o.proplist = props
	}
}

// WithQuery adds query words for filtering.
// Each entry is sent as a separate query word.
// User provides raw API Protocol query syntax including the ? prefix.
func WithQuery(query ...string) RequestOption {
	return func(o *requestOptions) {
		o.query = query
	}
}

// collectRequestOptions applies all options and returns the result.
func collectRequestOptions(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// proplistWord builds the .proplist API attribute word.
func proplistWord(props []string) string {
	return ".proplist=" + strings.Join(props, ",")
}
