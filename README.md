# go-routeros

[![Go Reference](https://pkg.go.dev/badge/github.com/Cepat-Kilat-Teknologi/go-routeros.svg)](https://pkg.go.dev/github.com/Cepat-Kilat-Teknologi/go-routeros)
[![Test](https://github.com/Cepat-Kilat-Teknologi/go-routeros/actions/workflows/test.yml/badge.svg)](https://github.com/Cepat-Kilat-Teknologi/go-routeros/actions)
[![codecov](https://codecov.io/gh/Cepat-Kilat-Teknologi/go-routeros/graph/badge.svg)](https://codecov.io/gh/Cepat-Kilat-Teknologi/go-routeros)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cepat-Kilat-Teknologi/go-routeros)](https://goreportcard.com/report/github.com/Cepat-Kilat-Teknologi/go-routeros)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go client library for [MikroTik RouterOS](https://mikrotik.com/). Supports both the **REST API** (RouterOS v7+) and the **API Protocol** (RouterOS v6 & v7). Tested on real hardware with full TLS support.

## Tested On

| RouterOS | Version | API (8728) | API-SSL (8729) | REST HTTP (80) | REST HTTPS (443) |
|----------|---------|:----------:|:--------------:|:--------------:|:----------------:|
| **v7** | 7.15 (stable) | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| **v6** | 6.49.19 (long-term) | :white_check_mark: | :white_check_mark: | — | — |

## Install

```bash
go get github.com/Cepat-Kilat-Teknologi/go-routeros
```

Requires Go 1.21 or later.

## Which Package to Use?

| Package | Protocol | Transport | Port | RouterOS Version |
|---|---|---|---|---|
| `rest` | [REST API](https://help.mikrotik.com/docs/space/ROS/2555940/REST+API) | HTTP/HTTPS | 80/443 | v7 only |
| `api` | [API Protocol](https://help.mikrotik.com/docs/space/ROS/2555940/API) | TCP/TLS | 8728/8729 | v6 & v7 |

- **RouterOS v6** &rarr; Use `api` (REST API is not available in v6)
- **RouterOS v7, simple CRUD** &rarr; Use `rest` (simpler HTTP-based approach)
- **RouterOS v7, advanced features** &rarr; Use `api` (more powerful query filtering, future streaming/listen support)

Both packages follow the same design patterns (functional options, typed errors, context support), so switching between them is straightforward.

---

## REST API Quick Start (v7)

Full documentation: **[docs/REST_API.md](docs/REST_API.md)**

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"

client := rest.NewClient("192.168.88.1", "admin", "",
    rest.WithInsecureSkipVerify(true),
)
ctx := context.Background()

// Auth
info, err := client.Auth(ctx)

// Print
result, err := client.Print(ctx, "ip/address",
    rest.WithProplist("address", "interface"),
)

// Add
payload, _ := json.Marshal(map[string]string{
    "address": "10.0.0.1/24", "interface": "ether1",
})
result, err := client.Add(ctx, "ip/address", payload)

// Remove
_, err := client.Remove(ctx, "ip/address/*1")
```

**Client Options:** `WithInsecureSkipVerify`, `WithTimeout`, `WithHTTPClient`
**Request Options:** `WithProplist`, `WithQuery`, `WithFilter`
**Error Type:** `*rest.APIError` with `StatusCode`, `Message`, `Detail`

---

## API Protocol Quick Start (v6 & v7)

Full documentation: **[docs/API_PROTOCOL.md](docs/API_PROTOCOL.md)**

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/api"

client, err := api.Dial("192.168.88.1", "admin", "")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
ctx := context.Background()

// Auth
reply, err := client.Auth(ctx)
fmt.Println("Platform:", reply.Re[0].Map["platform"])

// Print with proplist and query
reply, err = client.Print(ctx, "/interface",
    api.WithProplist("name", "type"),
    api.WithQuery("?type=ether", "?type=vlan", "?#|"),
)

// Add
reply, err = client.Add(ctx, "/ip/address", map[string]string{
    "address": "10.0.0.1/24", "interface": "ether1",
})
id, _ := reply.Done.Get("ret")

// Set
_, err = client.Set(ctx, "/ip/address", map[string]string{
    ".id": id, "comment": "Updated via API",
})

// Remove
_, err = client.Remove(ctx, "/ip/address", id)
```

**Client Options:** `WithTLS`, `WithTLSConfig`, `WithTimeout`
**Request Options:** `WithProplist`, `WithQuery`
**Error Types:** `*api.DeviceError` (trap), `*api.FatalError` (fatal)
**Authentication:** Auto-detects post-6.43 (plaintext) and pre-6.43 (MD5 challenge-response)

---

## TLS/SSL Setup

For secure connections via API-SSL (port 8729) or REST HTTPS (port 443), certificates must be configured on the router.

Full guide: **[docs/TLS_SETUP.md](docs/TLS_SETUP.md)**

Quick summary:

```routeros
# Generate CA and server certificate
/certificate add name=local-ca common-name=local-ca key-usage=key-cert-sign,crl-sign key-size=2048 days-valid=3650
/certificate sign local-ca ca-crl-host=192.168.88.1
/certificate add name=server common-name=server key-size=2048 days-valid=3650 subject-alt-name=IP:192.168.88.1
/certificate sign server ca=local-ca

# Assign to services and enable
/ip service set api-ssl certificate=server disabled=no
/ip service set www-ssl certificate=server disabled=no
```

```go
// API Protocol with TLS
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
)

// REST API with HTTPS
client := rest.NewClient("https://192.168.88.1", "admin", "password",
    rest.WithInsecureSkipVerify(true),
)
```

---

## Project Structure

```
go-routeros/
├── rest/                   # REST API client (v7, HTTP/HTTPS)
│   ├── client.go           #   Client, NewClient, CRUD methods
│   ├── options.go          #   WithProplist, WithQuery, WithFilter
│   ├── errors.go           #   APIError
│   ├── request.go          #   HTTP request internals
│   ├── protocol.go         #   Protocol detection, TLS retry
│   └── constant.go         #   Constants
│
├── api/                    # API Protocol client (v6 & v7, TCP)
│   ├── client.go           #   Client, Dial, Close, login, CRUD methods
│   ├── options.go          #   WithProplist, WithQuery
│   ├── errors.go           #   DeviceError, FatalError
│   ├── reply.go            #   Reply struct
│   ├── constant.go         #   Constants
│   └── proto/              #   Wire protocol (binary encoding)
│       ├── sentence.go     #     Sentence, Pair, ParseWord
│       ├── reader.go       #     Length-prefix decoding
│       └── writer.go       #     Length-prefix encoding
│
├── example/
│   ├── rest/               #   REST API examples (v7)
│   │   ├── basic/          #     Auth, Print, Add, Remove
│   │   ├── proplist/       #     Proplist filtering
│   │   ├── query/          #     Complex query filtering (POST)
│   │   └── filter/         #     URL parameter filtering (GET)
│   └── api/                #   API Protocol examples (v6 & v7)
│       ├── basic/          #     Dial, Auth, Print, Add, Remove
│       ├── proplist/       #     Proplist filtering
│       ├── query/          #     Stack-based query filtering
│       ├── set/            #     Full CRUD cycle
│       ├── error-handling/ #     DeviceError and FatalError handling
│       └── tls/            #     TLS connection with certificate
│
├── docs/
│   ├── REST_API.md         #   Full REST API documentation
│   ├── API_PROTOCOL.md     #   Full API Protocol documentation
│   ├── TLS_SETUP.md        #   TLS/SSL certificate setup guide
│   └── MIGRATION.md        #   Migration from routerosv7-restfull-api
│
├── go.mod
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
├── LICENSE                  # MIT
└── README.md
```

## Documentation

| Document | Description |
|---|---|
| [docs/REST_API.md](docs/REST_API.md) | Full REST API reference — client options, CRUD, queries, error handling |
| [docs/API_PROTOCOL.md](docs/API_PROTOCOL.md) | Full API Protocol reference — dial options, queries, replies, auth |
| [docs/TLS_SETUP.md](docs/TLS_SETUP.md) | RouterOS certificate setup guide — CA, server cert, service config |
| [docs/MIGRATION.md](docs/MIGRATION.md) | Migration guide from routerosv7-restfull-api |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines and integration testing |
| [SECURITY.md](SECURITY.md) | Security policy and best practices |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
go test ./... -cover   # Run tests (100% coverage required)
go vet ./...           # Static analysis
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
