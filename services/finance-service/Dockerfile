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
# Copy shared contracts (Expected to be in context)
COPY contracts/ ./contracts/
COPY . .
RUN uv sync --frozen --no-dev && \
    CONTRACTS_PATH=/app/contracts uv run python scripts/generate_proto.py

# ---- Stage 3: Tester ----
FROM builder AS tester
# Install development dependencies (including pytest)
RUN uv sync --frozen
# Run tests - If tests fail, the build will stop here
ENV PYTHONPATH="/app:/app/gen"
RUN uv run pytest tests/

# ---- Stage 4: Runtime ----
FROM python:3.12-slim-bookworm AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends tini && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
RUN groupadd -r appgroup && useradd -r -g appgroup appuser
# Copy production venv from base (clean, no dev tools)
COPY --from=base /app/.venv /app/.venv
# Copy generated code and source from tester (FORCING tests to run)
COPY --from=tester /app/gen /app/gen
COPY --from=tester /app/internal /app/internal
COPY --from=tester /app/main.py /app/main.py
# Ensure the runtime image knows about the generated code
RUN chown -R appuser:appgroup /app
ENV PATH="/app/.venv/bin:$PATH" \
    PYTHONPATH="/app/gen" \
    PYTHONUNBUFFERED=1
EXPOSE 8080 50051
USER appuser
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["python", "main.py"]
