# SERVICE TEMPLATE (CHẨN HOÁ CẤU TRÚC MICROSERVICE)

Tài liệu này định nghĩa cấu trúc thư mục và phân lớp logic tiêu chuẩn cho mọi Microservice trong hệ thống Rent-a-Girlfriend. Mục tiêu là đảm bảo tính nhất quán, dễ bảo trì và tách biệt hoàn toàn giữa nghiệp vụ (Domain) và công nghệ (Infrastructure).

---

## 1. MÔ HÌNH KIẾN TRÚC: HEXAGONAL ARCHITECTURE

Hệ thống áp dụng kiến trúc **Hexagonal (Ports and Adapters)**. Logic nghiệp vụ nằm ở trung tâm và không phụ thuộc vào bất kỳ framework hay database cụ thể nào.

### Các lớp logic chính:
1.  **Domain Layer (Lõi):** Chứa các quy tắc nghiệp vụ, Aggregates, Entities, Value Objects.
2.  **Application Layer (Điều phối):** Chứa Use Cases, Command/Query Handlers và quản lý giao dịch.
3.  **Infrastructure Layer (Hạ tầng):** Triển khai kỹ thuật như DB Access, Message Broker, API Client.
4.  **Interfaces Layer (Cổng vào):** Định nghĩa cách thế giới bên ngoài giao tiếp với service (HTTP, gRPC, Pub/Sub).

---

## 2. CẤU TRÚC THƯ MỤC TIÊU CHUẨN

```text
services/[service-name]/
├── cmd/                   # Điểm khởi đầu (Entry Point)
│   └── server/            # Khởi tạo DI, cấu hình và chạy Server
├── internal/              # Code riêng tư của service
│   ├── bootstrap/         # Khởi tạo DI, config, logger, kết nối DB/Broker
│   ├── domain/            # TẦNG LÕI (DOMAIN LAYER)
│   │   ├── aggregate/     # Aggregate Roots & Entities
│   │   ├── vo/            # Value Objects
│   │   ├── repository/    # Port: Interfaces cho Database
│   │   └── events/        # Định nghĩa Domain Events
│   ├── application/       # TẦNG ỨNG DỤNG (APPLICATION LAYER)
│   │   ├── commands/      # Xử lý các lệnh thay đổi trạng thái (CQRS)
│   │   ├── queries/       # Xử lý các lệnh lấy dữ liệu (CQRS)
│   │   └── saga/          # Điều phối các giao dịch phân tán
│   ├── infrastructure/    # TẦNG HẠ TẦNG (INFRASTRUCTURE LAYER)
│   │   ├── persistence/   # Adapter: Triển khai DB (Gorm, MongoDB,...)
│   │   ├── broker/        # Adapter: Gửi/Nhận Message (RabbitMQ, Kafka)
│   │   └── client/        # Adapter: Gọi API các service khác
│   └── interfaces/        # TẦNG GIAO DIỆN (INTERFACES LAYER)
│       ├── http/          # REST API Controllers & Routes
│       ├── grpc/          # gRPC Request Handlers
│       └── event_sub/     # Lắng nghe (Subscribe) các sự kiện bên ngoài
├── gen/                   # Code được generated từ Protobuf/AsyncAPI
├── deployments/           # HẠ TẦNG TRIỂN KHAI (DOCKER-READY/REPO-READY)
│   ├── k8s/               # Kubernetes Manifests (Deployment, Service, HPA)
│   └── istio/             # Istio Policies
├── tests/                 # Integration & E2E Tests
├── docs/                  # Tài liệu kỹ thuật riêng cho service này
├── README.md              # Hướng dẫn nhanh
├── Dockerfile             # Containerization (đặt tại root service)
├── Makefile               # Phím tắt cho các lệnh common (build, test, run, deploy)
└── .env.example           # Cấu hình mẫu
```

---

## 3. QUY TẮC PHỤ THUỘC (DEPENDENCY RULES)

Để tránh tạo ra một "Spaghetti code", team phải tuân thủ hướng phụ thuộc **từ ngoài vào trong**:
- `Domain` không được phép import bất kỳ thứ gì từ các tầng khác.
- `Application` chỉ được import `Domain`.
- `Infrastructure` và `Interfaces` phụ thuộc vào `Application` và `Domain`.

> [!IMPORTANT]
> Mọi giao tiếp với ngoại vi (DB, Redis, API bên ngoài) trong tầng Domain/Application phải thông qua **Interface (Port)**. Việc triển khai thật (Adapter) nằm ở tầng Infrastructure.

---

## 4. CHIẾN LƯỢC KIỂM THỬ (TESTING)

- **Unit Tests:** Đặt cạnh file code nguồn (`*.test.go` hoặc `*.spec.ts`). Tập trung test logic trong `domain` và `application`.
- **Integration Tests:** Đặt trong thư mục `tests/`. Test sự kết hợp giữa Service và Database/Broker thật (hoặc Testcontainers).

---

## 5. CONTAINERIZATION (DOCKER)

Mỗi service phải có một `Dockerfile` multi-stage để tối ưu dung lượng image:
1.  **Stage 1 (Build):** Biên dịch code.
2.  **Stage 2 (Runtime):** Chỉ chứa file thực thi và các thư viện cần thiết.

---

## 6. CROSS-CUTTING CONCERNS

Các thành phần chung phải có mặt trong mọi service:
- **Logging:** Sử dụng Structured Logging (JSON format).
- **Observability:** Health check endpoint (`/health`), Metrics (Prometheus) và Tracing (OpenTelemetry).
- **Error Handling:** Sử dụng các Domain Error chung để API Gateway có thể map về HTTP Status phù hợp.
