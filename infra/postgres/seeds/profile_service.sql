-- Seed Profile & Catalogue Service
DELETE FROM outbox;
DELETE FROM companion_profiles;

INSERT INTO companion_profiles (companion_id, user_id, display_name, intro_text, status, available_cities, avatar_url, created_at, updated_at) VALUES
('d0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', 'Chizuru Mizuhara', 'Professional rental girlfriend. I will make your day unforgettable!', 'APPROVED', '["Hanoi", "Saigon"]', 'http://localhost:9000/rentgf-media/chizuru.jpg', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (companion_id) DO NOTHING;

INSERT INTO scenarios (scenario_id, companion_id, title, description, price, duration_minutes, status, created_at) VALUES
('s0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'Coffee Date & Walk', 'Enjoy a cozy coffee chat followed by a romantic walk in the park.', 100, 60, 'ACTIVE', CURRENT_TIMESTAMP),
('s0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', 'Amusement Park', 'A full day of fun rides and sweet moments together.', 300, 120, 'ACTIVE', CURRENT_TIMESTAMP)
ON CONFLICT (scenario_id) DO NOTHING;
