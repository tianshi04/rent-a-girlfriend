-- Bookings table
CREATE TABLE IF NOT EXISTS bookings (
    id                UUID PRIMARY KEY,
    client_id         UUID NOT NULL,
    companion_id      UUID NOT NULL,
    scenario_price    BIGINT NOT NULL,
    scenario_duration INTEGER NOT NULL,
    start_time        TIMESTAMPTZ NOT NULL,
    end_time          TIMESTAMPTZ NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    cancelled_by_role VARCHAR(20),
    is_late_cancel    BOOLEAN DEFAULT FALSE,
    version           INTEGER NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bookings_client_id ON bookings(client_id);
CREATE INDEX IF NOT EXISTS idx_bookings_companion_id ON bookings(companion_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_client_companion_status ON bookings(client_id, companion_id, status);
CREATE INDEX IF NOT EXISTS idx_bookings_client_status ON bookings(client_id, status);
CREATE INDEX IF NOT EXISTS idx_bookings_companion_status ON bookings(companion_id, status);
CREATE INDEX IF NOT EXISTS idx_bookings_status_end_time ON bookings(status, end_time);
CREATE INDEX IF NOT EXISTS idx_bookings_pending_timeout ON bookings(status, created_at, start_time);

-- Outbox table (Transactional Outbox Pattern)
CREATE TABLE IF NOT EXISTS outbox (
    id             UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id   UUID NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB NOT NULL,
    correlation_id VARCHAR(100),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published      BOOLEAN DEFAULT FALSE,
    published_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(published) WHERE published = FALSE;

-- Booking Accept Sagas table (SAGA Pattern)
CREATE TABLE IF NOT EXISTS booking_accept_sagas (
    id         UUID PRIMARY KEY,
    booking_id UUID NOT NULL,
    state      VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_booking_accept_sagas_booking_id ON booking_accept_sagas(booking_id);

-- Processed Events table (Idempotency)
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(100) PRIMARY KEY,
    event_type   VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_processed_at ON processed_events(processed_at);
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox(published_at) WHERE published = TRUE;

