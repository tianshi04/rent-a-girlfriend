.PHONY: run test sync generate-proto build lint

# Run FastAPI & gRPC server locally using uv
run:
	PYTHONPATH=$$PWD/gen uv run main.py

# Generate Python gRPC code from local proto definition
generate-proto:
	uv run python scripts/generate_proto.py

# Lint and format check code using ruff
lint:
	uv run ruff check internal
	uv run ruff format --check internal

# Build Docker image from local context (Includes automatic testing)
build:
	@echo "Copying shared contracts into build context..."
	rm -rf ./contracts
	cp -r ../../contracts ./contracts
	docker build -t finance-service .
	@echo "Cleaning up build context..."
	rm -rf ./contracts

# Run all tests locally
test:
	PYTHONPATH=$$PWD/gen uv run pytest -v
