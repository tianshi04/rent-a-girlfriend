# ---- Stage 1: Build ----
FROM ghcr.io/astral-sh/uv:0.9-python3.12-bookworm-slim AS builder

WORKDIR /app

# Enable bytecode compilation
ENV UV_COMPILE_BYTECODE=1

# Copy dependency files
COPY pyproject.toml uv.lock ./

# Install dependencies using uv into a virtualenv (uses pre-compiled manylinux wheels instantly)
RUN uv sync --frozen --no-dev --no-install-project

# Copy project source
COPY . .

# Build/install the project
RUN uv sync --frozen --no-dev

# ---- Stage 2: Runtime ----
FROM python:3.12-slim AS runtime

# Install tini to handle signals correctly in the container environment
RUN apt-get update && apt-get install -y --no-install-recommends tini && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create a non-root system user for security
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Copy virtualenv and code from builder
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/internal /app/internal
COPY --from=builder /app/gen /app/gen
COPY --from=builder /app/main.py /app/main.py

# Ensure ownership of /app is set to the non-root user
RUN chown -R appuser:appgroup /app

# Set environment path to use virtualenv dependencies
ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH="/app/gen"
ENV PYTHONUNBUFFERED=1

EXPOSE 8080
EXPOSE 50051

# Run as non-root user
USER appuser

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["python", "main.py"]
