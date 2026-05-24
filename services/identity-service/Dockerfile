# ---- Stage 1: Build ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
ARG BUILD_TAGS="production"
RUN CGO_ENABLED=0 GOOS=linux go build -tags "${BUILD_TAGS}" -o /identity-service ./cmd/server

# ---- Stage 2: Runtime ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /identity-service .

EXPOSE 8080
EXPOSE 50051

ENTRYPOINT ["./identity-service"]
