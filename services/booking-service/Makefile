.PHONY: run build test test-integration test-e2e docker-up docker-down tidy generate-proto mock-event

# Run locally
run:
	go run ./cmd/server

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/booking-service ./cmd/server

# Run unit tests (standard library, no external dependencies)
test:
	go test ./internal/... -v -count=1

test-unit: test


test-integration:
	chmod +x scripts/run_tests.sh && ./scripts/run_tests.sh

test-e2e: test-integration

# Generate gRPC stubs
generate-proto:
	chmod +x scripts/gen_proto.sh && ./scripts/gen_proto.sh

# Docker compose up
docker-up:
	docker compose up --build -d

# Docker compose down
docker-down:
	docker compose down -v

# Tidy dependencies
tidy:
	go mod tidy

# Publish a mock event to Kafka for manual testing
mock-event:
	go run scripts/publish_mock_event.go

# Run the standalone gRPC mock server for Profile and Finance services
mock-grpc:
	go run scripts/mock_grpc/main.go

