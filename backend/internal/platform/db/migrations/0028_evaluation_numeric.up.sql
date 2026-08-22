-- Числовой результат: один балл на конкурсанта, выставляет администратор испытания.
CREATE TABLE evaluation_numeric_results (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id         UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    score                DOUBLE PRECISION NOT NULL,
    created_by           UUID NULL REFERENCES users (id),
    updated_by           UUID NULL REFERENCES users (id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id)
);

CREATE INDEX idx_eval_numeric_challenge ON evaluation_numeric_results (challenge_id);
