#!/bin/bash
set -euo pipefail

kubectl apply -f infra/k8s/base/

echo "Base infra deployed"