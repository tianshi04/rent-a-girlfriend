-- Seed Identity Service Users
DELETE FROM pkce_verifiers;
DELETE FROM outbox_events;
DELETE FROM user_accounts;

INSERT INTO user_accounts (id, email, google_id, role, status, violation_count, version, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000001', 'client@rentgf.com', 'google-oauth-client-1', 'CLIENT', 'ACTIVE', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('d0000000-0000-0000-0000-000000000002', 'companion@rentgf.com', 'google-oauth-companion-2', 'COMPANION', 'ACTIVE', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('a0000000-0000-0000-0000-000000000003', 'admin@rentgf.com', 'google-oauth-admin-3', 'ADMIN', 'ACTIVE', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;
