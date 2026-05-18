.PHONY: run test sync generate-proto

# Run FastAPI & gRPC server locally using uv
run:
	uv run main.py

# Generate Python gRPC code from local proto definition
generate-proto:
	uv run python -m grpc_tools.protoc \
		-I../../contracts/ \
		--python_out=internal/interfaces/grpc \
		--pyi_out=internal/interfaces/grpc \
		--grpc_python_out=internal/interfaces/grpc \
		profile.proto
	# Fix absolute import issue in Python 3 for generated gRPC files
	@sed -i -e 's/import profile_pb2 as profile__pb2/from . import profile_pb2 as profile__pb2/g' internal/interfaces/grpc/profile_pb2_grpc.py

# Run all tests sequentially
test:
	uv run pytest -v
