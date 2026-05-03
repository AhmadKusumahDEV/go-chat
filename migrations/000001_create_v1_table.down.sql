-- File: xxxxxxxxxx_create_chat_schema.down.sql

-- 6. Hapus direct_rooms (referensi ke rooms dan users)
DROP TABLE IF EXISTS direct_rooms;

-- 5. Hapus message_status (referensi ke messages dan users)
DROP TABLE IF EXISTS message_status;

-- 4. Hapus messages (referensi ke rooms dan users)
DROP TABLE IF EXISTS messages;

-- 3. Hapus room_members (referensi ke rooms dan users)
DROP TABLE IF EXISTS room_members;

-- 2. Hapus rooms (referensi ke users)
DROP TABLE IF EXISTS rooms;

-- 1. Hapus users (tabel dasar)
DROP TABLE IF EXISTS users;

DROP INDEX IF EXISTS idx_rooms_name_trgm;