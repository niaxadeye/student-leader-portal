-- Lectures, short-lived participant QR codes and immutable attendance history.
-- Contest is the existing event aggregate; every cross-entity FK is contest-scoped.

CREATE TABLE lectures (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    title                 TEXT NOT NULL,
    description           TEXT NULL,
    points                BIGINT NOT NULL,
    starts_at             TIMESTAMPTZ NULL,
    ends_at               TIMESTAMPTZ NULL,
    attendance_starts_at  TIMESTAMPTZ NULL,
    attendance_ends_at    TIMESTAMPTZ NULL,
    status                VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_lectures_contest_id UNIQUE (contest_id, id),
    CONSTRAINT chk_lecture_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_lecture_points CHECK (points > 0),
    CONSTRAINT chk_lecture_status CHECK (status IN ('DRAFT', 'ACTIVE', 'FINISHED')),
    CONSTRAINT chk_lecture_schedule CHECK (
        starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at
    ),
    CONSTRAINT chk_lecture_attendance_window CHECK (
        attendance_starts_at IS NULL OR attendance_ends_at IS NULL
        OR attendance_starts_at < attendance_ends_at
    )
);

CREATE INDEX idx_lectures_contest_schedule
    ON lectures(contest_id, starts_at, created_at);
CREATE INDEX idx_lectures_contest_status
    ON lectures(contest_id, status);

-- Token payload contains only a random nonce and expiration. The participant mapping
-- stays server-side; only SHA-256(nonce) is persisted.
CREATE TABLE participant_qr_codes (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL,
    event_participant_id  UUID NOT NULL,
    nonce_hash            CHAR(64) NOT NULL UNIQUE,
    expires_at            TIMESTAMPTZ NOT NULL,
    used_at               TIMESTAMPTZ NULL,
    used_for_lecture_id   UUID NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_participant_qr_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_participant_qr_lecture
        FOREIGN KEY (contest_id, used_for_lecture_id)
        REFERENCES lectures(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_participant_qr_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_participant_qr_use_pair CHECK (
        (used_at IS NULL AND used_for_lecture_id IS NULL)
        OR (used_at IS NOT NULL AND used_for_lecture_id IS NOT NULL)
    )
);

CREATE INDEX idx_participant_qr_lookup
    ON participant_qr_codes(nonce_hash, expires_at);
CREATE INDEX idx_participant_qr_participant
    ON participant_qr_codes(event_participant_id, expires_at DESC);

CREATE TABLE lecture_attendance (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL,
    lecture_id            UUID NOT NULL,
    event_participant_id  UUID NOT NULL,
    scanned_by_user_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    scanner_type          VARCHAR(16) NOT NULL,
    points_awarded        BIGINT NOT NULL,
    qr_nonce_hash         CHAR(64) NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_lecture_attendance_lecture
        FOREIGN KEY (contest_id, lecture_id)
        REFERENCES lectures(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_lecture_attendance_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_lecture_attendance UNIQUE (lecture_id, event_participant_id),
    CONSTRAINT uq_lecture_attendance_nonce UNIQUE (qr_nonce_hash),
    CONSTRAINT chk_lecture_attendance_scanner CHECK (scanner_type IN ('CAMERA', 'USB', 'MANUAL')),
    CONSTRAINT chk_lecture_attendance_points CHECK (points_awarded > 0)
);

CREATE INDEX idx_lecture_attendance_lecture_created
    ON lecture_attendance(lecture_id, created_at DESC);
CREATE INDEX idx_lecture_attendance_participant_created
    ON lecture_attendance(event_participant_id, created_at DESC);
