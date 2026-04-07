package api

import "strings"

// RequestOption configures a single API request.
// Pass one or more RequestOption values to Print, Add, Set, Remove, or Run
// to customize the request behavior.
type RequestOption func(*requestOptions)

// requestOptions holds the accumulated options for a single request.
type requestOptions struct {
	proplist []string // properties to include in the response
	query    []string // query words for filtering results
}

// WithProplist limits which properties are returned in the response.
// This improves performance by skipping slow-access properties.
// Sent to the router as a ".proplist=field1,field2" API attribute word.
//
// Example:
//
//	reply, err := client.Print(ctx, "/ip/address",
//	    api.WithProplist("address", "interface", "disabled"),
//	)
func WithProplist(props ...string) RequestOption {
	return func(o *requestOptions) {
		o.proplist = props
	}
}

// WithQuery adds query words for stack-based filtering.
// Each entry is sent as a separate query word to the router.
// Users provide raw API Protocol query syntax including the "?" prefix.
//
// Query operators:
//   - "?=key=value" or "?key=value" — push true if property equals value
//   - "?>key=value" — push true if property > value
//   - "?<key=value" — push true if property < value
//   - "?#|" — pop two, push OR result
//   - "?#!" — pop one, push NOT result
//   - "?#&" — pop two, push AND result
//
// Example:
//
//	// Find ether OR vlan interfaces
//	reply, err := client.Print(ctx, "/interface",
//	    api.WithQuery("?type=ether", "?type=vlan", "?#|"),
//	)
func WithQuery(query ...string) RequestOption {
	return func(o *requestOptions) {
		o.query = query
	}
}

// collectRequestOptions applies all RequestOption functions to a new
// requestOptions struct and returns the result.
func collectRequestOptions(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// proplistWord builds the ".proplist=field1,field2" API attribute word
// from a slice of property names.
func proplistWord(props []string) string {
	return ".proplist=" + strings.Join(props, ",")
}
