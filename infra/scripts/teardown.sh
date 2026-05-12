#!/bin/bash
set -euo pipefail

kind delete cluster --name micro || true
docker rm -f local-registry || true

echo "Base infra teared down"