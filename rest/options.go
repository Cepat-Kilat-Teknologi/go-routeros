package rest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// RequestOption configures a single REST API request.
// Pass one or more RequestOption values to Print, Add, Set, Remove, or Run
// to customize the request behavior.
type RequestOption func(*requestOptions)

// requestOptions holds the accumulated options for a single request.
type requestOptions struct {
	proplist []string          // properties to include in the response
	query    []string          // query stack for complex filtering (POST only)
	filter   map[string]string // key-value filters for URL parameters (GET only)
}

// WithProplist limits which properties are returned in the response.
// This improves performance by skipping slow-access properties.
//
// For GET requests, sent as URL query parameter: ?.proplist=address,interface
// For POST requests, merged into the JSON payload body.
//
// Example:
//
//	result, err := client.Print(ctx, "ip/address",
//	    rest.WithProplist("address", "interface", "disabled"),
//	)
func WithProplist(props ...string) RequestOption {
	return func(o *requestOptions) {
		o.proplist = props
	}
}

// WithQuery sets the query stack for complex filtering via POST requests.
// Query uses a stack-based logic system with operators:
//   - "#|" — OR (pop two values, push result)
//   - "#!" — NOT (pop one value, push result)
//   - "#&" — AND (pop two values, push result)
//
// Example:
//
//	// Find ether OR vlan interfaces
//	result, err := client.Run(ctx, "interface/print", nil,
//	    rest.WithQuery("type=ether", "type=vlan", "#|"),
//	)
func WithQuery(query ...string) RequestOption {
	return func(o *requestOptions) {
		o.query = query
	}
}

// WithFilter sets simple key-value URL query parameter filters for GET requests.
// Multiple filters act as AND conditions.
//
// Example:
//
//	result, err := client.Print(ctx, "ip/address",
//	    rest.WithFilter(map[string]string{"dynamic": "true"}),
//	    rest.WithProplist("address", "interface"),
//	)
func WithFilter(filter map[string]string) RequestOption {
	return func(o *requestOptions) {
		o.filter = filter
	}
}

// collectRequestOptions applies all RequestOption functions to a new
// requestOptions struct and returns the accumulated result.
func collectRequestOptions(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// buildURLQuery builds URL query parameters from requestOptions.
// Used for GET requests to encode proplist and filter values.
// Returns url.Values ready for encoding into a query string.
func buildURLQuery(opts *requestOptions) url.Values {
	q := url.Values{}
	// Add .proplist parameter if specified.
	if len(opts.proplist) > 0 {
		q.Set(".proplist", strings.Join(opts.proplist, ","))
	}
	// Add each filter as a separate URL query parameter.
	for k, v := range opts.filter {
		q.Set(k, v)
	}
	return q
}

// mergePayloadWithOptions merges .proplist and .query into a JSON payload.
// Used for POST requests where options must be sent in the request body.
//
// If both payload and options are empty, returns the original payload unchanged.
// If the payload contains existing JSON, the options are merged into it.
// If the payload is nil/empty, a new JSON object is created with just the options.
func mergePayloadWithOptions(payload []byte, opts *requestOptions) ([]byte, error) {
	// No options to merge — return payload as-is.
	if len(opts.proplist) == 0 && len(opts.query) == 0 {
		return payload, nil
	}

	var data map[string]interface{}

	// Parse existing payload or create a new map.
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse existing payload: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	// Merge proplist and query into the payload.
	if len(opts.proplist) > 0 {
		data[".proplist"] = opts.proplist
	}
	if len(opts.query) > 0 {
		data[".query"] = opts.query
	}

	return json.Marshal(data)
}
