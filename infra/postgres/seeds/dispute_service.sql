-- Seed Dispute Service
DELETE FROM saga_states;
DELETE FROM outbox;
DELETE FROM processed_events;
DELETE FROM disputes;

INSERT INTO disputes (dispute_id, booking_id, reporter_id, accused_id, reason, status, admin_id, resolution, notes, version, created_at, updated_at) VALUES
('e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'MISCONDUCT', 'REFUNDED', 'a0000000-0000-0000-0000-000000000003', 'REFUND_CLIENT', 'Refunded 50 Kano-Coins to client due to late arrival.', 1, CURRENT_TIMESTAMP - INTERVAL '12 hours', CURRENT_TIMESTAMP - INTERVAL '10 hours')
ON CONFLICT (dispute_id) DO NOTHING;
