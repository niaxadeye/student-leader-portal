-- Подача ответов конкурсантом: работы, immutable-ревизии, файлы (SITE.md §21.11–21.14, Этап 4).

-- files — метаданные объектов в MinIO. Пишется до привязки к submission (черновик).
CREATE TABLE files (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id  UUID NOT NULL REFERENCES users (id),
    contest_id     UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    challenge_id   UUID NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    submission_id  UUID NULL,
    field_id       UUID NULL,
    bucket         TEXT NOT NULL,
    object_key     TEXT NOT NULL UNIQUE,
    original_name  TEXT NOT NULL,
    safe_name      TEXT NOT NULL,
    extension      TEXT NULL,
    mime_type      TEXT NULL,
    size_bytes     BIGINT NULL,
    checksum       TEXT NULL,
    status         VARCHAR(32) NOT NULL DEFAULT 'READY', -- READY|DELETED
    metadata       JSONB NOT NULL DEFAULT '{}',
    uploaded_at    TIMESTAMPTZ NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ NULL
);
CREATE INDEX idx_files_owner ON files (owner_user_id) WHERE deleted_at IS NULL;

-- submissions — одна работа конкурсанта по испытанию (UNIQUE на пару).
CREATE TABLE submissions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id            UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id      UUID NOT NULL REFERENCES users (id),
    status                  VARCHAR(32) NOT NULL DEFAULT 'DRAFT', -- DRAFT|SUBMITTED|LOCKED
    answers_json            JSONB NOT NULL DEFAULT '{}',
    schema_version          INT NOT NULL,
    version                 INT NOT NULL DEFAULT 1,
    current_revision_number INT NOT NULL DEFAULT 0,
    first_opened_at         TIMESTAMPTZ NULL,
    last_saved_at           TIMESTAMPTZ NULL,
    submitted_at            TIMESTAMPTZ NULL,
    last_resubmitted_at     TIMESTAMPTZ NULL,
    locked_at               TIMESTAMPTZ NULL,
    locked_by               UUID NULL REFERENCES users (id),
    lock_reason             TEXT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id)
);
CREATE INDEX idx_submissions_challenge ON submissions (challenge_id);
CREATE INDEX idx_submissions_contestant ON submissions (contestant_user_id);

-- submission_revisions — immutable-снимок при каждой отправке/обновлении.
CREATE TABLE submission_revisions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id    UUID NOT NULL REFERENCES submissions (id) ON DELETE CASCADE,
    revision_number  INT NOT NULL,
    action_type      VARCHAR(32) NOT NULL, -- SUBMIT|RESUBMIT
    schema_version   INT NOT NULL,
    schema_snapshot  JSONB NOT NULL,
    answers_snapshot JSONB NOT NULL,
    files_snapshot   JSONB NOT NULL,
    checksum         TEXT NOT NULL,
    created_by       UUID NULL REFERENCES users (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (submission_id, revision_number)
);

-- submission_files — привязка файлов к работе и полю.
CREATE TABLE submission_files (
    submission_id UUID NOT NULL REFERENCES submissions (id) ON DELETE CASCADE,
    file_id       UUID NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    field_id      UUID NULL,
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (submission_id, file_id)
);
CREATE INDEX idx_submission_files_sub ON submission_files (submission_id);
