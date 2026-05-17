import os
import sys

# Dynamically add the service root directory to sys.path to resolve root-level imports
root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if root_dir not in sys.path:
    sys.path.insert(0, root_dir)

# Add grpc output directory to sys.path to solve generated proto import bug
grpc_dir = os.path.join(root_dir, "internal", "interfaces", "grpc")
if grpc_dir not in sys.path:
    sys.path.insert(0, grpc_dir)
