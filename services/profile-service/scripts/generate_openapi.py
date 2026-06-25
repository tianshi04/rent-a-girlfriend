import os
import sys

# Set TESTING=1 before importing bootstrap to bypass live DB connection and run cleanly
os.environ["TESTING"] = "1"

# Add project root and gen directories to sys.path to resolve modules correctly
current_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(current_dir)

if project_root not in sys.path:
    sys.path.insert(0, project_root)

gen_root = os.path.join(project_root, "gen")
if gen_root not in sys.path:
    sys.path.insert(0, gen_root)

import json  # noqa: E402
from internal.bootstrap import app  # noqa: E402


def generate_openapi(output_path: str = None):
    if not output_path:
        output_path = os.path.join(project_root, "openapi.json")

    # Generate OpenAPI schema
    openapi_schema = app.openapi()

    # Create directories if they do not exist
    os.makedirs(os.path.dirname(output_path), exist_ok=True)

    # Write openapi.json
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(openapi_schema, f, indent=2, ensure_ascii=False)
    print(f"Successfully generated OpenAPI schema at {output_path}")


if __name__ == "__main__":
    # Support overriding the target file path via CLI arg
    target_file = sys.argv[1] if len(sys.argv) > 1 else None
    generate_openapi(target_file)
