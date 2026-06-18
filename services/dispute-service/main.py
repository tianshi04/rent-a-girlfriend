import asyncio
import logging
import signal
import sys
import grpc
import uvicorn

from internal.bootstrap import (
    settings,
    SessionLocal,
    outbox_worker,
    saga_retry_worker,
    event_consumer,
    db_cleanup_worker,
    app,
    init_db,
)
from gen.dispute.v1.service import dispute_service_pb2_grpc
from internal.interfaces.grpc.servicer import DisputeServiceServicer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("server")


async def run_grpc_server(shutdown_event: asyncio.Event):
    server = grpc.aio.server()
    servicer = DisputeServiceServicer(SessionLocal)
    dispute_service_pb2_grpc.add_DisputeServiceServicer_to_server(servicer, server)
    listen_addr = f"0.0.0.0:{settings.GRPC_PORT}"
    server.add_insecure_port(listen_addr)
    logger.info(f"Starting gRPC server on {listen_addr}...")
    await server.start()

    # Wait for the shutdown signal
    await shutdown_event.wait()
    logger.info(
        "Gracefully stopping gRPC server (waiting for active RPCs to finish)..."
    )
    await server.stop(grace=5)


async def run_http_server(shutdown_event: asyncio.Event):
    config = uvicorn.Config(
        app=app,
        host="0.0.0.0",
        port=settings.SERVER_PORT,
        log_level="info",
    )
    server = uvicorn.Server(config)
    logger.info(f"Starting HTTP/REST server on port {settings.SERVER_PORT}...")

    # Start the server as a background task
    server_task = asyncio.create_task(server.serve())

    # Wait for the shutdown signal
    await shutdown_event.wait()
    logger.info("Gracefully stopping HTTP/REST server...")
    server.should_exit = True
    await server_task


async def main():
    loop = asyncio.get_running_loop()
    shutdown_event = asyncio.Event()

    def handle_shutdown(sig_name):
        logger.info(f"Received {sig_name}, initiating graceful shutdown...")
        shutdown_event.set()

    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, lambda s=sig: handle_shutdown(s.name))

    # Initialize Database Tables
    await init_db()

    # Start transactional Outbox Worker
    try:
        await outbox_worker.start()
    except Exception as e:
        logger.warning(
            f"Outbox Worker failed to start: {e}. Check Kafka configuration."
        )

    # Start Saga Retry Worker
    try:
        await saga_retry_worker.start()
    except Exception as e:
        logger.warning(f"Saga Retry Worker failed to start: {e}.")

    # Start Kafka Event Consumer
    try:
        await event_consumer.start()
    except Exception as e:
        logger.warning(f"Event Consumer failed to start: {e}.")

    # Start Database Cleanup Worker
    try:
        await db_cleanup_worker.start()
    except Exception as e:
        logger.warning(f"Database Cleanup Worker failed to start: {e}.")

    # Concurrently execute gRPC Server and FastAPI Server
    try:
        await asyncio.gather(
            run_grpc_server(shutdown_event), run_http_server(shutdown_event)
        )
    finally:
        logger.info("Stopping background workers...")
        await outbox_worker.stop()
        await saga_retry_worker.stop()
        await event_consumer.stop()
        await db_cleanup_worker.stop()
        logger.info("Server successfully stopped.")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Server shutdown initiated by KeyboardInterrupt...")
