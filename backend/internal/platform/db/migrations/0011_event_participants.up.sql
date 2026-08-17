-- Участники платформы мероприятий, их независимые сессии и scoped permissions.
-- В существующей модели Contest является мероприятием, поэтому все event-связи
-- направлены на contests.id. Обычные users/contest_participants не используются.

CREATE TABLE event_participants (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    full_name            TEXT NOT NULL,
    full_name_normalized TEXT NOT NULL,
    birth_date           DATE NOT NULL,
    union_card_number    CITEXT NULL,
    sks_barcode          CITEXT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at          TIMESTAMPTZ NULL,
    CONSTRAINT chk_event_participant_name CHECK (btrim(full_name) <> ''),
    CONSTRAINT chk_event_participant_name_normalized CHECK (btrim(full_name_normalized) <> ''),
    CONSTRAINT chk_event_participant_status CHECK (status IN ('ACTIVE', 'BLOCKED', 'ARCHIVED')),
    CONSTRAINT uq_event_participant_contest_id UNIQUE (contest_id, id)
);

CREATE INDEX idx_event_participants_contest
    ON event_participants(contest_id);
CREATE INDEX idx_event_participants_name_birth
    ON event_participants(contest_id, full_name_normalized, birth_date);
CREATE INDEX idx_event_participants_status
    ON event_participants(contest_id, status);
CREATE UNIQUE INDEX uq_event_participants_union_card
    ON event_participants(contest_id, union_card_number)
    WHERE union_card_number IS NOT NULL;
CREATE UNIQUE INDEX uq_event_participants_sks_barcode
    ON event_participants(contest_id, sks_barcode)
    WHERE sks_barcode IS NOT NULL;

-- В БД хранится только SHA-256 hash opaque cookie token.
CREATE TABLE participant_sessions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL,
    event_participant_id UUID NOT NULL,
    token_hash           CHAR(64) UNIQUE NOT NULL,
    user_agent           TEXT NULL,
    ip_hash              TEXT NULL,
    last_activity_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at           TIMESTAMPTZ NOT NULL,
    revoked_at           TIMESTAMPTZ NULL,
    revoke_reason        TEXT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_participant_session_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT
);

CREATE INDEX idx_participant_sessions_active
    ON participant_sessions(token_hash, expires_at)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_participant_sessions_participant
    ON participant_sessions(event_participant_id, expires_at DESC);

-- Расширение существующего scoped RBAC без создания параллельной системы ролей.
-- MEGA_ADMIN и владелец конкурса имеют все permissions неявно; для остальных
-- сотрудников выдаются явные разрешения в рамках конкретного конкурса.
CREATE TABLE event_staff_permissions (
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id    UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    permission    VARCHAR(64) NOT NULL,
    granted_by    UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, contest_id, permission),
    CONSTRAINT chk_event_staff_permission CHECK (permission IN (
        'event.participants.manage',
        'event.attendance.scan',
        'event.attendance.manage',
        'event.tasks.manage',
        'event.tasks.moderate',
        'event.merch.manage',
        'event.merch.orders.manage',
        'event.points.manage'
    ))
);

CREATE INDEX idx_event_staff_permissions_contest
    ON event_staff_permissions(contest_id, permission);

-- Позволяет связывать audit event как со staff user, так и с участником мероприятия.
ALTER TABLE audit_logs
    ADD COLUMN event_participant_id UUID NULL
        REFERENCES event_participants(id) ON DELETE SET NULL;

CREATE INDEX idx_audit_logs_event_participant
    ON audit_logs(event_participant_id, created_at DESC)
    WHERE event_participant_id IS NOT NULL;
