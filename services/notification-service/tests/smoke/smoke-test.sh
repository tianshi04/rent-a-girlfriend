#!/bin/bash
set -e

# Get script's parent folder
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"

echo "=========================================="
echo "[*] STARTING NOTIFICATION SERVICE SMOKE TEST"
echo "=========================================="

# 1. Build the Notification Service Docker Image
echo -e "\n[+] Step 1: Building Docker image..."
docker build -t rentagf/notification-service:smoke -f ../../Dockerfile ../../

# 2. Run Docker Compose
echo -e "\n[+] Step 2: Spinning up container environment..."
NOTIFICATION_IMAGE=rentagf/notification-service:smoke docker compose -f docker-compose.smoke.yml up -d

# 3. Poll Actuator Health Endpoint
echo -e "\n[*] Step 3: Waiting for Actuator health status (UP)..."
url="http://localhost:8084/actuator/health"
maxRetries=30
started=false

for ((i=1; i<=maxRetries; i++)); do
    response=$(curl -s $url || true)
    if echo "$response" | grep -q '"status":"UP"'; then
        echo -e "\n[V] SUCCESS: Notification Service started successfully!"
        echo "Response: $response"
        started=true
        break
    else
        echo -n "."
    fi
    sleep 2
done

if [ "$started" = false ]; then
    echo -e "\n[X] FAILED: Service did not start successfully or report UP status within timeout."
    echo "Checking service container logs:"
    docker logs notification-service-smoke
    
    echo -e "\n[-] Cleaning up environment..."
    docker compose -f docker-compose.smoke.yml down -v --remove-orphans
    exit 1
fi

# 4. Cleanup
echo -e "\n[-] Step 4: Cleaning up environment..."
docker compose -f docker-compose.smoke.yml down -v --remove-orphans
echo "Smoke test completed and cleaned up."
