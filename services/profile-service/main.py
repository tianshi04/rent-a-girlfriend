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

    # gRPC Server Setup
    grpc_server = grpc.aio.server()
    servicer = ProfileServiceServicer(SessionLocal)
    profile_service_pb2_grpc.add_ProfileServiceServicer_to_server(servicer, grpc_server)
    listen_addr = f"0.0.0.0:{settings.GRPC_PORT}"
    grpc_server.add_insecure_port(listen_addr)
    logger.info(f"Starting gRPC server on {listen_addr}...")
    await grpc_server.start()

    # HTTP/REST Server Setup
    # Rationale: We override install_signal_handlers to a no-op so that our coordinated signal
    # handlers in the main event loop can manage the shutdown sequence of all components together.
    config = uvicorn.Config(
        app=app,
        host="0.0.0.0",
        port=settings.SERVER_PORT,
        log_level="info",
    )
    http_server = uvicorn.Server(config)
    http_server.install_signal_handlers = lambda: None
    logger.info(f"Starting HTTP/REST server on port {settings.SERVER_PORT}...")

    http_task = asyncio.create_task(http_server.serve())
    grpc_task = asyncio.create_task(grpc_server.wait_for_termination())

    # Coordinated Shutdown Event
    shutdown_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def handle_signal(sig):
        logger.info(f"Received signal {sig.name}, initiating graceful shutdown...")
        shutdown_event.set()

    # Setup signal handlers for SIGINT and SIGTERM with safe fallback for Windows platform
    import signal

    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, lambda s=sig: handle_signal(s))
        except NotImplementedError:
            pass

    shutdown_task = asyncio.create_task(shutdown_event.wait())

    try:
        # Wait until either the shutdown event is set, or one of the server tasks terminates
        await asyncio.wait(
            [shutdown_task, http_task, grpc_task],
            return_when=asyncio.FIRST_COMPLETED,
        )
    except asyncio.CancelledError:
        logger.info("Main lifecycle task cancelled, triggering shutdown...")
    finally:
        logger.info("Server shutdown initiated...")
        if not shutdown_task.done():
            shutdown_task.cancel()

        # 1. Stop HTTP server gracefully
        if not http_task.done():
            logger.info("Stopping HTTP/REST server...")
            http_server.should_exit = True
            try:
                await asyncio.wait_for(http_task, timeout=5.0)
            except asyncio.TimeoutError:
                logger.warning("HTTP/REST server stop timed out.")
            except Exception as e:
                logger.error(f"Error stopping HTTP/REST server: {e}")

        # 2. Stop gRPC server gracefully
        logger.info("Stopping gRPC server gracefully...")
        await grpc_server.stop(grace=5.0)
        try:
            await asyncio.wait_for(grpc_task, timeout=5.0)
        except asyncio.TimeoutError:
            logger.warning("gRPC server stop timed out.")
        except Exception as e:
            logger.error(f"Error stopping gRPC server: {e}")
        logger.info("gRPC server stopped.")

        # 3. Stop Outbox Worker
        logger.info("Stopping Outbox Worker...")
        try:
            await outbox_worker.stop()
        except Exception as e:
            logger.error(f"Error stopping Outbox Worker: {e}")

        logger.info("Server successfully stopped.")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        # KeyboardInterrupt can be raised at the process level before loop handles it
        logger.info("Process interrupted. Exiting.")
