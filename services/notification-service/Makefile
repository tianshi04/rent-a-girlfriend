IMAGE_TAG ?= latest

# OS Detection for Gradle execution
ifeq ($(OS),Windows_NT)
    GRADLE_CMD = gradlew.bat
else
    GRADLE_CMD = ./gradlew
endif

.PHONY: run build test-unit clean docker-build

# Run the Spring Boot application locally
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

# Build the production-ready Docker image
docker-build:
	docker build -t rentagf/notification-service:$(IMAGE_TAG) .
