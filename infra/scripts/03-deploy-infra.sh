#!/bin/bash
set -euo pipefail

ENV=${1:-dev}

if [[ "$ENV" != "dev" && "$ENV" != "prod" ]]; then
  echo "Error: Environment must be 'dev' or 'prod'"
  exit 1
fi

echo "Deploying base infra and Kafka operator for environment: ${ENV}..."

# Deploy operator and base namespaces using Kustomize and pipe to kubectl apply
kubectl kustomize "infra/k8s/envs/${ENV}/" --enable-helm | kubectl apply -f -

echo "Waiting for Strimzi Kafka CRDs to be established..."
kubectl wait --for=condition=established crd/kafkas.kafka.strimzi.io crd/kafkanodepools.kafka.strimzi.io --timeout=60s

echo "Deploying Kafka cluster custom resources..."
kubectl apply -f "infra/k8s/envs/${ENV}/kafka-cluster.yaml"

echo "Base infra and Kafka deployed successfully for ${ENV}"