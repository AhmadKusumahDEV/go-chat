-- 1. Hapus tabel baru jika migrasi dibatalkan (index otomatis ikut terhapus)
DROP TABLE IF EXISTS message_attachments;

-- 2. Kembalikan kolom attachments JSONB ke tabel messages
ALTER TABLE messages ADD COLUMN attachments JSONB DEFAULT '[]'::jsonb;