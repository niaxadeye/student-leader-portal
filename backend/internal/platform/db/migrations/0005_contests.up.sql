-- Конкурсы и участники (SITE.md §21.6–21.7, Этап 2).
CREATE TABLE contests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        CITEXT UNIQUE NOT NULL,
    description TEXT NULL,
    status      VARCHAR(32) NOT NULL DEFAULT 'DRAFT', -- DRAFT|ACTIVE|FINISHED|ARCHIVED
    start_at    TIMESTAMPTZ NULL,
    end_at      TIMESTAMPTZ NULL,
    timezone    TEXT NOT NULL DEFAULT 'Europe/Moscow',
    settings    JSONB NOT NULL DEFAULT '{}',
    created_by  UUID NULL REFERENCES users (id),
    updated_by  UUID NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_contests_status ON contests (status);

-- Универсальная таблица участников (конкурсанты, жюри, сотрудники).
CREATE TABLE contest_participants (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id       UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    participant_type VARCHAR(32) NOT NULL DEFAULT 'CONTESTANT', -- CONTESTANT|PARTICIPANT|STAFF|JURY
    participant_code CITEXT NULL,
    metadata         JSONB NOT NULL DEFAULT '{}',
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at          TIMESTAMPTZ NULL,
    UNIQUE (contest_id, user_id)
);

CREATE INDEX idx_participants_contest ON contest_participants (contest_id);
CREATE INDEX idx_participants_user ON contest_participants (user_id);
