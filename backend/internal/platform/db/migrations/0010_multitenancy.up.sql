-- Мульти-арендность: над-роль MEGA_ADMIN, владение данными, per-contest доступ,
-- персональные Telegram-уведомления. См. docs/RBAC_MULTITENANCY.md.
-- ВНИМАНИЕ: этап допускает сброс тестовых данных (согласовано с заказчиком).

-- 1. Над-роль платформы.
INSERT INTO roles (code, name) VALUES ('MEGA_ADMIN', 'Мегаадминистратор')
    ON CONFLICT (code) DO NOTHING;

-- 2. Уровень доступа на назначении ADMIN→конкурс (EDIT|VIEW). NULL для GLOBAL/не-ADMIN.
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS access_level VARCHAR(8) NULL;

-- 3. Владение (основа изоляции). created_by — кто завёл пользователя;
--    owner_user_id — организатор-владелец конкурса.
ALTER TABLE users    ADD COLUMN IF NOT EXISTS created_by UUID NULL REFERENCES users(id);
ALTER TABLE users    ADD COLUMN IF NOT EXISTS org_name   TEXT NULL;
ALTER TABLE contests ADD COLUMN IF NOT EXISTS owner_user_id UUID NULL REFERENCES users(id);
CREATE INDEX IF NOT EXISTS idx_users_created_by ON users(created_by);
CREATE INDEX IF NOT EXISTS idx_contests_owner   ON contests(owner_user_id);

-- 4. Персональная привязка Telegram (админ/суперадмин/мега). O8: chat_id глобально уникален.
CREATE TABLE IF NOT EXISTS user_telegram (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    chat_id     TEXT NULL,
    link_token  TEXT NULL,
    linked_at   TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_telegram_chat
    ON user_telegram(chat_id) WHERE chat_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_telegram_token
    ON user_telegram(link_token) WHERE link_token IS NOT NULL;

-- 5. Подписки на уведомления (O2, opt-out): строка появляется только при ЯВНОМ отключении.
CREATE TABLE IF NOT EXISTS notification_subscriptions (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, contest_id)
);

-- 6. Доставки outbox (O3): строка на получателя, независимые ретраи/DEAD.
CREATE TABLE IF NOT EXISTS outbox_deliveries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL REFERENCES outbox_events(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id      TEXT NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING|SENT|DEAD
    attempts     INT NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at    TIMESTAMPTZ NULL,
    locked_by    TEXT NULL,
    last_error   TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_deliveries_claim
    ON outbox_deliveries(available_at) WHERE status = 'PENDING';

-- 7. Флаг «событие разослано» (fan-out выполнен): дальше диспетчер поллит deliveries.
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS fanned_out_at TIMESTAMPTZ NULL;
