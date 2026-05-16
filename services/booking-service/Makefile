.PHONY: run build test docker-up docker-down tidy

# Run locally
run:
	go run ./cmd/server

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/booking-service ./cmd/server

# Run unit tests
test:
	go test ./internal/... -v -count=1

# Docker compose up
docker-up:
	docker compose up --build -d

# Docker compose down
docker-down:
	docker compose down -v

# Tidy dependencies
tidy:
	go mod tidy
