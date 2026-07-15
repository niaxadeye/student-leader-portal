-- Испытания, поля конструктора и версии схемы (SITE.md §21.8–21.10, Этап 3).
CREATE TABLE contest_challenges (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id             UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    title                  TEXT NOT NULL,
    slug                   CITEXT NOT NULL,
    short_description      TEXT NULL,
    full_description       TEXT NULL,
    instructions           TEXT NULL,
    status                 VARCHAR(32) NOT NULL DEFAULT 'DRAFT', -- DRAFT|PUBLISHED|CLOSED|ARCHIVED
    sort_order             INT NOT NULL DEFAULT 0,
    open_at                TIMESTAMPTZ NULL,
    deadline_at            TIMESTAMPTZ NULL,
    close_at               TIMESTAMPTZ NULL,
    settings               JSONB NOT NULL DEFAULT '{}',
    current_schema_version INT NOT NULL DEFAULT 1,
    created_by             UUID NULL REFERENCES users (id),
    updated_by             UUID NULL REFERENCES users (id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at           TIMESTAMPTZ NULL,
    archived_at            TIMESTAMPTZ NULL,
    UNIQUE (contest_id, slug)
);

CREATE INDEX idx_challenges_contest ON contest_challenges (contest_id);
CREATE INDEX idx_challenges_status ON contest_challenges (status);

-- Поля формы. Версионируются: schema_version_from/to задают окно жизни поля.
-- Физически не удаляем (soft delete) — ответы прошлых ревизий ссылаются на схему.
CREATE TABLE challenge_fields (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id        UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    field_key           CITEXT NOT NULL,
    field_type          VARCHAR(32) NOT NULL,
    label               TEXT NOT NULL,
    description         TEXT NULL,
    help_text           TEXT NULL,
    placeholder         TEXT NULL,
    required            BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order          INT NOT NULL DEFAULT 0,
    settings            JSONB NOT NULL DEFAULT '{}',
    validation          JSONB NOT NULL DEFAULT '{}',
    visibility          JSONB NOT NULL DEFAULT '{}',
    schema_version_from INT NOT NULL DEFAULT 1,
    schema_version_to   INT NULL,
    created_by          UUID NULL REFERENCES users (id),
    updated_by          UUID NULL REFERENCES users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ NULL,
    UNIQUE (challenge_id, field_key, schema_version_from)
);

CREATE INDEX idx_fields_challenge ON challenge_fields (challenge_id) WHERE deleted_at IS NULL;

-- Снапшоты схемы: создаются при публикации и при правке опубликованной формы.
CREATE TABLE challenge_schema_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id   UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    version        INT NOT NULL,
    schema_json    JSONB NOT NULL,
    change_summary TEXT NULL,
    created_by     UUID NULL REFERENCES users (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, version)
);
