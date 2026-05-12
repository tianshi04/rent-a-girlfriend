-- Drop Tables
DROP TABLE IF EXISTS pkce_verifiers CASCADE;
DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS system_configs CASCADE;
DROP TABLE IF EXISTS upgrade_requests CASCADE;
DROP TABLE IF EXISTS signing_keys CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS user_accounts CASCADE;

-- Drop Extension (Optional, but good practice if created by this script)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
