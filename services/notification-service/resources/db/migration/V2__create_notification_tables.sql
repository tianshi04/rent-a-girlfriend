-- Create notifications and delivery_attempts tables
-- Reference: docs/data-model.md & docs/adr/0002-database-choice-postgresql.md

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    type VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL,
    payload JSONB NOT NULL,
    policy_overrides JSONB,
    status VARCHAR(20) NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_notifications_event_id_user UNIQUE (event_id, user_id)
);

CREATE TABLE delivery_attempts (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message_id VARCHAR(128),
    error_code VARCHAR(50),
    error_message TEXT,
    attempted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_delivery_attempts_notification FOREIGN KEY (notification_id) 
        REFERENCES notifications(id) ON DELETE CASCADE
);

CREATE INDEX idx_notifications_user_inbox ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_stuck_status ON notifications (status, created_at) WHERE status IN ('PENDING', 'PROCESSING');
CREATE INDEX idx_delivery_attempts_notif_id ON delivery_attempts (notification_id);
