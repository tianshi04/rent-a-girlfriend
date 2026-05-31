-- Add locked_until column
ALTER TABLE outbox ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE DEFAULT NULL;

-- Create an index to optimize concurrent queries
CREATE INDEX IF NOT EXISTS idx_outbox_locked_until ON outbox(locked_until) WHERE processed = FALSE;
