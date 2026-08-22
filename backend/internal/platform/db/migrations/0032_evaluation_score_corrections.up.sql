-- Журнал правок баллов мегаадмином: append-only, с обязательной причиной.
CREATE TABLE evaluation_score_corrections (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    challenge_id         UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    actor_user_id        UUID NOT NULL REFERENCES users (id),
    contestant_user_id   UUID NOT NULL REFERENCES users (id),
    jury_user_id         UUID NULL REFERENCES users (id),
    criterion_id         UUID NULL REFERENCES evaluation_criteria (id) ON DELETE SET NULL,
    criterion_title      TEXT NOT NULL DEFAULT '',
    kind                 VARCHAR(16) NOT NULL,
    old_score            DOUBLE PRECISION NULL,
    new_score            DOUBLE PRECISION NULL,
    reason               TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_score_corrections_kind CHECK (kind IN ('CRITERION', 'NUMERIC')),
    CONSTRAINT evaluation_score_corrections_reason CHECK (char_length(btrim(reason)) >= 5)
);

CREATE INDEX idx_eval_score_corrections_challenge
    ON evaluation_score_corrections (challenge_id, created_at DESC);
