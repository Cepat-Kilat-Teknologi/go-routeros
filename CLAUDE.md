# go-routeros

Go client library for MikroTik RouterOS. Supports both the REST API (v7+)
and the API Protocol (v6 & v7). Tested on real hardware with full TLS support.

## Stack

Go 1.26+ · stdlib only · no external dependencies

## Packages

| Package | Protocol | Transport | Port | RouterOS |
|---------|----------|-----------|------|----------|
| `rest` | REST API | HTTP/HTTPS | 80/443 | v7 only |
| `api` | API Protocol | TCP/TLS | 8728/8729 | v6 & v7 |

- **RouterOS v6** → use `api` (REST not available)
- **RouterOS v7, simple CRUD** → use `rest`
- **RouterOS v7, advanced** → use `api` (query filtering, future streaming)

Both packages share design patterns (functional options, typed errors, context
support) — switching between them is straightforward.

## Build & Test

```bash
go build ./...
go test ./...
go test -cover ./...
go vet ./...
```

## Project Structure

```
api/              API Protocol client (v6 & v7)
rest/             REST API client (v7 only)
docs/             API documentation (REST_API.md, API_PROTOCOL.md)
examples/         Usage examples
```

## Integration in CKT

Currently imported by `freeradius-api` for CoA kick dispatch. Adoption by
`olt-executor` for MikroTik targets is a v2+ question.

```go
import "github.com/Cepat-Kilat-Teknologi/go-routeros/rest"
```

## Conventions

- Conventional Commits, English: `feat(rest): add batch command support`
- No `Co-Authored-By` trailers
- Library — no env vars, no HTTP server, no `.env` files
- Tested on RouterOS v7.15 (stable) and v6.49.19 (long-term)
