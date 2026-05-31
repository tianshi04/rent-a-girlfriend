import logging
from fastapi import FastAPI
from prometheus_client import CollectorRegistry

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

# Instrumentors
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
from opentelemetry.instrumentation.sqlalchemy import SQLAlchemyInstrumentor

logger = logging.getLogger("observability")


def setup_observability(app: FastAPI, service_name: str = "dispute-service"):
    """
    Initializes OpenTelemetry Tracing and integrates it with FastAPI, gRPC, and SQLAlchemy.
    """
    try:
        # 1. Setup Resource (Service metadata)
        resource = Resource.create({"service.name": service_name})

        # 2. Setup Tracer Provider
        provider = TracerProvider(resource=resource)

        # 3. Setup OTLP Exporter (Defaults to localhost:4317 if OTEL_EXPORTER_OTLP_ENDPOINT is not set)
        otlp_exporter = OTLPSpanExporter()
        processor = BatchSpanProcessor(otlp_exporter)
        provider.add_span_processor(processor)

        # Register the provider globally
        trace.set_tracer_provider(provider)

        # 4. Auto-instrument FastAPI
        FastAPIInstrumentor.instrument_app(app)

        # 5. Auto-instrument gRPC Server
        grpc_instrumentor = GrpcInstrumentorServer()
        grpc_instrumentor.instrument()

        # 6. Auto-instrument SQLAlchemy
        # (SQLAlchemy instrumentation is typically called after the engine is created,
        # but calling instrument() globally hooks into create_engine automatically)
        SQLAlchemyInstrumentor().instrument()

        logger.info(f"Observability initialized for {service_name} with OTLP Exporter")
    except Exception as e:
        logger.warning(
            f"Failed to initialize OpenTelemetry: {e}. Tracing will be disabled."
        )


def get_metrics_registry() -> CollectorRegistry:
    """
    Returns a configured Prometheus registry.
    Useful if running Uvicorn with multiple workers (multiprocess mode).
    """
    registry = CollectorRegistry()
    # multiprocess.MultiProcessCollector(registry) # Uncomment if using gunicorn workers
    return registry
