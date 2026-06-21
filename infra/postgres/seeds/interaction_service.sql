-- Seed Interaction Service Chats and Reviews
DELETE FROM chat_messages;
DELETE FROM chat_rooms;
DELETE FROM reviews;
DELETE FROM outbox;
DELETE FROM processed_events;
INSERT INTO chat_rooms (room_id, booking_id, client_id, companion_id, status, created_at, updated_at) VALUES
('r0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (room_id) DO NOTHING;

INSERT INTO chat_messages (message_id, room_id, sender_id, content, created_at) VALUES
('m0000000-0000-0000-0000-000000000001', 'r0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'Hello Chizuru! Looking forward to our date.', CURRENT_TIMESTAMP - INTERVAL '1 hour'),
('m0000000-0000-0000-0000-000000000002', 'r0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000002', 'Hi! Me too, I will see you soon!', CURRENT_TIMESTAMP - INTERVAL '50 minutes')
ON CONFLICT (message_id) DO NOTHING;

INSERT INTO reviews (review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at) VALUES
('v0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 5, 'She was absolutely amazing! Highly recommended!', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (review_id) DO NOTHING;
