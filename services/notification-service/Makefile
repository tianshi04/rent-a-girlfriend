IMAGE_TAG ?= latest

# OS Detection for Gradle execution
ifeq ($(OS),Windows_NT)
    GRADLE_CMD = gradlew.bat
else
    GRADLE_CMD = ./gradlew
endif

.PHONY: run build test-unit clean docker-build up down dev test-docker test-docker-unit test-docker-int

# Run the Spring Boot application locally (on host machine)
run:
	$(GRADLE_CMD) bootRun

# Build the Spring Boot bootable JAR
build:
	$(GRADLE_CMD) bootJar

# Run all unit/integration tests
test-unit:
	$(GRADLE_CMD) test

# Clean all build artifacts
clean:
	$(GRADLE_CMD) clean

# Build the standalone production-ready Docker image using Multi-stage build
docker-build:
	docker build -t rentagf/notification-service:$(IMAGE_TAG) .

# [FOR CLONE/CI-CD] Start all services in Docker (Multi-stage build, self-contained)
up:
	docker compose up --build -d

# [FOR LOCAL DEV] Build JAR first and up Docker container in under 2 seconds
dev:
	$(GRADLE_CMD) bootJar
	docker compose -f docker-compose.local.yml up --build -d

# Stop all docker compose services
down:
	docker compose -f docker-compose.local.yml down
	docker compose down

# ── Test in Docker ────────────────────────────────────────────────────────────

# [TEST] Chạy TẤT CẢ test trong Docker (H2 in-memory, không cần infra nào)
test-docker:
	docker compose -f docker-compose.test.yml run --rm test-runner \
		./gradlew test --no-daemon

# [TEST] Chỉ chạy test có @Tag("unit")
test-docker-unit:
	docker compose -f docker-compose.test.yml run --rm test-runner \
		./gradlew unitTest --no-daemon

# [TEST] Chỉ chạy test có @Tag("integration")
test-docker-int:
	docker compose -f docker-compose.test.yml run --rm test-runner \
		./gradlew integrationTest --no-daemon
