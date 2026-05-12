#!/bin/bash
set -euo pipefail

CLUSTER_NAME="micro"

echo "Creating Kind cluster..."

kind create cluster \
  --name $CLUSTER_NAME \
  --config infra/kind/cluster.yaml \
  --image kindest/node:v1.29.2

echo "Cluster created"