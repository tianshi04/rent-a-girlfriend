-- Add booking_id column to outbox for Kafka partition keying (Aggregate Root ID)
ALTER TABLE outbox ADD COLUMN IF NOT EXISTS booking_id VARCHAR(36);
