#!/bin/bash
set -euo pipefail

echo "Deploying application microservices..."

# Deploy Interaction Service
echo "Deploying interaction-service..."
helm upgrade --install interaction-service services/interaction-service/deployments/helm \
  -f services/interaction-service/deployments/helm/values.dev.yaml \
  -n interaction-service

echo "Services deployed successfully!"
