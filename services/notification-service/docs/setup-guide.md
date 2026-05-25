# 🛠️ HƯỚNG DẪN CÀI ĐẶT MÔI TRƯỜNG (Environment Setup Guide)

## Yêu cầu

| Tool | Version | Mô tả |
|:---|:---|:---|
| **Java** | 21 | JDK (Oracle hoặc OpenJDK) |
| **Gradle** | 8.x | Build tool (hoặc dùng `./gradlew` wrapper đi kèm) |
| **PostgreSQL** | 15+ | Database chính |
| **Redis** | 7+ | Pub/Sub cho SSE phân tán |
| **Apache Kafka** | 3.x+ | Message Broker |
| **Docker** (khuyến nghị) | 24+ | Để chạy PostgreSQL, Redis, Kafka local |

---

## Cài đặt nhanh (Docker Compose)

Tạo file `docker-compose.yml` tại thư mục project:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: notification_db
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  kafka:
    image: confluentinc/cp-kafka:7.6.0
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:29093
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:29093
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      CLUSTER_ID: "notification-dev-cluster"
    ports:
      - "9092:9092"

volumes:
  pgdata:
```

Khởi chạy:
```bash
docker compose up -d
```

---

## Cài đặt thủ công (WSL/Linux)

### Java 21
```bash
sudo apt update
sudo apt install openjdk-21-jdk -y
java --version
```

### PostgreSQL
```bash
sudo apt install postgresql postgresql-contrib -y
sudo -u postgres createdb notification_db
```

### Redis
```bash
sudo apt install redis-server -y
sudo systemctl start redis
```

### Kafka
Tham khảo: https://kafka.apache.org/quickstart

---

## Chạy ứng dụng

```bash
# Dùng Gradle wrapper (không cần cài Gradle)
./gradlew bootRun

# Hoặc build JAR
./gradlew build
java -jar build/libs/notification-service-0.0.1-SNAPSHOT.jar
```

---

## Environment Variables

| Variable | Default | Mô tả |
|:---|:---|:---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `notification_db` | Database name |
| `DB_USERNAME` | `postgres` | Database username |
| `DB_PASSWORD` | `postgres` | Database password |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `KAFKA_BOOTSTRAP_SERVERS` | `localhost:9092` | Kafka brokers |
| `SERVER_PORT` | `8084` | HTTP server port |

---

## Kiểm tra

```bash
# Health check
curl http://localhost:8084/actuator/health

# Expected: {"status":"UP"}
```
