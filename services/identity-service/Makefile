IMAGE_TAG ?= latest

.PHONY: run build test-unit test-integration test-e2e test-all docker-up docker-down tidy deploy

# Run locally
run:
	go run ./cmd/server

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/identity-service ./cmd/server

# Unit tests — không cần external deps, bỏ qua integration tests bằng -short
test-unit:
	go test ./internal/... -v -count=1 -short

# Integration tests — dùng testcontainers (postgres + redis ephemeral)
# Cần Docker daemon đang chạy
test-integration:
	go test ./tests/integration/... -v -count=1 -timeout 120s

# E2E tests — spin up full stack, chạy tests trong container, tear down
# Cần Docker daemon đang chạy
test-e2e:
	docker compose -f docker-compose.test.yml build --no-cache
	docker compose -f docker-compose.test.yml up -d --wait
	docker compose -f docker-compose.test.yml run --rm e2e-runner; \
	EXIT=$$?; \
	docker compose -f docker-compose.test.yml down -v; \
	exit $$EXIT

# Run tất cả tests theo thứ tự
test-all: test-unit test-integration test-e2e

# Contract tests
test-contract:
	go test ./tests/contract/... -v -count=1

# Fuzz tests
test-fuzz:
	go test ./internal/domain/... -fuzz=Fuzz -fuzztime=10s

# Docker compose up (production mode)
docker-up:
	docker compose up --build -d

# Docker compose down
docker-down:
	docker compose down -v

# Tidy dependencies
tidy:
	go mod tidy

# Deploy to k8s
deploy:
	bash deploy.sh $(IMAGE_TAG)
