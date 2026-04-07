# TLS/SSL Certificate Setup (RouterOS)

Both the `api` package (port 8729) and `rest` package (port 443) support TLS-encrypted connections. RouterOS requires a certificate to be configured on the router before TLS services will work.

This guide has been tested on:
- **RouterOS 7.15 (stable)** — CHR x86_64
- **RouterOS 6.49.19 (long-term)** — x86

## Step 1: Generate a Self-Signed CA Certificate

```routeros
/certificate add name=local-ca common-name=local-ca key-usage=key-cert-sign,crl-sign key-size=2048 days-valid=3650
/certificate sign local-ca ca-crl-host=192.168.88.1
```

Replace `192.168.88.1` with your router's management IP. Wait for signing to complete (`KLAT` flags appear in `/certificate print`).

## Step 2: Generate a Server Certificate

```routeros
/certificate add name=server common-name=server key-size=2048 days-valid=3650 subject-alt-name=IP:192.168.88.1
/certificate sign server ca=local-ca
```

Set `subject-alt-name` to match how clients connect. Use `IP:x.x.x.x` for IP access or `DNS:router.example.com` for DNS access. Multiple SANs are supported: `subject-alt-name=IP:192.168.88.1,DNS:router.local`.

## Step 3: Assign Certificate to Services

```routeros
# API-SSL (port 8729) — used by the api package with WithTLS(true)
/ip service set api-ssl certificate=server tls-version=only-1.2

# WWW-SSL (port 443) — used by the rest package with HTTPS
/ip service set www-ssl certificate=server tls-version=only-1.2
```

## Step 4: Enable Services

```routeros
/ip service enable api-ssl
/ip service enable www-ssl
```

Verify with:

```routeros
/ip service print
```

## Step 5: Connect with Go

**API Protocol (TLS):**

```go
// Option 1: Skip certificate verification (self-signed certs)
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLSConfig(&tls.Config{
        InsecureSkipVerify: true,
    }),
)

// Option 2: Simple TLS (uses system CA pool)
client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLS(true),
)
```

**REST API (HTTPS):**

```go
// Self-signed certificate
client := rest.NewClient("https://192.168.88.1", "admin", "password",
    rest.WithInsecureSkipVerify(true),
)
```

## Security Best Practices

**Restrict service access by IP:**

```routeros
/ip service set api-ssl address=192.168.88.0/24
/ip service set www-ssl address=192.168.88.0/24
```

**Disable insecure (plaintext) services after TLS is working:**

```routeros
/ip service disable api
/ip service disable www
```

**Export the CA certificate** for client-side verification (avoids `InsecureSkipVerify`):

```routeros
/certificate export-certificate local-ca export-passphrase=""
```

Download the exported `.crt` file from the router's file system via Winbox or FTP, then load it in your Go client:

```go
caCert, _ := os.ReadFile("local-ca.crt")
caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)

client, err := api.Dial("192.168.88.1", "admin", "password",
    api.WithTLSConfig(&tls.Config{
        RootCAs: caCertPool,
    }),
)
```

**ECDSA alternative** (faster handshake on RouterOS v7):

```routeros
/certificate add name=local-ca common-name=local-ca key-usage=key-cert-sign,crl-sign key-type=ecdsa curve=secp384r1 days-valid=3650
/certificate sign local-ca
/certificate add name=server common-name=server key-type=ecdsa curve=secp384r1 days-valid=3650 subject-alt-name=IP:192.168.88.1
/certificate sign server ca=local-ca
```

## Verify Certificate Status

```routeros
/certificate print detail
```

Expected flags:
- CA certificate: `KLAT` (Key, Local, Authority, Trusted)
- Server certificate: `KLA` or `KL` (Key, Local, Authority)
