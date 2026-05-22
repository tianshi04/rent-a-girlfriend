.PHONY: run test build clean lint

# Run Axum REST and Tonic gRPC server locally
run:
	cargo run

# Run unit and integration tests
test:
	cargo test

# Build optimized binary for production
build:
	cargo build --release

# Clean up target build files
clean:
	cargo clean

# Run formatting and linting checks
lint:
	cargo fmt -- --check
	cargo clippy -- -D warnings

# Build docker image using the workspace root as context
docker-build:
	docker build -t interaction-service:latest -f Dockerfile ../..
