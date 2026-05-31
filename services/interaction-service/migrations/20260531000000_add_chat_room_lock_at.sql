-- Add lock_at column to chat_rooms table
ALTER TABLE chat_rooms ADD COLUMN lock_at TIMESTAMP WITH TIME ZONE;

-- Create a partial index for the background worker to scan active rooms scheduled for locking
CREATE INDEX IF NOT EXISTS idx_chat_rooms_status_lock_at 
ON chat_rooms(status, lock_at) 
WHERE status = 'ACTIVE' AND lock_at IS NOT NULL;
