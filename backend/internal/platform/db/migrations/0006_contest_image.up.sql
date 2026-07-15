-- Обложка конкурса: ключ объекта в S3/MinIO (SITE.md §8 — файлы в объектном хранилище).
ALTER TABLE contests ADD COLUMN image_key TEXT NULL;
