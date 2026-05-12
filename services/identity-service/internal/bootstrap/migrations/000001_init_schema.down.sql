-- DOWN MIGRATION

DROP TABLE IF EXISTS pkce_verifiers;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS system_configs;
DROP TABLE IF EXISTS upgrade_requests;
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_accounts;
