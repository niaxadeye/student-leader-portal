-- Event tasks, immutable submission attempts and private evidence assets.
-- Contest remains the existing event aggregate; EventParticipant is not a User.

CREATE TABLE event_tasks (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id               UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    title                    TEXT NOT NULL,
    description              TEXT NOT NULL,
    image_key                TEXT NULL,
    icon                     TEXT NULL,
    points                   BIGINT NOT NULL,
    starts_at                TIMESTAMPTZ NULL,
    ends_at                  TIMESTAMPTZ NULL,
    status                   VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    sort_order               INT NOT NULL DEFAULT 0,
    allowed_submission_types TEXT[] NOT NULL DEFAULT ARRAY['IMAGE','LINK']::TEXT[],
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_tasks_contest_id UNIQUE (contest_id, id),
    CONSTRAINT chk_event_task_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_event_task_description CHECK (btrim(description) <> ''),
    CONSTRAINT chk_event_task_points CHECK (points > 0),
    CONSTRAINT chk_event_task_status CHECK (status IN ('DRAFT','ACTIVE','DISABLED','ARCHIVED')),
    CONSTRAINT chk_event_task_schedule CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at),
    CONSTRAINT chk_event_task_submission_types CHECK (
        cardinality(allowed_submission_types) > 0
        AND allowed_submission_types <@ ARRAY['IMAGE','LINK']::TEXT[]
    )
);

CREATE INDEX idx_event_tasks_contest_sort
    ON event_tasks(contest_id, sort_order, created_at);
CREATE INDEX idx_event_tasks_contest_status
    ON event_tasks(contest_id, status, starts_at, ends_at);

-- One logical submission per task + participant. Current moderation state is copied
-- here for efficient lists; every immutable attempt remains in the attempts table.
CREATE TABLE event_task_submissions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL,
    task_id               UUID NOT NULL,
    event_participant_id  UUID NOT NULL,
    status                VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    current_attempt       INT NOT NULL DEFAULT 0,
    participant_comment   TEXT NULL,
    moderator_comment     TEXT NULL,
    reviewed_by_user_id   UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at          TIMESTAMPTZ NULL,
    reviewed_at           TIMESTAMPTZ NULL,
    reward_granted_at     TIMESTAMPTZ NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_event_task_submission_task
        FOREIGN KEY (contest_id, task_id)
        REFERENCES event_tasks(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_event_task_submission_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_event_task_submission UNIQUE (task_id, event_participant_id),
    CONSTRAINT uq_event_task_submission_contest_id UNIQUE (contest_id, id),
    CONSTRAINT chk_event_task_submission_status CHECK (status IN ('PENDING','APPROVED','REJECTED')),
    CONSTRAINT chk_event_task_submission_attempt CHECK (current_attempt >= 0),
    CONSTRAINT chk_event_task_submission_reward CHECK (
        (status='APPROVED' AND reward_granted_at IS NOT NULL)
        OR (status<>'APPROVED' AND reward_granted_at IS NULL)
    )
);

CREATE INDEX idx_event_task_submissions_moderation
    ON event_task_submissions(contest_id, status, submitted_at DESC);
CREATE INDEX idx_event_task_submissions_participant
    ON event_task_submissions(event_participant_id, updated_at DESC);

CREATE TABLE event_task_submission_attempts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL,
    submission_id         UUID NOT NULL,
    attempt_number        INT NOT NULL,
    status                VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    participant_comment   TEXT NULL,
    moderator_comment     TEXT NULL,
    reviewed_by_user_id   UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at           TIMESTAMPTZ NULL,
    CONSTRAINT fk_event_task_attempt_submission
        FOREIGN KEY (contest_id, submission_id)
        REFERENCES event_task_submissions(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_event_task_attempt UNIQUE (submission_id, attempt_number),
    CONSTRAINT uq_event_task_attempt_contest_id UNIQUE (contest_id, id),
    CONSTRAINT chk_event_task_attempt_number CHECK (attempt_number > 0),
    CONSTRAINT chk_event_task_attempt_status CHECK (status IN ('PENDING','APPROVED','REJECTED')),
    CONSTRAINT chk_event_task_attempt_review CHECK (
        (status='PENDING' AND reviewed_at IS NULL AND reviewed_by_user_id IS NULL)
        OR (status<>'PENDING' AND reviewed_at IS NOT NULL AND reviewed_by_user_id IS NOT NULL)
    )
);

CREATE INDEX idx_event_task_attempts_submission
    ON event_task_submission_attempts(submission_id, attempt_number DESC);

-- IMAGE objects stay in the private S3 bucket; LINK keeps the participant supplied URL.
CREATE TABLE event_task_submission_assets (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id     UUID NOT NULL,
    attempt_id     UUID NOT NULL,
    type           VARCHAR(16) NOT NULL,
    object_key     TEXT NULL,
    external_url   TEXT NULL,
    original_name  TEXT NULL,
    mime_type      TEXT NULL,
    size_bytes     BIGINT NULL,
    sort_order     INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_event_task_asset_attempt
        FOREIGN KEY (contest_id, attempt_id)
        REFERENCES event_task_submission_attempts(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_event_task_asset_type CHECK (type IN ('IMAGE','LINK')),
    CONSTRAINT chk_event_task_asset_payload CHECK (
        (type='IMAGE' AND object_key IS NOT NULL AND external_url IS NULL
          AND original_name IS NOT NULL AND size_bytes IS NOT NULL AND size_bytes > 0)
        OR
        (type='LINK' AND external_url IS NOT NULL AND object_key IS NULL
          AND original_name IS NULL AND size_bytes IS NULL)
    )
);

CREATE INDEX idx_event_task_assets_attempt
    ON event_task_submission_assets(attempt_id, sort_order, created_at);
CREATE UNIQUE INDEX uq_event_task_asset_object_key
    ON event_task_submission_assets(object_key) WHERE object_key IS NOT NULL;
