-- Расширения и таблица пользователей (SITE.md §21.1).
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;     -- регистронезависимый email

CREATE TABLE users (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login                VARCHAR(255) UNIQUE NOT NULL,
    password_hash        TEXT NOT NULL,
    full_name            TEXT NOT NULL,
    email                CITEXT NULL,
    phone                TEXT NULL,
    organization         TEXT NULL,
    city                 TEXT NULL,
    status               VARCHAR(32) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE | BLOCKED
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    failed_login_count   INT NOT NULL DEFAULT 0,
    locked_until         TIMESTAMPTZ NULL,
    last_login_at        TIMESTAMPTZ NULL,
    password_changed_at  TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ NULL
);

CREATE INDEX idx_users_login ON users (login);
CREATE INDEX idx_users_email ON users (email);
