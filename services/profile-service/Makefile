.PHONY: run test sync generate-proto

# Run FastAPI & gRPC server locally using uv
run:
	PYTHONPATH=$$(pwd)/gen uv run main.py

# Generate Python gRPC code from local proto definition
generate-proto:
	@mkdir -p gen
	uv run python -m grpc_tools.protoc \
		-I../../contracts/ \
		--python_out=gen \
		--pyi_out=gen \
		--grpc_python_out=gen \
		$$(find ../../contracts/profile -name "*.proto")
	@find gen -type d -exec touch {}/__init__.py \;

# Run all tests sequentially
test:
	PYTHONPATH=$$(pwd)/gen uv run pytest -v
