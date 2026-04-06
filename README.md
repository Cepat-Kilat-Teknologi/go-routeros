# go-routeros

Go client library for [MikroTik RouterOS REST API](https://help.mikrotik.com/docs/space/ROS/2555940/REST+API) (RouterOS v7+).

## Install

```bash
go get github.com/Cepat-Kilat-Teknologi/go-routeros
```

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
    client := rest.NewClient("192.168.88.1", "admin", "",
        rest.WithInsecureSkipVerify(true),
    )

    ctx := context.Background()

    // Authenticate
    info, err := client.Auth(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Connected:", info)

    // List IP addresses
    result, err := client.Print(ctx, "ip/address")
    if err != nil {
        log.Fatal(err)
    }
    data, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(data))
}
```

## Features

- **Client pattern** with reusable credentials and configuration
- **Functional options** for flexible configuration
- **TLS support** with `WithInsecureSkipVerify` for self-signed certificates
- **Structured errors** — type-assert to `*rest.APIError` for status code and detail
- **`.proplist`** — limit returned properties for better performance
- **`.query`** — complex filtering with logical operators (OR, NOT, AND)
- **URL filter** — simple key-value filtering on GET requests
- **Context support** for cancellation and timeouts
- **Automatic protocol detection** — HTTPS first, fallback to HTTP

## Client Options

| Option | Description | Default |
|---|---|---|
| `WithInsecureSkipVerify(true)` | Skip TLS certificate verification | `false` |
| `WithTimeout(30 * time.Second)` | HTTP client timeout | No timeout |
| `WithHTTPClient(client)` | Use custom `*http.Client` | Auto-created |

## Request Options

| Option | Applies to | Description |
|---|---|---|
| `WithProplist("a", "b")` | GET, POST | Limit returned properties |
| `WithQuery("k=v", "#\|")` | POST | Complex stack-based filtering |
| `WithFilter(map)` | GET | URL query parameter filtering |

## Methods

| Method | HTTP | RouterOS | Description |
|---|---|---|---|
| `Auth()` | GET | system/resource | Verify connection |
| `Print()` | GET | print | Retrieve records |
| `Add()` | PUT | add | Create record |
| `Set()` | PATCH | set | Update record |
| `Remove()` | DELETE | remove | Delete record |
| `Run()` | POST | (any) | Execute command |

## Error Handling

```go
result, err := client.Print(ctx, "ip/address")
if err != nil {
    if apiErr, ok := err.(*rest.APIError); ok {
        fmt.Printf("Code: %d, Message: %s, Detail: %s\n",
            apiErr.StatusCode, apiErr.Message, apiErr.Detail)
    }
    log.Fatal(err)
}
```

## API Protocol (v6 & v7)

For the TCP-based API Protocol (port 8728/8729), use the `api` package:

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/api"

client, err := api.Dial("192.168.88.1", "admin", "")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

reply, err := client.Print(context.Background(), "/ip/address",
    api.WithProplist("address", "interface"),
)
```

### Which package to use?

| Your RouterOS version | Package | Why |
|---|---|---|
| v6 | `api` | REST API not available in v6 |
| v7 (simple CRUD) | `rest` | Simpler HTTP-based API |
| v7 (advanced) | `api` | More powerful filtering, future streaming support |

### API Client Options

| Option | Description | Default |
|---|---|---|
| `WithTLS(true)` | Enable TLS (port 8729) | `false` |
| `WithTLSConfig(cfg)` | Custom TLS config | `nil` |
| `WithTimeout(d)` | Connection timeout | No timeout |

### API Error Handling

```go
reply, err := client.Print(ctx, "/ip/address")
if err != nil {
    if de, ok := err.(*api.DeviceError); ok {
        fmt.Printf("Trap category %d: %s\n", de.Category, de.Message)
    }
    if fe, ok := err.(*api.FatalError); ok {
        fmt.Printf("Fatal: %s (connection closed)\n", fe.Message)
    }
}
```

## Migration from routerosv7-restfull-api

```go
// Before (old library)
import routeros "github.com/sumitroajiprabowo/routerosv7-restfull-api"
result, err := routeros.Print(ctx, host, user, pass, "ip/address")

// After (new library)
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
client := rest.NewClient(host, user, pass)
result, err := client.Print(ctx, "ip/address")
```

## License

MIT
