.PHONY: build test clean fmt lint

# Build the main executable
build:
	go build -o sortd ./cmd/sortd

# Run tests (excluding deprecated TUI tests)
test:
	./run_tests.sh

# Format code
fmt:
	go fmt ./cmd/... ./internal/config ./internal/gui ./internal/organize ./internal/watch ./pkg/...

# Run linter
lint:
	go vet ./cmd/... ./internal/config ./internal/gui ./internal/organize ./internal/watch ./pkg/...

# Clean build artifacts
clean:
	rm -f sortd
	rm -f sortd-cli

# All-in-one command for quick verification
verify: fmt lint test build
	@echo "All verification steps passed!"