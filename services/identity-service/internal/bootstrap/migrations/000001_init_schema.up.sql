-- UP MIGRATION

-- Enable UUID extension if not enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: user_accounts
CREATE TABLE IF NOT EXISTS user_accounts (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    google_id VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'CLIENT',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    violation_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Table: refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    family_id UUID NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rt_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_rt_family ON refresh_tokens(family_id);

-- Table: signing_keys
CREATE TABLE IF NOT EXISTS signing_keys (
    kid VARCHAR(50) PRIMARY KEY,
    private_key_pem TEXT NOT NULL,
    public_key_pem TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Table: upgrade_requests
CREATE TABLE IF NOT EXISTS upgrade_requests (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    reason TEXT,
    reject_reason TEXT,
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_upgrade_user ON upgrade_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_upgrade_status ON upgrade_requests(status);

-- Table: system_configs
CREATE TABLE IF NOT EXISTS system_configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Table: outbox_events
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Table: pkce_verifiers
CREATE TABLE IF NOT EXISTS pkce_verifiers (
    state VARCHAR(255) PRIMARY KEY,
    code_verifier VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Seed Initial Data
INSERT INTO system_configs (key, value, updated_at) 
VALUES ('violation_lock_threshold', '3', CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

INSERT INTO system_configs (key, value, updated_at) 
VALUES ('companion_upgrade_manual_approval', 'true', CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;
