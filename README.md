# go-routeros

[![Go Reference](https://pkg.go.dev/badge/github.com/Cepat-Kilat-Teknologi/go-routeros.svg)](https://pkg.go.dev/github.com/Cepat-Kilat-Teknologi/go-routeros)
[![Test](https://github.com/Cepat-Kilat-Teknologi/go-routeros/actions/workflows/test.yml/badge.svg)](https://github.com/Cepat-Kilat-Teknologi/go-routeros/actions)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/Cepat-Kilat-Teknologi/go-routeros)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cepat-Kilat-Teknologi/go-routeros)](https://goreportcard.com/report/github.com/Cepat-Kilat-Teknologi/go-routeros)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go client library for [MikroTik RouterOS](https://mikrotik.com/). Supports both the **REST API** (RouterOS v7+) and the **API Protocol** (RouterOS v6 & v7).

## Table of Contents

- [Install](#install)
- [Which Package to Use?](#which-package-to-use)
- [REST API (v7)](#rest-api-v7)
  - [Quick Start](#rest-quick-start)
  - [Client Options](#rest-client-options)
  - [Retrieving Data (Print)](#retrieving-data-print)
  - [Filtering with Proplist](#filtering-with-proplist)
  - [Filtering with URL Parameters](#filtering-with-url-parameters)
  - [Complex Queries (POST)](#complex-queries-post)
  - [Creating Records (Add)](#creating-records-add)
  - [Updating Records (Set)](#updating-records-set)
  - [Deleting Records (Remove)](#deleting-records-remove)
  - [Running Commands (Run)](#running-commands-run)
  - [Error Handling](#rest-error-handling)
  - [Methods Reference](#rest-methods-reference)
- [API Protocol (v6 & v7)](#api-protocol-v6--v7)
  - [Quick Start](#api-quick-start)
  - [Client Options](#api-client-options)
  - [Retrieving Data (Print)](#api-retrieving-data-print)
  - [Filtering with Proplist](#api-filtering-with-proplist)
  - [Filtering with Query](#api-filtering-with-query)
  - [Creating Records (Add)](#api-creating-records-add)
  - [Updating Records (Set)](#api-updating-records-set)
  - [Deleting Records (Remove)](#api-deleting-records-remove)
  - [Running Commands (Run)](#api-running-commands-run)
  - [Error Handling](#api-error-handling)
  - [Working with Replies](#working-with-replies)
  - [Methods Reference](#api-methods-reference)
  - [Authentication](#authentication)
- [Migration from routerosv7-restfull-api](#migration-from-routerosv7-restfull-api)
- [Project Structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

## Install

```bash
go get github.com/Cepat-Kilat-Teknologi/go-routeros
```

Requires Go 1.21 or later.

## Which Package to Use?

This library provides two packages for two different protocols:

| Package | Protocol | Transport | Port | RouterOS Version |
|---|---|---|---|---|
| `rest` | [REST API](https://help.mikrotik.com/docs/space/ROS/2555940/REST+API) | HTTP/HTTPS | 80/443 | v7 only |
| `api` | [API Protocol](https://help.mikrotik.com/docs/space/ROS/2555940/API) | TCP/TLS | 8728/8729 | v6 & v7 |

**Decision guide:**

- **RouterOS v6** &rarr; Use `api` (REST API is not available in v6)
- **RouterOS v7, simple CRUD** &rarr; Use `rest` (simpler HTTP-based approach)
- **RouterOS v7, advanced features** &rarr; Use `api` (more powerful query filtering, future streaming/listen support)

Both packages follow the same design patterns (functional options, typed errors, context support), so switching between them is straightforward.

---

## REST API (v7)

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
```

The `rest` package communicates with RouterOS v7 devices over HTTP/HTTPS using the REST API.

### REST Quick Start

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

### REST Client Options

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

### Retrieving Data (Print)

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

### Filtering with Proplist

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

### Filtering with URL Parameters

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

### Complex Queries (POST)

For complex filtering with logical operators, use `WithQuery` via the `Run` method (POST):

```go
// Find interfaces that are ether OR vlan type
result, err := client.Run(ctx, "interface/print", nil,
    rest.WithProplist("name", "type"),
    rest.WithQuery("type=ether", "type=vlan", "#|"),
)
```

Query operators:
- `#|` &mdash; OR (pop two values, push result)
- `#!` &mdash; NOT (pop one value, push result)
- `#&` &mdash; AND (implicit, pop two values, push result)

### Creating Records (Add)

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

### Updating Records (Set)

```go
payload, _ := json.Marshal(map[string]string{
    "comment": "Updated via API",
})

result, err := client.Set(ctx, "ip/address/*1", payload)
```

### Deleting Records (Remove)

```go
_, err := client.Remove(ctx, "ip/address/*1")
```

### Running Commands (Run)

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

### REST Error Handling

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

### REST Methods Reference

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

---

## API Protocol (v6 & v7)

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/api"
```

The `api` package communicates with RouterOS devices using the binary TCP-based API Protocol. Works with **all RouterOS versions** (v6 and v7).

### API Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
    // Connect and authenticate (auto-detects auth method)
    client, err := api.Dial("192.168.88.1", "admin", "")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Verify connection
    reply, err := client.Auth(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Platform:", reply.Re[0].Map["platform"])

    // List all IP addresses
    reply, err = client.Print(ctx, "/ip/address",
        api.WithProplist("address", "interface"),
    )
    if err != nil {
        log.Fatal(err)
    }

    for _, re := range reply.Re {
        fmt.Printf("  %s on %s\n", re.Map["address"], re.Map["interface"])
    }
}
```

### API Client Options

Configure when calling `api.Dial()`:

```go
// Plain TCP (default, port 8728)
client, err := api.Dial("192.168.88.1", "admin", "password")

// With TLS (port 8729)
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLS(true),
)

// With custom TLS config (e.g., skip certificate verification)
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLSConfig(&tls.Config{
        InsecureSkipVerify: true,
    }),
)

// With timeout
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTimeout(10 * time.Second),
)

// Custom port
client, err := api.Dial("192.168.88.1:9000", "admin", "password")
```

| Option | Description | Default |
|---|---|---|
| `WithTLS(bool)` | Enable TLS connection (port 8729) | `false` |
| `WithTLSConfig(*tls.Config)` | Custom TLS configuration (implies TLS) | `nil` |
| `WithTimeout(time.Duration)` | Connection timeout | No timeout |

**Default ports:**
- Plain TCP: `8728`
- TLS: `8729`
- Custom port: specify in address (`"host:port"`)

### API Retrieving Data (Print)

```go
// List all IP addresses
reply, err := client.Print(ctx, "/ip/address")

// List all interfaces
reply, err := client.Print(ctx, "/interface")

// List firewall rules
reply, err := client.Print(ctx, "/ip/firewall/filter")
```

**Note:** API Protocol commands use `/` prefix (e.g., `/ip/address`), while REST uses no prefix (e.g., `ip/address`).

### API Filtering with Proplist

Limit returned properties for better performance:

```go
reply, err := client.Print(ctx, "/ip/address",
    api.WithProplist("address", "interface", "disabled"),
)

for _, re := range reply.Re {
    fmt.Printf("Address: %s, Interface: %s, Disabled: %s\n",
        re.Map["address"], re.Map["interface"], re.Map["disabled"])
}
```

### API Filtering with Query

The API Protocol supports powerful stack-based query filtering:

```go
// Simple equals filter
reply, err := client.Print(ctx, "/interface",
    api.WithQuery("?=type=ether"),
)

// OR logic: type=ether OR type=vlan
reply, err := client.Print(ctx, "/interface",
    api.WithProplist("name", "type"),
    api.WithQuery("?type=ether", "?type=vlan", "?#|"),
)

// NOT logic: NOT disabled
reply, err := client.Print(ctx, "/interface",
    api.WithQuery("?disabled=true", "?#!"),
)

// Greater than: routes with non-empty comment
reply, err := client.Print(ctx, "/ip/route",
    api.WithQuery("?>comment="),
)

// Combined: (type=ether OR type=vlan) AND NOT disabled
reply, err := client.Print(ctx, "/interface",
    api.WithQuery(
        "?type=ether",
        "?type=vlan",
        "?#|",
        "?disabled=true",
        "?#!",
        "?#&",
    ),
)
```

**Query operators:**

| Syntax | Description |
|---|---|
| `?=key=value` | Push true if property equals value |
| `?key=value` | Same as above (shorthand) |
| `?>key=value` | Push true if property > value |
| `?<key=value` | Push true if property < value |
| `?#\|` | Pop two, push OR result |
| `?#!` | Pop one, push NOT result |
| `?#&` | Pop two, push AND result |

### API Creating Records (Add)

```go
reply, err := client.Add(ctx, "/ip/address", map[string]string{
    "address":   "10.0.0.1/24",
    "interface": "ether1",
    "comment":   "Management network",
})
if err != nil {
    log.Fatal(err)
}

// Get the ID of the created record
id, _ := reply.Done.Get("ret")
fmt.Println("Created record ID:", id)
```

### API Updating Records (Set)

```go
_, err := client.Set(ctx, "/ip/address", map[string]string{
    ".id":     "*1",
    "comment": "Updated via API",
    "disabled": "yes",
})
```

### API Deleting Records (Remove)

```go
_, err := client.Remove(ctx, "/ip/address", "*1")
```

### API Running Commands (Run)

Execute any RouterOS command:

```go
// Reboot the device
_, err := client.Run(ctx, "/system/reboot", nil)

// Run a script
_, err = client.Run(ctx, "/system/script/run", map[string]string{
    ".id": "*1",
})

// Get system resource info
reply, err := client.Run(ctx, "/system/resource/print", nil)
if err != nil {
    log.Fatal(err)
}
for _, re := range reply.Re {
    fmt.Printf("Uptime: %s, CPU: %s%%\n",
        re.Map["uptime"], re.Map["cpu-load"])
}
```

### API Error Handling

The API Protocol has two error types:

**`*api.DeviceError`** &mdash; returned when RouterOS sends a `!trap` response:

```go
reply, err := client.Print(ctx, "/ip/address")
if err != nil {
    if de, ok := err.(*api.DeviceError); ok {
        fmt.Printf("Category: %d\n", de.Category)  // 0-7
        fmt.Printf("Message: %s\n", de.Message)
    }
    log.Fatal(err)
}
```

Trap categories:

| Category | Meaning |
|---|---|
| 0 | Missing item or command |
| 1 | Argument value failure |
| 2 | Command interrupted |
| 3 | Scripting failure |
| 4 | General failure |
| 5 | API-related failure |
| 6 | TTY-related failure |
| 7 | Value from `:return` |

**`*api.FatalError`** &mdash; returned when RouterOS sends a `!fatal` response. The connection is closed by the router:

```go
if fe, ok := err.(*api.FatalError); ok {
    fmt.Printf("Fatal: %s\n", fe.Message)
    // Connection is now closed, must reconnect
}
```

### Working with Replies

The `api` package returns structured `*api.Reply` objects:

```go
reply, err := client.Print(ctx, "/ip/address")
if err != nil {
    log.Fatal(err)
}

// reply.Re  — slice of data sentences ([]*proto.Sentence)
// reply.Done — the completion sentence (*proto.Sentence)

// Iterate over results
for _, re := range reply.Re {
    // Access properties by key
    address := re.Map["address"]
    iface := re.Map["interface"]
    id := re.Map[".id"]

    fmt.Printf("[%s] %s on %s\n", id, address, iface)
}

// Check if a property exists
if val, ok := reply.Re[0].Get("comment"); ok {
    fmt.Println("Comment:", val)
}

// Get return value from !done sentence (e.g., after Add)
if ret, ok := reply.Done.Get("ret"); ok {
    fmt.Println("Created ID:", ret)
}

// Check number of results
fmt.Printf("Found %d records\n", len(reply.Re))
```

### API Methods Reference

| Method | Signature | Command Sent | Description |
|---|---|---|---|
| `Auth` | `Auth(ctx) (*Reply, error)` | `/system/resource/print` | Verify connection |
| `Print` | `Print(ctx, command, ...RequestOption) (*Reply, error)` | `{command}/print` | Retrieve records |
| `Add` | `Add(ctx, command, params, ...RequestOption) (*Reply, error)` | `{command}/add` | Create a new record |
| `Set` | `Set(ctx, command, params, ...RequestOption) (*Reply, error)` | `{command}/set` | Update a record |
| `Remove` | `Remove(ctx, command, id, ...RequestOption) (*Reply, error)` | `{command}/remove` | Delete a record |
| `Run` | `Run(ctx, command, params, ...RequestOption) (*Reply, error)` | `{command}` | Execute any command |

**Request Options:**

| Option | Description |
|---|---|
| `WithProplist("a", "b")` | Limit returned properties |
| `WithQuery("?type=ether", "?#!")` | Stack-based query filtering |

### Authentication

`api.Dial` handles authentication automatically. It supports both methods:

- **Post-6.43 (plaintext)** &mdash; Used by all modern RouterOS. Sends username and password directly.
- **Pre-6.43 (MD5 challenge-response)** &mdash; Auto-detected. If the router responds with a challenge, the library computes the MD5 response automatically.

No configuration needed &mdash; just pass username and password to `Dial`.

---

## Migration from routerosv7-restfull-api

If you're migrating from the old `routerosv7-restfull-api` library:

```go
// BEFORE (old library — standalone functions, credentials on every call)
import routeros "github.com/sumitroajiprabowo/routerosv7-restfull-api"

result, err := routeros.Print(ctx, host, user, pass, "ip/address")
result, err := routeros.Add(ctx, host, user, pass, "ip/address", payload)
result, err := routeros.Remove(ctx, host, user, pass, "ip/address/*1")

// AFTER (new library — client pattern, credentials once)
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"

client := rest.NewClient(host, user, pass)
result, err := client.Print(ctx, "ip/address")
result, err := client.Add(ctx, "ip/address", payload)
result, err := client.Remove(ctx, "ip/address/*1")
```

**Key improvements:**
- Credentials stored once in the client (not passed on every call)
- `WithInsecureSkipVerify` for self-signed certificates
- `WithProplist` for better performance
- `WithQuery` and `WithFilter` for filtering
- Structured error types (`*rest.APIError`)
- New `api` package for v6 support

---

## Project Structure

```
go-routeros/
├── rest/                  # REST API client (v7, HTTP/HTTPS)
│   ├── client.go          #   Client, NewClient, ClientOption, CRUD methods
│   ├── options.go         #   WithProplist, WithQuery, WithFilter
│   ├── errors.go          #   APIError
│   ├── request.go         #   HTTP request internals
│   ├── protocol.go        #   Protocol detection, TLS retry
│   └── constant.go        #   Constants
│
├── api/                   # API Protocol client (v6 & v7, TCP)
│   ├── client.go          #   Client, Dial, Close, login, CRUD methods
│   ├── options.go         #   WithProplist, WithQuery
│   ├── errors.go          #   DeviceError, FatalError
│   ├── reply.go           #   Reply struct
│   ├── constant.go        #   Constants
│   └── proto/             #   Wire protocol (binary encoding)
│       ├── sentence.go    #     Sentence, Pair, ParseWord
│       ├── reader.go      #     Length-prefix decoding, sentence reading
│       └── writer.go      #     Length-prefix encoding, sentence writing
│
├── example/
│   ├── rest/              #   REST API examples
│   │   ├── basic/         #     Auth, Print, Add, Remove
│   │   ├── proplist/      #     Proplist filtering
│   │   ├── query/         #     Complex query filtering
│   │   └── filter/        #     URL parameter filtering
│   └── api/               #   API Protocol examples
│       ├── basic/         #     Dial, Auth, Print, Add, Remove
│       └── query/         #     Query filtering
│
├── go.mod
├── LICENSE                # MIT
└── README.md
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests (target: 100% coverage)
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

Run tests:

```bash
go test ./... -cover
go vet ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
