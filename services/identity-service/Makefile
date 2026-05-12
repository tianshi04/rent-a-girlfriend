.PHONY: run build test-unit test-integration test-contract test-fuzz test-all docker-up docker-down tidy

# Run locally
run:
	go run ./cmd/server

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/identity-service ./cmd/server

# Run unit tests
test-unit:
	go test ./internal/... -v -count=1

# Run integration tests
test-integration:
	go test ./tests/integration/... -v -count=1

# Run contract tests
test-contract:
	go test ./tests/contract/... -v -count=1

# Run fuzz tests (example for a package)
test-fuzz:
	go test ./internal/domain/... -fuzz=Fuzz -fuzztime=10s

# Run all tests
test-all: test-unit test-integration test-contract

# Docker compose up
docker-up:
	docker compose up --build -d

# Docker compose down
docker-down:
	docker compose down -v

# Tidy dependencies
tidy:
	go mod tidy
