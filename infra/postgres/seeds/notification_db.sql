-- Seed Notification Service
DELETE FROM delivery_attempts;
DELETE FROM notifications;

INSERT INTO notifications (id, user_id, event_id, type, priority, payload, status, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'evt_booking_completed_1', 'BOOKING_COMPLETED', 'MEDIUM', '{"bookingId":"b0000000-0000-0000-0000-000000000001","message":"Your booking with Chizuru has been completed!"}', 'SENT', CURRENT_TIMESTAMP - INTERVAL '1 hour', CURRENT_TIMESTAMP - INTERVAL '1 hour')
ON CONFLICT (id) DO NOTHING;
