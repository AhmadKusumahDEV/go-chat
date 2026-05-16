-- Seed data for WebSocket testing
-- Password for all users: test1234 (SHA256 hash)

UPDATE users SET password_hash = '937e8d5fbb48bd4949536cd65b8d35c426b80d2f830c5c308e2cdec422ae2244';

-- Create 2 test rooms
INSERT INTO rooms (id, room_name, description, room_type, is_private, created_by) VALUES
(
    '11111111-1111-1111-1111-111111111111',
    'General Chat',
    'Public room for general discussions',
    'group',
    FALSE,
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890'  -- john_doe
),
(
    '22222222-2222-2222-2222-222222222222',
    'Dev Team',
    'Development team private room',
    'group',
    TRUE,
    'b2c3d4e5-f6a7-8901-bcde-f12345678901'  -- jane_smith
);

-- Add members to General Chat (all 5 users)
INSERT INTO room_members (room_id, user_id, added_by, role) VALUES
('11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'admin'),
('11111111-1111-1111-1111-111111111111', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'e5f6a7b8-c9d0-1234-efab-345678901234', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member');

-- Add members to Dev Team (3 users: jane, bob, alice)
INSERT INTO room_members (room_id, user_id, added_by, role) VALUES
('22222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'admin'),
('22222222-2222-2222-2222-222222222222', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'member'),
('22222222-2222-2222-2222-222222222222', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'member');
