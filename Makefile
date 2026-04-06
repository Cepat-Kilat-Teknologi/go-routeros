.PHONY: test cover lint vet build check clean

# Run all tests
test:
	go test ./... -v -race -count=1

# Run tests with coverage report
cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Run go vet
vet:
	go vet ./...

# Build all packages
build:
	go build ./...

# Run all checks (test + vet + lint)
check: vet lint test

# Clean generated files
clean:
	rm -f coverage.out
