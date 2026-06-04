IMAGE_TAG ?= latest

# OS Detection for proto generation
ifeq ($(OS),Windows_NT)
    GEN_PROTO_CMD = powershell -ExecutionPolicy Bypass -File scripts/gen_proto.ps1
else
    GEN_PROTO_CMD = chmod +x scripts/gen_proto.sh && ./scripts/gen_proto.sh
endif

.PHONY: run build test-unit test-integration test-e2e test-all docker-up docker-down tidy deploy generate-proto

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

# E2E tests — build image local, spin up full stack, chạy tests trong container, tear down.
# IDENTITY_IMAGE có thể được override từ bên ngoài (e.g., CI đã build sẵn image).
test-e2e:
	@if [ -z "$$IDENTITY_IMAGE" ]; then \
		echo "==> [test-e2e] Building local image identity-service-test:local..."; \
		docker build --build-arg BUILD_TAGS="" --target test -t identity-service-test:local .; \
	fi
	IDENTITY_IMAGE=$${IDENTITY_IMAGE:-identity-service-test:local} \
		docker compose -f docker-compose.test.yml up -d --wait
	IDENTITY_IMAGE=$${IDENTITY_IMAGE:-identity-service-test:local} \
		docker compose -f docker-compose.test.yml run --rm e2e-runner; \
	EXIT=$$?; \
	IDENTITY_IMAGE=$${IDENTITY_IMAGE:-identity-service-test:local} \
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

# Generate cả gRPC stubs lẫn HTTP gateway
generate-proto:
	$(GEN_PROTO_CMD)

# Tidy dependencies
tidy:
	go mod tidy

# Deploy to k8s
deploy:
	bash deploy.sh $(IMAGE_TAG)
