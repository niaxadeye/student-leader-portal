-- Дата/время и место проведения испытания + флаг приёма файлов и ТЗ.
ALTER TABLE contest_challenges
    ADD COLUMN held_at TIMESTAMPTZ NULL,
    ADD COLUMN venue TEXT NULL,
    ADD COLUMN accepts_submissions BOOLEAN NOT NULL DEFAULT TRUE;
