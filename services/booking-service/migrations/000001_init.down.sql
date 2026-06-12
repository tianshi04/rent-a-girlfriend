DROP INDEX IF EXISTS idx_processed_events_processed_at;
DROP INDEX IF EXISTS idx_outbox_published_at;

DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS booking_accept_sagas;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS bookings;

