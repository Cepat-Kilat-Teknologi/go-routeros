# API Protocol Package (v6 & v7)

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/api"
```

The `api` package communicates with RouterOS devices using the binary TCP-based API Protocol. Works with **all RouterOS versions** (v6 and v7).

## Quick Start

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

## Client Options

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

## Retrieving Data (Print)

```go
// List all IP addresses
reply, err := client.Print(ctx, "/ip/address")

// List all interfaces
reply, err := client.Print(ctx, "/interface")

// List firewall rules
reply, err := client.Print(ctx, "/ip/firewall/filter")
```

**Note:** API Protocol commands use `/` prefix (e.g., `/ip/address`), while REST uses no prefix (e.g., `ip/address`).

## Filtering with Proplist

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

## Filtering with Query

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

## Creating Records (Add)

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

## Updating Records (Set)

```go
_, err := client.Set(ctx, "/ip/address", map[string]string{
    ".id":     "*1",
    "comment": "Updated via API",
    "disabled": "yes",
})
```

## Deleting Records (Remove)

```go
_, err := client.Remove(ctx, "/ip/address", "*1")
```

## Running Commands (Run)

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

## Error Handling

The API Protocol has two error types:

**`*api.DeviceError`** — returned when RouterOS sends a `!trap` response:

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

**`*api.FatalError`** — returned when RouterOS sends a `!fatal` response. The connection is closed by the router:

```go
if fe, ok := err.(*api.FatalError); ok {
    fmt.Printf("Fatal: %s\n", fe.Message)
    // Connection is now closed, must reconnect
}
```

## Working with Replies

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

## Methods Reference

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

## Authentication

`api.Dial` handles authentication automatically. It supports both methods:

- **Post-6.43 (plaintext)** — Used by all modern RouterOS. Sends username and password directly.
- **Pre-6.43 (MD5 challenge-response)** — Auto-detected. If the router responds with a challenge, the library computes the MD5 response automatically.

No configuration needed — just pass username and password to `Dial`.
