# --- Builder Stage ---
FROM rust:slim-bookworm AS builder

WORKDIR /usr/src/app

# Install compilation dependencies
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    cmake \
    build-essential \
    libssl-dev \
    libcurl4-openssl-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Copy cargo manifest and lock file
COPY services/interaction-service/Cargo.toml services/interaction-service/Cargo.lock ./
# Create dummy src file to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
# Build dependencies to cache them
RUN cargo build --release
RUN rm -rf src

# Copy real source, migrations, and contracts
COPY services/interaction-service/build.rs ./
COPY services/interaction-service/src/ ./src/
COPY services/interaction-service/migrations/ ./migrations/
# Copy the contracts directory which is outside this service directory.
# This requires running docker build with the repository root as context.
COPY contracts/ /usr/contracts/

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

EXPOSE 8080 50051

CMD ["/app/interaction-service"]
