# ── Stage 1: Builder ──────────────────────────────────────────────────────────
# Biên dịch mã nguồn Go một lần. Đầu ra được chia sẻ bởi cả hai runtime bên dưới.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache lớp phụ thuộc riêng biệt — chỉ bị mất hiệu lực khi go.mod/go.sum thay đổi
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG BUILD_TAGS="production"
# -s -w: loại bỏ ký hiệu gỡ lỗi và thông tin DWARF (giúp file chạy nhỏ hơn ~20%)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -tags "${BUILD_TAGS}" \
    -o /booking-service \
    ./cmd/server

# ── Stage 2: test ─────────────────────────────────────────────────────────────
# Runtime dựa trên Alpine phục vụ việc chạy test E2E và Smoke Test trong CI.
# Giữ lại wget/sh để chạy health check trong docker compose.
# KHÔNG đẩy lên GHCR — chỉ dùng làm đích chạy test trong CI.
FROM alpine:3.19 AS test

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /booking-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
EXPOSE 50051

ENTRYPOINT ["./booking-service"]

# ── Stage 3: production ───────────────────────────────────────────────────────
# Runtime Distroless tối giản — không có shell, không có quản lý gói, giảm thiểu bề mặt tấn công.
# Các chứng chỉ ca-certificates và dữ liệu tzdata đã được tích hợp sẵn trong ảnh gốc.
# Đây là ảnh Docker chính thức được đẩy lên GHCR.
FROM gcr.io/distroless/static-debian12 AS production

WORKDIR /app
COPY --from=builder /booking-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
EXPOSE 50051

ENTRYPOINT ["./booking-service"]
