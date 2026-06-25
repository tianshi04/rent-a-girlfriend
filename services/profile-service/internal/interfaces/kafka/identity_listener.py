import asyncio
import json
import logging
from aiokafka import AIOKafkaConsumer
from internal.bootstrap import settings, SessionLocal, bootstrap_services

logger = logging.getLogger("identity_listener")


class IdentityEventListener:
    def __init__(self):
        self.consumer = None
        self.task = None

    async def start(self):
        self.task = asyncio.create_task(self._run())

    async def stop(self):
        if self.task:
            self.task.cancel()
            try:
                await self.task
            except asyncio.CancelledError:
                pass
            self.task = None

    async def _run(self):
        logger.info(
            f"Starting Identity Event Listener on topic: {settings.KAFKA_TOPIC_IDENTITY}..."
        )
        self.consumer = AIOKafkaConsumer(
            settings.KAFKA_TOPIC_IDENTITY,
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id=settings.KAFKA_GROUP_ID,
            value_deserializer=lambda m: json.loads(m.decode("utf-8")),
            auto_offset_reset="earliest",
        )

        retries = 5
        while retries > 0:
            try:
                await self.consumer.start()
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
            async for msg in self.consumer:
                logger.info(f"Received identity event: {msg.value}")
                try:
                    event_type = msg.value.get("type")
                    if event_type == "identity.role-upgraded.v1":
                        event_data = msg.value.get("data", {})
                        user_id = event_data.get("userId") or event_data.get("user_id")

                        if user_id:
                            logger.info(f"Upgrading profile role to COMPANION for user: {user_id}")
                            async with SessionLocal() as session:
                                profile_cmd, _, _, _ = bootstrap_services(session)
                                await profile_cmd.upgrade_profile_role(user_id)
                                await session.commit()
                                logger.info(
                                    f"Profile role upgraded successfully to COMPANION for user: {user_id}"
                                )
                        else:
                            logger.warning(f"No userId found in event payload: {msg.value}")
                except Exception as e:
                    logger.error(f"Error processing identity event: {e}", exc_info=True)
        except asyncio.CancelledError:
            logger.info("Identity Event Listener loop cancelled.")
        finally:
            await self.consumer.stop()
            logger.info("Identity Event Listener stopped.")
