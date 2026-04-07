# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-04-07

### Added

#### TLS/SSL Documentation
- Complete TLS/SSL certificate setup guide in README (CA generation, server cert, service assignment)
- Go code examples for TLS connections (API Protocol and REST API)
- Security best practices: IP restriction, disabling plaintext services, CA export for client trust
- ECDSA certificate alternative for RouterOS v7

#### Integration Testing
- Verified all examples against real RouterOS devices:
  - **RouterOS 7.15 (stable)** — CHR x86_64: API, API-SSL, REST HTTP, REST HTTPS
  - **RouterOS 6.49.19 (long-term)** — x86: API, API-SSL
- Confirmed identical output format between v6 and v7

### Fixed
- **REST client**: handle HTTP 204 No Content on DELETE — RouterOS returns empty body on successful delete, which caused `EOF` error in `decodeJSONBody`

### Changed
- Bump `actions/checkout` from v4 to v6
- Bump `actions/setup-go` from v5 to v6
- Bump `github.com/stretchr/testify` from v1.8.4 to v1.11.1

---

## [0.1.0] - 2026-04-07

### Added

#### REST API Package (`rest/`)
- `Client` with `NewClient()` and functional options (`WithInsecureSkipVerify`, `WithTimeout`, `WithHTTPClient`)
- CRUD methods: `Auth`, `Print`, `Add`, `Set`, `Remove`, `Run`
- Request options: `WithProplist`, `WithQuery`, `WithFilter`
- Structured error type `APIError` with `StatusCode`, `Message`, `Detail`
- Automatic HTTPS/HTTP protocol detection with TLS fallback
- Context support for cancellation and timeouts

#### API Protocol Package (`api/`)
- `Client` with `Dial()` and functional options (`WithTLS`, `WithTLSConfig`, `WithTimeout`)
- CRUD methods: `Auth`, `Print`, `Add`, `Set`, `Remove`, `Run`
- Request options: `WithProplist`, `WithQuery`
- Structured error types: `DeviceError` (trap) and `FatalError`
- Auto-detect authentication (pre/post-6.43 RouterOS)
- MD5 challenge-response login for legacy RouterOS
- Context deadline support for TCP operations
- `!empty` reply handling (RouterOS 7.18+)

#### Wire Protocol (`api/proto/`)
- Binary length-prefix encoding/decoding (1-5 byte variable length)
- `Sentence`, `Pair` structs for structured data
- `Reader` and `Writer` for wire format I/O
- `ParseWord` for API word parsing

#### Documentation
- Comprehensive README with both packages documented
- Usage examples for REST and API Protocol
- Godoc example functions for pkg.go.dev
- Thread safety documentation

#### CI/CD
- GitHub Actions workflow (Go 1.21/1.22/1.23, Ubuntu/macOS)
- 100% test coverage across all packages
