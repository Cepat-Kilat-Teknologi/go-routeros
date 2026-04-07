# Migration from routerosv7-restfull-api

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

## Key Improvements

- Credentials stored once in the client (not passed on every call)
- `WithInsecureSkipVerify` for self-signed certificates
- `WithProplist` for better performance
- `WithQuery` and `WithFilter` for filtering
- Structured error types (`*rest.APIError`)
- New `api` package for v6 support
- TLS support for both API Protocol and REST API
- Context support for cancellation and timeouts
- 100% test coverage
