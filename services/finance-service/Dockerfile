# syntax=docker/dockerfile:1
# Build this Dockerfile from the current directory (services/finance-service)
# Command: make docker-build

# ---- Stage 1: Base & Production Deps ----
FROM ghcr.io/astral-sh/uv:0.9-python3.12-bookworm-slim AS base
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy
WORKDIR /app
COPY pyproject.toml uv.lock ./
# Install ONLY production dependencies first
RUN uv sync --frozen --no-dev --no-install-project

# ---- Stage 2: Builder & Proto Generation ----
FROM base AS builder
# Copy shared contracts and third_party (Expected to be in context)
COPY contracts/ ./contracts/
COPY third_party/ ./third_party/
COPY . .
RUN uv sync --frozen --no-dev && \
    CONTRACTS_PATH=/app/contracts THIRD_PARTY_PATH=/app/third_party/googleapis uv run python scripts/generate_proto.py

# ---- Stage 3: Runtime ----
FROM python:3.12-slim-bookworm AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends tini && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
RUN groupadd -r appgroup && useradd -r -g appgroup appuser
# Copy production venv from base (clean, no dev tools)
COPY --from=base /app/.venv /app/.venv
# Copy generated code and source from builder
COPY --from=builder /app/gen /app/gen
COPY --from=builder /app/internal /app/internal
COPY --from=builder /app/main.py /app/main.py
# Ensure the runtime image knows about the generated code
RUN chown -R appuser:appgroup /app
ENV PATH="/app/.venv/bin:$PATH" \
    PYTHONPATH="/app/gen" \
    PYTHONUNBUFFERED=1
EXPOSE 8080 50051
USER appuser
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["python", "main.py"]
