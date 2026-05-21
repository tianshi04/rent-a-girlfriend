.PHONY: run test sync generate-proto

# Run FastAPI & gRPC server locally using uv
run:
	PYTHONPATH=$$PWD/gen uv run main.py

# Generate Python gRPC code from local proto definition
generate-proto:
	uv run python scripts/generate_proto.py

# Run all tests
test:
	PYTHONPATH=$$PWD/gen uv run pytest -v
