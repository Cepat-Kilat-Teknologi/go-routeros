# Contributing to go-routeros

Thank you for your interest in contributing to go-routeros! This document provides guidelines for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/go-routeros.git
   cd go-routeros
   ```
3. Create a branch:
   ```bash
   git checkout -b feature/your-feature
   ```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for convenience commands)

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make cover

# Run linter
make lint

# Run all checks
make check
```

### Code Standards

- **100% test coverage** is required for all packages
- **golint/golangci-lint** must pass with no issues
- **Comments** must be in US English, following godoc conventions
- Every exported type, function, and method must have a doc comment starting with its name
- Follow existing code patterns and naming conventions

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(rest): add support for new feature
fix(api): resolve connection timeout issue
docs: update README with new examples
test: add missing edge case tests
chore: update dependencies
refactor(api/proto): simplify length encoding
```

### Pull Request Process

1. Ensure all tests pass: `make check`
2. Update documentation if needed
3. Add tests for new functionality (100% coverage required)
4. Keep PRs focused — one feature or fix per PR
5. Fill out the PR template

## Project Structure

```
go-routeros/
├── rest/           # REST API client (v7)
├── api/            # API Protocol client (v6 & v7)
│   └── proto/      # Wire protocol encoding
├── example/        # Usage examples
└── .github/        # CI and templates
```

## Integration Testing

Unit tests run against mock servers and achieve 100% coverage. For integration testing against real RouterOS devices:

### Prerequisites

- RouterOS v7 device (for REST API + API Protocol testing)
- RouterOS v6 device (for API Protocol backward-compatibility testing)
- API service enabled on both devices (`/ip service enable api`)

### Running Examples

Update credentials in the example files, then run:

```bash
# API Protocol examples (works on v6 & v7)
go run ./example/api/basic/
go run ./example/api/proplist/
go run ./example/api/query/
go run ./example/api/set/
go run ./example/api/error-handling/

# API Protocol with TLS (requires certificate setup, see README)
go run ./example/api/tls/

# REST API examples (v7 only)
go run ./example/rest/basic/
go run ./example/rest/proplist/
go run ./example/rest/filter/
go run ./example/rest/query/
```

### TLS Testing

To test TLS connections, certificates must be configured on the router. See the [TLS/SSL Certificate Setup](README.md#tlsssl-certificate-setup-routeros) section in README for step-by-step instructions.

### Verified Platforms

| RouterOS | Version | API | API-SSL | REST | REST HTTPS |
|----------|---------|:---:|:-------:|:----:|:----------:|
| v7 | 7.15 (stable) | Yes | Yes | Yes | Yes |
| v6 | 6.49.19 (long-term) | Yes | Yes | N/A | N/A |

## Questions?

Open a [Discussion](https://github.com/Cepat-Kilat-Teknologi/go-routeros/discussions) for questions or ideas.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
