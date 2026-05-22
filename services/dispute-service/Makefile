.PHONY: run test sync generate-proto

run:
	PYTHONPATH=$$(pwd)/gen uv run main.py

generate-proto:
	@mkdir -p gen
	uv run python -m grpc_tools.protoc \
		-I../../contracts/ \
		--python_out=gen \
		--pyi_out=gen \
		--grpc_python_out=gen \
		$$(find ../../contracts/dispute ../../contracts/finance ../../contracts/interaction -name "*.proto")
	@find gen -type d -exec touch {}/__init__.py \;

test:
	PYTHONPATH=$$(pwd)/gen uv run pytest -v
