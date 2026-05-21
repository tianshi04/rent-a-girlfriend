import os
import sys
import subprocess


def main():
    # Make sure we are in the services/finance-service directory or resolve absolute paths
    script_dir = os.path.dirname(os.path.abspath(__file__))
    service_dir = os.path.dirname(script_dir)
    workspace_dir = os.path.dirname(os.path.dirname(service_dir))

    # Allow overriding via environment variable for Docker builds or different environments
    contracts_dir = os.environ.get("CONTRACTS_PATH")
    
    if not contracts_dir:
        # Default logic for local development in monorepo structure
        contracts_dir = os.path.join(workspace_dir, "contracts")
        
        # Fallback for standalone/Docker build context where contracts is copied into the service root
        if not os.path.exists(contracts_dir):
            contracts_dir = os.path.join(service_dir, "contracts")

    gen_dir = os.path.join(service_dir, "gen")

    os.makedirs(gen_dir, exist_ok=True)

    print(f"Compiling protobuf contracts from {contracts_dir} to {gen_dir}...")

    # Find all .proto files in contracts/finance and contracts/common
    proto_files = []
    for root, dirs, files in os.walk(os.path.join(contracts_dir, "finance")):
        for file in files:
            if file.endswith(".proto"):
                proto_files.append(os.path.join(root, file))

    for root, dirs, files in os.walk(os.path.join(contracts_dir, "common")):
        for file in files:
            if file.endswith(".proto"):
                proto_files.append(os.path.join(root, file))

    if not proto_files:
        print("No .proto files found!")
        sys.exit(1)

    # Compile
    for proto_file in proto_files:
        print(f"Compiling: {os.path.basename(proto_file)}")
        args = [
            "grpc_tools.protoc",
            f"-I{contracts_dir}",
            f"--python_out={gen_dir}",
            f"--pyi_out={gen_dir}",
            f"--grpc_python_out={gen_dir}",
            proto_file,
        ]

        # We run it via python module to ensure exact environment
        cmd = [sys.executable, "-m", "grpc_tools.protoc"] + args[1:]
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            print(f"Error compiling {proto_file}:")
            print(result.stderr)
            sys.exit(1)

    # Touch __init__.py in all gen subdirectories
    for root, dirs, files in os.walk(gen_dir):
        init_file = os.path.join(root, "__init__.py")
        if not os.path.exists(init_file):
            open(init_file, "w").close()

    print("Protobuf compilation completed successfully.")


if __name__ == "__main__":
    main()
