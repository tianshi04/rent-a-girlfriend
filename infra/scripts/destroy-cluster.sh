#!/bin/bash
set -euo pipefail

kind delete cluster --name micro || true

echo "Base infra teared down"