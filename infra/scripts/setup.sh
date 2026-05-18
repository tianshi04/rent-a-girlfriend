#!/bin/bash
set -euo pipefail

CLUSTER_NAME="micro"

echo "Creating Kind cluster..."

kind create cluster \
  --name $CLUSTER_NAME \
  --config infra/kind/cluster.yaml \
  --image kindest/node:v1.35.0

echo "Connecting registry to cluster network..."
docker network connect "kind" "local-registry" || true

echo "Cluster created"