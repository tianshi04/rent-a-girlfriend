#!/bin/bash
set -euo pipefail

NAME="local-registry"

if docker ps -a --format '{{.Names}}' | grep -q "$NAME"; then
  echo "Removing existing registry..."
  docker rm -f $NAME
fi

docker run -d \
  --restart=always \
  -p 5000:5000 \
  --name $NAME \
  registry:2

echo "Registry running at localhost:5000"