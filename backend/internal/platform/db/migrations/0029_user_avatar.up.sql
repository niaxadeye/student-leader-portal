-- Портрет конкурсанта (S3 object key). Нужен для live / 2к1.
ALTER TABLE users
    ADD COLUMN avatar_key TEXT NULL;
