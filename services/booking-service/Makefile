.PHONY: run build test-unit test-contract test-e2e test-integration test-all docker-up docker-down tidy generate-proto mock-event mock-grpc

# Run locally
run:
	go run ./cmd/server

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/booking-service ./cmd/server

# Run unit tests (standard library, no external dependencies)
test-unit:
	go test ./internal/... -v -count=1 -short

# Run contract tests
test-contract:
	go test ./tests/contract/... -v -count=1

# Run E2E tests in Docker containers
test-e2e:
	@if [ -z "$$BOOKING_IMAGE" ]; then \
		echo "==> [test-e2e] Building local image booking-service-test:local..."; \
		docker build --build-arg BUILD_TAGS="" --target test -t booking-service-test:local .; \
	fi
	BOOKING_IMAGE=$${BOOKING_IMAGE:-booking-service-test:local} \
		docker compose -f docker-compose.test.yml up -d --wait
	BOOKING_IMAGE=$${BOOKING_IMAGE:-booking-service-test:local} \
		docker compose -f docker-compose.test.yml run --rm e2e-runner; \
	EXIT=$$?; \
	BOOKING_IMAGE=$${BOOKING_IMAGE:-booking-service-test:local} \
		docker compose -f docker-compose.test.yml down -v; \
	exit $$EXIT

# Run integration tests (requires docker containers for DB & Kafka)
test-integration:
	chmod +x scripts/run_integration_tests.sh && ./scripts/run_integration_tests.sh

# Run all tests sequentially
test-all: test-unit test-contract test-e2e test-integration

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
	go run tests/mock/mock_event/publish_mock_event.go

# Run the standalone gRPC mock server for Profile and Finance services
mock-grpc:
	go run tests/mock/mock_grpc/mock_grpc.go
