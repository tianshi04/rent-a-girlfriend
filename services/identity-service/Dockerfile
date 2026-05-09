# ---- Stage 1: Build ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /identity-service ./cmd/server

# ---- Stage 2: Runtime ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /identity-service .

EXPOSE 8081

ENTRYPOINT ["./identity-service"]
