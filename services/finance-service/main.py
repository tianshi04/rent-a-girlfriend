import asyncio
import logging
import sys
import json
import grpc
import uvicorn
from aiokafka import AIOKafkaConsumer

from internal.bootstrap import (
    settings,
    SessionLocal,
    outbox_worker,
    app,
    init_db,
    bootstrap_services,
)
from finance.v1.service import finance_service_pb2_grpc
from internal.interfaces.grpc.servicer import FinanceServiceServicer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)
logger = logging.getLogger("server")


async def run_grpc_server():
    server = grpc.aio.server()
    servicer = FinanceServiceServicer(SessionLocal)
    finance_service_pb2_grpc.add_FinanceServiceServicer_to_server(servicer, server)
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


async def run_identity_event_listener():
    """
    Background worker listening to Kafka identity-events.
    Automatically onboards wallet for newly registered users (Option B).
    """
    logger.info(
        f"Starting Identity Event Listener on topic: {settings.KAFKA_TOPIC_IDENTITY}..."
    )
    consumer = AIOKafkaConsumer(
        settings.KAFKA_TOPIC_IDENTITY,
        bootstrap_servers=settings.KAFKA_BROKERS,
        group_id="finance-service-onboarder",
        value_deserializer=lambda m: json.loads(m.decode("utf-8")),
        auto_offset_reset="earliest",
    )

    # Try starting the consumer with a retry loop to prevent crashing if Kafka is down
    retries = 5
    while retries > 0:
        try:
            await consumer.start()
            logger.info("Identity Event Listener connected to Kafka successfully.")
            break
        except Exception as e:
            retries -= 1
            logger.warning(
                f"Failed to start Identity Event Listener: {e}. Retrying in 5 seconds... ({retries} retries left)"
            )
            await asyncio.sleep(5)

    if retries == 0:
        logger.error(
            "Could not start Identity Event Listener because Kafka is unavailable."
        )
        return

    try:
        async for msg in consumer:
            logger.info(f"Received identity event: {msg.value}")
            try:
                # CloudEvents structure usually wraps data inside 'data' field
                event_data = msg.value.get("data", {})
                user_id = event_data.get("user_id") or event_data.get("id")

                if user_id:
                    logger.info(f"Onboarding wallet for user_id: {user_id}")
                    async with SessionLocal() as session:
                        cmd_service = bootstrap_services(session)
                        await cmd_service.create_wallet_onboard(user_id)
                        await session.commit()
                        logger.info(
                            f"Wallet onboarded successfully for user_id: {user_id}"
                        )
                else:
                    logger.warning(f"No user_id found in event payload: {msg.value}")
            except Exception as e:
                logger.error(f"Error processing identity event: {e}", exc_info=True)
    except asyncio.CancelledError:
        logger.info("Identity Event Listener task cancelled.")
    finally:
        await consumer.stop()
        logger.info("Identity Event Listener stopped.")


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

    # Start Identity Listener in background
    identity_listener_task = asyncio.create_task(run_identity_event_listener())

    # Concurrently execute gRPC Server and FastAPI Server
    try:
        await asyncio.gather(
            run_grpc_server(), run_http_server(), return_exceptions=True
        )
    finally:
        # Clean up background listener
        identity_listener_task.cancel()
        try:
            await identity_listener_task
        except asyncio.CancelledError:
            pass


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Server shutdown initiated...")
        # Stop background worker loop
        asyncio.run(outbox_worker.stop())
        logger.info("Server successfully stopped.")
