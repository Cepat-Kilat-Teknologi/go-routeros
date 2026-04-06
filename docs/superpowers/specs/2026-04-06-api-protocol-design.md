# go-routeros/api: RouterOS API Protocol Package Design

## Overview

Package `api` implements the MikroTik RouterOS API Protocol — a binary TCP-based protocol for communicating with RouterOS devices (port 8728 plain, 8729 TLS). Works with all RouterOS versions (v6 and v7). Complements the existing `rest/` package which covers the v7 HTTP REST API.

## Motivasi

- RouterOS v6 tidak punya REST API — hanya API Protocol yang tersedia
- RouterOS v7 juga masih support API Protocol, dan beberapa fitur (listen/streaming) hanya tersedia via API Protocol
- User butuh satu library yang cover semua versi RouterOS

## Keputusan Arsitektur

### Package Structure

- **Module**: `github.com/Cepat-Kilat-Teknologi/go-routeros` (sama, shared module)
- **Package**: `api` (subpackage untuk API Protocol)
- **Subpackage**: `api/proto` (wire protocol encoding/decoding)
- **Import path**: `github.com/Cepat-Kilat-Teknologi/go-routeros/api`

### Pattern: Dial + Close (Persistent TCP Connection)

Berbeda dari `rest/` yang stateless per-request, `api/` menggunakan persistent TCP connection:
- `Dial()` membuat koneksi dan melakukan login
- `Close()` menutup koneksi
- Semua method beroperasi pada koneksi yang sama
- Sync only (satu command pada satu waktu) — Async dan Listen out of scope

### Scope: Sync Only

- Kirim command, tunggu response sampai `!done`
- Satu command pada satu waktu (no concurrent commands)
- Async (tag-based multiplexing) dan Listen (streaming) ditambah nanti

## Komponen

### 1. Wire Protocol — Subpackage `api/proto/`

Pure wire format encoding/decoding. Tidak tahu tentang login, CRUD, atau business logic.

#### Sentence (proto/sentence.go)

```go
// Pair represents a key-value pair from an API word (e.g., "=address=10.0.0.1").
type Pair struct {
    Key   string
    Value string
}

// Sentence represents a single API sentence (sequence of words).
type Sentence struct {
    Word string            // First word: command or reply type ("/ip/address/print", "!re", "!done", "!trap", "!fatal")
    List []Pair            // Ordered key-value pairs from attribute words
    Map  map[string]string // Same data as List but indexed by key for fast lookup
    Tag  string            // Value of .tag if present
}
```

#### Reader (proto/reader.go)

```go
// Reader reads sentences from a RouterOS API connection.
type Reader struct {
    r *bufio.Reader
}

func NewReader(r io.Reader) *Reader

// ReadSentence reads one complete sentence from the connection.
// Returns the parsed Sentence with all words decoded.
func (r *Reader) ReadSentence() (*Sentence, error)

// Internal:
// readLength() — decode variable-length prefix (1-5 bytes)
// readWord() — read length-prefixed word
```

Length prefix encoding scheme:

| Length Range | Bytes | Encoding |
|---|---|---|
| 0-0x7F | 1 | `len` directly |
| 0x80-0x3FFF | 2 | `len \| 0x8000` |
| 0x4000-0x1FFFFF | 3 | `len \| 0xC00000` |
| 0x200000-0xFFFFFFF | 4 | `len \| 0xE0000000` |
| >= 0x10000000 | 5 | `0xF0` + 4 bytes |

#### Writer (proto/writer.go)

```go
// Writer writes sentences to a RouterOS API connection.
type Writer struct {
    w   *bufio.Writer
    mu  sync.Mutex // protects concurrent writes
}

func NewWriter(w io.Writer) *Writer

// BeginSentence acquires the write lock.
func (w *Writer) BeginSentence() *Writer

// WriteWord writes a single length-prefixed word.
func (w *Writer) WriteWord(word string) *Writer

// EndSentence writes the zero-length terminator, flushes, and releases the lock.
func (w *Writer) EndSentence() error

// Internal:
// encodeLength(l int) []byte — encode length as 1-5 byte prefix
```

### 2. Error Types (api/errors.go)

```go
// DeviceError represents a !trap response from RouterOS.
type DeviceError struct {
    Category int    // Trap category (0-7)
    Message  string // Error message from router
}

func (e *DeviceError) Error() string {
    return fmt.Sprintf("routeros: trap: %s (category %d)", e.Message, e.Category)
}

// FatalError represents a !fatal response from RouterOS.
// The connection is closed by the router after a fatal error.
type FatalError struct {
    Message string
}

func (e *FatalError) Error() string {
    return fmt.Sprintf("routeros: fatal: %s", e.Message)
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

### 3. Constants (api/constant.go)

```go
const (
    DefaultPort    = "8728"
    DefaultTLSPort = "8729"

    replyRe    = "!re"
    replyDone  = "!done"
    replyTrap  = "!trap"
    replyFatal = "!fatal"
)
```

### 4. Request Options (api/options.go)

```go
type RequestOption func(*requestOptions)

type requestOptions struct {
    proplist []string
    query    []string
}

// WithProplist limits returned properties.
// Sent as ".proplist=address,interface" API attribute.
func WithProplist(props ...string) RequestOption

// WithQuery adds query words for filtering.
// Each entry becomes a query word: "?=interface=ether1", "?type=ether", "?#|"
// User provides raw API Protocol query syntax.
func WithQuery(query ...string) RequestOption
```

Query syntax (API Protocol format):
```go
// Equals filter
api.WithQuery("?=interface=ether1")

// OR logic: type=ether OR type=vlan
api.WithQuery("?type=ether", "?type=vlan", "?#|")

// Greater than
api.WithQuery("?>comment=")
```

### 5. Client (api/client.go)

```go
type ClientOption func(*clientConfig)

type clientConfig struct {
    useTLS    bool
    tlsConfig *tls.Config
    timeout   time.Duration
}

type Client struct {
    conn   io.ReadWriteCloser
    reader *proto.Reader
    writer *proto.Writer
}

// Dial connects to a RouterOS device and authenticates.
// Address format: "host:port" or just "host" (default port 8728, or 8729 with TLS).
// Login is performed automatically — supports both pre-6.43 (MD5 challenge)
// and post-6.43 (plaintext) authentication.
func Dial(address, username, password string, opts ...ClientOption) (*Client, error)

// Close closes the TCP connection.
func (c *Client) Close() error
```

Client options:
```go
// WithTLS enables TLS connection (port 8729).
func WithTLS(useTLS bool) ClientOption

// WithTLSConfig sets custom TLS configuration.
func WithTLSConfig(config *tls.Config) ClientOption

// WithTimeout sets connection and read/write timeout.
func WithTimeout(d time.Duration) ClientOption
```

### 6. Reply (api/reply.go)

```go
// Reply holds the complete response to a command.
type Reply struct {
    Re   []*proto.Sentence // Data sentences (!re)
    Done *proto.Sentence   // Completion sentence (!done)
}
```

### 7. Client Methods (api/client.go)

```go
// Auth verifies the connection (already authenticated during Dial).
// Returns system resource info.
func (c *Client) Auth(ctx context.Context) (*Reply, error)

// Print retrieves data (sends /command/print).
func (c *Client) Print(ctx context.Context, command string, opts ...RequestOption) (*Reply, error)

// Add creates a new record (sends /command/add).
func (c *Client) Add(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error)

// Set updates a record (sends /command/set with =.id=).
func (c *Client) Set(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error)

// Remove deletes a record (sends /command/remove with =.id=).
func (c *Client) Remove(ctx context.Context, command string, id string, opts ...RequestOption) (*Reply, error)

// Run executes an arbitrary command with optional parameters.
func (c *Client) Run(ctx context.Context, command string, params map[string]string, opts ...RequestOption) (*Reply, error)
```

Perbedaan dengan `rest/`: payload berupa `map[string]string` bukan `[]byte` JSON, karena API Protocol menggunakan key-value attribute words.

Internal flow setiap method:
1. Build command word: `/{command}/{operation}` (misal `/ip/address/print`)
2. Build attribute words dari params: `=key=value`
3. Add `.proplist` jika ada
4. Add query words jika ada
5. Kirim sentence via Writer
6. Baca sentences via Reader sampai `!done` atau `!fatal`
7. Return `*Reply` atau error

### 8. Authentication — Auto-detect (internal)

```go
// login handles authentication, auto-detecting pre/post-6.43.
func (c *Client) login(username, password string) error
```

Flow:
1. Send `/login` dengan `=name=` dan `=password=`
2. Baca response
3. Jika `!done` tanpa `=ret=` → login sukses (post-6.43)
4. Jika `!done` dengan `=ret=` → pre-6.43, compute MD5 challenge-response:
   - `MD5(0x00 + password + decoded_challenge)`
   - Send `/login` dengan `=name=` dan `=response=00<hex_md5>`
5. Jika `!trap` → return `DeviceError`

## File Structure

```
go-routeros/
├── rest/                    # Sudah ada (v7 REST API)
├── api/
│   ├── client.go            # Client, Dial, Close, ClientOption, CRUD methods, login
│   ├── client_test.go
│   ├── reply.go             # Reply struct
│   ├── reply_test.go
│   ├── options.go           # RequestOption, WithProplist, WithQuery
│   ├── options_test.go
│   ├── errors.go            # DeviceError, FatalError
│   ├── errors_test.go
│   ├── constant.go          # Port and reply word constants
│   ├── proto/
│   │   ├── reader.go        # Word/sentence reading
│   │   ├── reader_test.go
│   │   ├── writer.go        # Word/sentence writing
│   │   ├── writer_test.go
│   │   ├── sentence.go      # Sentence, Pair structs
│   │   └── sentence_test.go
│   └── doc.go               # Package documentation
├── example/
│   ├── rest/                # Sudah ada
│   └── api/
│       ├── basic/main.go    # Connect, auth, print, add, remove
│       └── query/main.go    # Query filtering example
└── README.md                # Update with api/ section
```

## Contoh Penggunaan

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Cepat-Kilat-Teknologi/go-routeros/api"
)

func main() {
    // Connect and login (auto-detect auth method)
    client, err := api.Dial("192.168.88.1:8728", "admin", "")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Print IP addresses with proplist
    reply, err := client.Print(ctx, "/ip/address",
        api.WithProplist("address", "interface"),
    )
    if err != nil {
        if de, ok := err.(*api.DeviceError); ok {
            log.Printf("Trap category %d: %s", de.Category, de.Message)
        }
        log.Fatal(err)
    }

    for _, re := range reply.Re {
        fmt.Printf("Address: %s, Interface: %s\n",
            re.Map["address"], re.Map["interface"])
    }

    // Add IP address
    reply, err = client.Add(ctx, "/ip/address", map[string]string{
        "address":   "10.0.0.1/24",
        "interface": "ether1",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Added ID:", reply.Done.Map["ret"])

    // Remove by ID
    _, err = client.Remove(ctx, "/ip/address", reply.Done.Map["ret"])
    if err != nil {
        log.Fatal(err)
    }
}
```

## Testing Strategy

- **proto/**: Unit test dengan byte buffers — no network needed. Test semua length encoding edge cases (0, 127, 128, 16383, 16384, dll).
- **errors**: Unit test type assertion dan Error() format.
- **options**: Unit test option collection dan word building.
- **client**: Mock TCP connection (io.ReadWriteCloser) untuk test login flow, CRUD methods, error handling. Simulate pre/post-6.43 auth responses.
- **Target: 100% coverage**

## Out of Scope

- Async mode (tag-based multiplexing) — future
- Listen/streaming (`/listen` command) — future
- `/cancel` command — future (depends on async)
- Regex query filtering (not supported by API Protocol)
