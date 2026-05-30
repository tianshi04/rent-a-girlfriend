import asyncio
import logging
import sys
import grpc
import uvicorn

from internal.bootstrap import (
    settings,
    SessionLocal,
    outbox_worker,
    saga_retry_worker,
    event_consumer,
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


async def run_grpc_server():
    server = grpc.aio.server()
    servicer = DisputeServiceServicer(SessionLocal)
    dispute_service_pb2_grpc.add_DisputeServiceServicer_to_server(servicer, server)
    listen_addr = f"0.0.0.0:{settings.GRPC_PORT}"
    server.add_insecure_port(listen_addr)
    logger.info(f"Starting gRPC server on {listen_addr}...")
    await server.start()
    await server.wait_for_termination()


async def run_http_server():
    config = uvicorn.Config(
        app=app, host="0.0.0.0", port=settings.SERVER_PORT, log_level="info"
    )
    server = uvicorn.Server(config)
    logger.info(f"Starting HTTP/REST server on port {settings.SERVER_PORT}...")
    await server.serve()


async def main():
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

    # Concurrently execute gRPC Server and FastAPI Server
    try:
        await asyncio.gather(run_grpc_server(), run_http_server())
    finally:
        logger.info("Stopping background workers...")
        await outbox_worker.stop()
        await saga_retry_worker.stop()
        await event_consumer.stop()
        logger.info("Server successfully stopped.")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Server shutdown initiated by KeyboardInterrupt...")
