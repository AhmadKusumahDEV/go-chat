-- File: xxxxxxxxxx_create_chat_schema.up.

--type for room 
CREATE TYPE room_type_enum AS ENUM (
    'group', 
    'direct', 
    'channel'
);

-- type for message 
CREATE TYPE message_type_enum AS ENUM (
    'text', 
    'image', 
    'file',
    'system'
);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 1. Tabel utama untuk pengguna
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    avatar_url TEXT DEFAULT NULL,
    provider_name VARCHAR(20) DEFAULT NULL,
    provider_id VARCHAR(100) DEFAULT NULL,

    CONSTRAINT users_provider_required CHECK (
        provider_name IS NULL = (provider_id IS NULL)
    )
);

-- 2. Tabel untuk grup chat, channel, atau DM
CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_name VARCHAR(100),
    description TEXT,
    room_type room_type_enum NOT NULL DEFAULT 'group',
    is_private BOOLEAN DEFAULT FALSE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Tabel penghubung antara users dan rooms
CREATE TABLE room_members (
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    added_by UUID REFERENCES users(id),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    role VARCHAR(20) DEFAULT 'member',
    PRIMARY KEY (room_id, user_id)
);

-- 4. Tabel untuk menyimpan semua pesan
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    message_type message_type_enum DEFAULT 'text',
    reply_to UUID REFERENCES messages(id), -- untuk threaded replies
    attachments JSONB, -- menyimpan metadata file
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(128) NOT NULL UNIQUE,
    provider VARCHAR(30) NOT NULL,
    verifier VARCHAR(128) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT state_min_length CHECK (char_length(state) >= 32)
);

CREATE INDEX idx_oauth_state ON oauth_states(state);
-- pg_cron requires the pg_cron extension to be installed on your PostgreSQL server.
-- If you want to use it, run: CREATE EXTENSION IF NOT EXISTS pg_cron;
-- Otherwise, handle cleanup from your Go application (background goroutine).
-- GRANT USAGE ON SCHEMA pg_cron TO your_app_user;

-- SELECT cron.schedule(
--     'cleanup-oauth-states',
--     '*/5 * * * *',
--     $$DELETE FROM oauth_states WHERE expires_at < CURRENT_TIMESTAMP$$
-- );

-- 5. Tabel BARU untuk status pesan (read receipts)
-- CREATE TABLE message_status (
--     message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
--     user_id UUID REFERENCES users(id) ON DELETE CASCADE,
--     status VARCHAR(20) DEFAULT 'sent', -- 'sent', 'delivered', 'read'
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     PRIMARY KEY (message_id, user_id)
-- );

-- 6. Tabel BARU untuk menandai DM (chat 1-lawan-1)
-- CREATE TABLE direct_rooms (
--     room_id UUID PRIMARY KEY REFERENCES rooms(id) ON DELETE CASCADE,
--     user1_id UUID REFERENCES users(id),
--     user2_id UUID REFERENCES users(id),
--     UNIQUE(user1_id, user2_id)
-- );

CREATE INDEX idx_rooms_created_by ON rooms(created_by);
CREATE INDEX idx_rooms_name_trgm ON rooms USING GIN (room_name gin_trgm_ops);