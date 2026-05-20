import asyncio
import logging
import sys
import grpc
import uvicorn

from internal.bootstrap import settings, SessionLocal, outbox_worker, app, init_db
from gen.profile.v1.service import profile_service_pb2_grpc
from internal.interfaces.grpc.servicer import ProfileServiceServicer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("server")


async def run_grpc_server():
    server = grpc.aio.server()
    servicer = ProfileServiceServicer(SessionLocal)
    profile_service_pb2_grpc.add_ProfileServiceServicer_to_server(servicer, server)
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

    # Start Transactional Outbox Worker
    try:
        await outbox_worker.start()
    except Exception as e:
        logger.warning(
            f"Outbox Worker failed to start: {e}. Check your Kafka configuration."
        )

    # Concurrently execute gRPC Server and FastAPI Server
    await asyncio.gather(run_grpc_server(), run_http_server())


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Server shutdown initiated...")
        # Stop background worker loop
        asyncio.run(outbox_worker.stop())
        logger.info("Server successfully stopped.")
