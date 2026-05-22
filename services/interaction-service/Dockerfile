# --- Builder Stage ---
FROM rust:1.79-slim AS builder

WORKDIR /usr/src/app

# Install compilation dependencies
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    cmake \
    build-essential \
    libssl-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Copy cargo manifest and build files
COPY Cargo.toml ./
# Create dummy src file to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
# Build dependencies to cache them
RUN cargo build --release
RUN rm -rf src

# Copy real source and contracts
COPY build.rs ./
COPY src/ ./src/
# Copy the contracts directory which is outside this service directory
# (Note: In standard Docker context you must run from workspace root, but for local Kubernetes
# setup we assume standard build context will copy contracts or we use volume mount)
COPY ../../contracts/ ../../contracts/

# Force compile target using actual source files
RUN touch src/main.rs && cargo build --release

# --- Runner Stage ---
FROM debian:bookworm-slim AS runner

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /usr/src/app/target/release/interaction-service /app/interaction-service

EXPOSE 8083 50054

CMD ["/app/interaction-service"]
