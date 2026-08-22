-- Live-состояние испытания: сессия, выступления, шаблоны фаз.
CREATE TABLE performances (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id         UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id   UUID NOT NULL REFERENCES users (id),
    sequence_number      INT NULL,
    status               VARCHAR(32) NOT NULL DEFAULT 'PLANNED',
    started_at           TIMESTAMPTZ NULL,
    finished_at          TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id)
);

CREATE INDEX idx_performances_challenge ON performances (challenge_id, sequence_number);

CREATE TABLE performance_phase_templates (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_scheme_id UUID NOT NULL REFERENCES evaluation_schemes (id) ON DELETE CASCADE,
    title                TEXT NOT NULL,
    duration_seconds     INT NULL,
    scoring_allowed      BOOLEAN NOT NULL DEFAULT FALSE,
    maps_to_state        VARCHAR(32) NOT NULL,
    sort_order           INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_eval_phase_templates ON performance_phase_templates (evaluation_scheme_id, sort_order);

CREATE TABLE evaluation_sessions (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id                UUID NOT NULL UNIQUE REFERENCES contest_challenges (id) ON DELETE CASCADE,
    current_performance_id      UUID NULL REFERENCES performances (id) ON DELETE SET NULL,
    current_contestant_user_id  UUID NULL REFERENCES users (id),
    current_match_id            UUID NULL,
    state                       VARCHAR(32) NOT NULL DEFAULT 'NOT_STARTED',
    current_phase_id            UUID NULL REFERENCES performance_phase_templates (id) ON DELETE SET NULL,
    started_at                  TIMESTAMPTZ NULL,
    state_changed_at            TIMESTAMPTZ NULL,
    finished_at                 TIMESTAMPTZ NULL,
    controlled_by               UUID NULL REFERENCES users (id),
    revision                    INT NOT NULL DEFAULT 0,
    phase_started_at            TIMESTAMPTZ NULL,
    phase_duration_seconds      INT NULL,
    paused_at                   TIMESTAMPTZ NULL,
    accumulated_pause_seconds   DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);
