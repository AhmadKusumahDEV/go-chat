-- Seed data for WebSocket testing
-- Password for all users: test1234

UPDATE users SET password_hash = '937e8d5fbb48bd4949536cd65b8d35c426b80d2f830c5c308e2cdec422ae2244';

-- Create 2 test rooms
INSERT INTO rooms (id, room_name, description, room_type, is_private, created_by) VALUES
('11111111-1111-1111-1111-111111111111', 'General Chat', 'Public room for general discussions', 'group', FALSE, 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'),
('22222222-2222-2222-2222-222222222222', 'Dev Team', 'Development team private room', 'group', TRUE, 'b2c3d4e5-f6a7-8901-bcde-f12345678901');

-- Add members to General Chat (all 5 users)
INSERT INTO room_members (room_id, user_id, added_by, role) VALUES
('11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'admin'),
('11111111-1111-1111-1111-111111111111', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member'),
('11111111-1111-1111-1111-111111111111', 'e5f6a7b8-c9d0-1234-efab-345678901234', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'member');

-- Add members to Dev Team (3 users)
INSERT INTO room_members (room_id, user_id, added_by, role) VALUES
('22222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'admin'),
('22222222-2222-2222-2222-222222222222', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'member'),
('22222222-2222-2222-2222-222222222222', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'member');

-- Seed messages for General Chat
INSERT INTO messages (room_id, user_id, content, message_type, attachments, timestamp) VALUES
('11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Hey everyone! Welcome to the General Chat room! 👋', 'text', '[]'::jsonb, NOW() - INTERVAL '7 days'),
( '11111111-1111-1111-1111-111111111111', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Thanks for setting this up! Looking forward to chatting here.', 'text', '[]'::jsonb, NOW() - INTERVAL '6 days 23 hours'),
( '11111111-1111-1111-1111-111111111111', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'Hello everyone! This is bob_wilson here.', 'text', '[]'::jsonb, NOW() - INTERVAL '5 days 12 hours'),
( '11111111-1111-1111-1111-111111111111', 'e5f6a7b8-c9d0-1234-efab-345678901234', 'Nice to meet you all! Charlie here!', 'text', '[]'::jsonb, NOW() - INTERVAL '3 days 18 hours'),
( '11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Has anyone tried the new feature we discussed?', 'text', '[]'::jsonb, NOW() - INTERVAL '2 days 9 hours'),
( '11111111-1111-1111-1111-111111111111', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Yes! I tested it yesterday and it works great!', 'text', '[]'::jsonb, NOW() - INTERVAL '1 day 15 hours'),
( '11111111-1111-1111-1111-111111111111', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'Can someone share the documentation link?', 'text', '[]'::jsonb, NOW() - INTERVAL '12 hours'),
( '11111111-1111-1111-1111-111111111111', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'Here you go: https://docs.example.com/api/v1', 'text', '[]'::jsonb, NOW() - INTERVAL '6 hours'),
( '11111111-1111-1111-1111-111111111111', 'e5f6a7b8-c9d0-1234-efab-345678901234', 'Perfect, thanks! I will check it out now.', 'text', '[]'::jsonb, NOW() - INTERVAL '2 hours');

-- Seed messages for Dev Team
INSERT INTO messages (id, room_id, user_id, content, message_type, attachments, timestamp) VALUES
('aaaa1111-aaaa-aaaa-aaaa-aaaa11111111', '22222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Team standup in 10 minutes! 🚀', 'text', '[]'::jsonb, NOW() - INTERVAL '3 days'),
('bbbb1111-bbbb-bbbb-bbbb-bbbb11111111', '22222222-2222-2222-2222-222222222222', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'On my way!', 'text', '[]'::jsonb, NOW() - INTERVAL '3 days' + INTERVAL '5 minutes'),
('cccc1111-cccc-cccc-cccc-cccc11111111', '22222222-2222-2222-2222-222222222222', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'Same here, 1 minute!', 'text', '[]'::jsonb, NOW() - INTERVAL '3 days' + INTERVAL '8 minutes'),
('dddd1111-dddd-dddd-dddd-dddd11111111', '22222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Great progress on the sprint! Lets discuss the next steps.', 'text', '[]'::jsonb, NOW() - INTERVAL '2 days'),
('eeee1111-eeee-eeee-eeee-eeee11111111', '22222222-2222-2222-2222-222222222222', 'c3d4e5f6-a7b8-9012-cdef-123456789012', 'I think we should prioritize the API optimization this week.', 'text', '[]'::jsonb, NOW() - INTERVAL '1 day 20 hours'),
('ffff1111-ffff-ffff-ffff-ffff11111111', '22222222-2222-2222-2222-222222222222', 'd4e5f6a7-b8c9-0123-defa-234567890123', 'Agreed! Also, dont forget the code review for PR #42', 'text', '[]'::jsonb, NOW() - INTERVAL '18 hours');

-- System messages for room creation
INSERT INTO messages (room_id, user_id, content, message_type, attachments, timestamp) VALUES
( '11111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Room created', 'system', '[]'::jsonb, NOW() - INTERVAL '10 days'),
( '22222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f12345678901', 'Room created', 'system', '[]'::jsonb, NOW() - INTERVAL '5 days');
