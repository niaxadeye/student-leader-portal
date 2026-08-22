-- Одно предъявленное refresh-звено может породить не более одного следующего
-- звена даже при ошибке в прикладной логике ротации.
CREATE UNIQUE INDEX uq_refresh_tokens_rotated_from
    ON refresh_tokens (rotated_from_id)
    WHERE rotated_from_id IS NOT NULL;
