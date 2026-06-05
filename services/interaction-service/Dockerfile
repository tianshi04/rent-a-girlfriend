# --- Cargo Chef Planner Stage ---
FROM lukemathwalker/cargo-chef:latest-rust-alpine AS planner
WORKDIR /usr/src/app
COPY services/interaction-service/Cargo.toml services/interaction-service/Cargo.lock ./
COPY services/interaction-service/build.rs ./
COPY services/interaction-service/src/ ./src/
RUN cargo chef prepare --recipe-path recipe.json

# --- Cargo Chef Cacher Stage ---
FROM lukemathwalker/cargo-chef:latest-rust-alpine AS cacher
WORKDIR /usr/src/app
RUN apk add --no-cache \
    protobuf-dev \
    cmake \
    make \
    g++ \
    gcc \
    musl-dev \
    openssl-dev \
    openssl-libs-static \
    git \
    pkgconfig \
    perl \
    curl-dev \
    zlib-static \
    lld

COPY --from=planner /usr/src/app/recipe.json recipe.json
ENV RUSTFLAGS="-C link-arg=-fuse-ld=lld"
RUN cargo chef cook --release --recipe-path recipe.json

# --- Builder Stage ---
FROM rust:alpine AS builder
WORKDIR /usr/src/app
RUN apk add --no-cache protobuf-dev lld g++ gcc musl-dev

# Copy dependency build artifacts and registry cache
COPY --from=cacher /usr/src/app/target/ ./target/
COPY --from=cacher /usr/local/cargo/ /usr/local/cargo/

# Copy real source, migrations, and contracts
COPY services/interaction-service/Cargo.toml services/interaction-service/Cargo.lock ./
COPY services/interaction-service/build.rs ./
COPY services/interaction-service/src/ ./src/
COPY services/interaction-service/migrations/ ./migrations/
# Copy the contracts directory which is outside this service directory.
# This requires running docker build with the repository root as context.
COPY contracts/ /usr/contracts/

# Compile application using the cached target and lld linker
ENV RUSTFLAGS="-C link-arg=-fuse-ld=lld"
RUN cargo build --release

# --- Runner Stage ---
FROM gcr.io/distroless/static-debian12 AS runner
WORKDIR /app
COPY --from=builder /usr/src/app/target/release/interaction-service /app/interaction-service
EXPOSE 8080 50051
CMD ["/app/interaction-service"]
