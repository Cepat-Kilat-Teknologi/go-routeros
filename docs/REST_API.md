# REST API Package (v7)

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
```

The `rest` package communicates with RouterOS v7 devices over HTTP/HTTPS using the REST API.

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
)

func main() {
    // Create a client (credentials are stored, reused for every request)
    client := rest.NewClient("192.168.88.1", "admin", "",
        rest.WithInsecureSkipVerify(true), // for self-signed certificates
    )

    ctx := context.Background()

    // Verify connection
    info, err := client.Auth(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Connected:", info)

    // List all IP addresses
    result, err := client.Print(ctx, "ip/address")
    if err != nil {
        log.Fatal(err)
    }

    data, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(data))
}
```

## Client Options

Configure the client when calling `rest.NewClient()`:

```go
client := rest.NewClient(host, username, password,
    rest.WithInsecureSkipVerify(true),        // skip TLS cert verification (self-signed)
    rest.WithTimeout(30 * time.Second),       // HTTP client timeout
    rest.WithHTTPClient(customHTTPClient),    // use your own *http.Client
)
```

| Option | Description | Default |
|---|---|---|
| `WithInsecureSkipVerify(bool)` | Skip TLS certificate verification | `false` (secure) |
| `WithTimeout(time.Duration)` | HTTP client timeout | No timeout |
| `WithHTTPClient(*http.Client)` | Override the entire HTTP client | Auto-created |

## Retrieving Data (Print)

```go
// List all IP addresses
result, err := client.Print(ctx, "ip/address")

// List all interfaces
result, err := client.Print(ctx, "interface")

// Get a single record by ID
result, err := client.Print(ctx, "ip/address/*1")

// Get a single record by name
result, err := client.Print(ctx, "interface/ether1")
```

## Filtering with Proplist

Limit which properties are returned. This **improves performance** by skipping slow-access properties:

```go
result, err := client.Print(ctx, "ip/address",
    rest.WithProplist("address", "interface", "disabled"),
)
```

For GET requests, `.proplist` is sent as a URL query parameter:
```
GET /rest/ip/address?.proplist=address,interface,disabled
```

## Filtering with URL Parameters

Filter records using simple key-value URL parameters (GET only):

```go
result, err := client.Print(ctx, "ip/address",
    rest.WithFilter(map[string]string{
        "network":  "10.155.101.0",
        "dynamic": "true",
    }),
)
```

Multiple filters act as AND conditions. Can be combined with `WithProplist`:

```go
result, err := client.Print(ctx, "ip/address",
    rest.WithFilter(map[string]string{"dynamic": "true"}),
    rest.WithProplist("address", "interface"),
)
```

## Complex Queries (POST)

For complex filtering with logical operators, use `WithQuery` via the `Run` method (POST):

```go
// Find interfaces that are ether OR vlan type
result, err := client.Run(ctx, "interface/print", nil,
    rest.WithProplist("name", "type"),
    rest.WithQuery("type=ether", "type=vlan", "#|"),
)
```

Query operators:
- `#|` — OR (pop two values, push result)
- `#!` — NOT (pop one value, push result)
- `#&` — AND (implicit, pop two values, push result)

## Creating Records (Add)

```go
payload, _ := json.Marshal(map[string]string{
    "address":   "10.0.0.1/24",
    "interface": "ether1",
    "comment":   "Management",
})

result, err := client.Add(ctx, "ip/address", payload)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Created:", result)
```

## Updating Records (Set)

```go
payload, _ := json.Marshal(map[string]string{
    "comment": "Updated via API",
})

result, err := client.Set(ctx, "ip/address/*1", payload)
```

## Deleting Records (Remove)

```go
_, err := client.Remove(ctx, "ip/address/*1")
```

## Running Commands (Run)

Execute any RouterOS command via POST:

```go
// Ping
payload, _ := json.Marshal(map[string]string{
    "address": "8.8.8.8",
    "count":   "4",
})
result, err := client.Run(ctx, "ping", payload)

// Export config
payload, _ = json.Marshal(map[string]string{
    "compact": "",
    "file":    "backup.rsc",
})
result, err = client.Run(ctx, "export", payload)
```

## Error Handling

Errors from RouterOS are returned as `*rest.APIError`, which you can type-assert:

```go
result, err := client.Print(ctx, "ip/address")
if err != nil {
    if apiErr, ok := err.(*rest.APIError); ok {
        fmt.Printf("Status: %d\n", apiErr.StatusCode)   // 404
        fmt.Printf("Message: %s\n", apiErr.Message)     // "Not Found"
        fmt.Printf("Detail: %s\n", apiErr.Detail)       // "no such command or directory"
    }
    log.Fatal(err)
}
```

RouterOS returns structured JSON errors:
```json
{"error": 404, "message": "Not Found", "detail": "no such command or directory"}
```

## Methods Reference

| Method | Signature | HTTP | Description |
|---|---|---|---|
| `Auth` | `Auth(ctx) (interface{}, error)` | GET | Verify connection (calls `/rest/system/resource`) |
| `Print` | `Print(ctx, command, ...RequestOption) (interface{}, error)` | GET | Retrieve records |
| `Add` | `Add(ctx, command, payload, ...RequestOption) (interface{}, error)` | PUT | Create a new record |
| `Set` | `Set(ctx, command, payload, ...RequestOption) (interface{}, error)` | PATCH | Update an existing record |
| `Remove` | `Remove(ctx, command, ...RequestOption) (interface{}, error)` | DELETE | Delete a record |
| `Run` | `Run(ctx, command, payload, ...RequestOption) (interface{}, error)` | POST | Execute any command |

**Request Options:**

| Option | Applies to | Description |
|---|---|---|
| `WithProplist("a", "b")` | GET, POST | Limit returned properties |
| `WithQuery("k=v", "#\|")` | POST only | Complex stack-based filtering |
| `WithFilter(map[string]string{...})` | GET only | URL query parameter filtering |

## Decoding Responses

REST API methods return `interface{}`. Use `rest.Decode()` to convert responses into typed structs:

```go
type IPAddress struct {
    ID        string `json:".id"`
    Address   string `json:"address"`
    Interface string `json:"interface"`
}

result, err := client.Print(ctx, "ip/address")

// Decode into a slice of typed structs
var addresses []IPAddress
err = rest.Decode(result, &addresses)

for _, addr := range addresses {
    fmt.Printf("[%s] %s on %s\n", addr.ID, addr.Address, addr.Interface)
}
```

This eliminates the manual `json.Marshal` / `json.Unmarshal` pattern.
