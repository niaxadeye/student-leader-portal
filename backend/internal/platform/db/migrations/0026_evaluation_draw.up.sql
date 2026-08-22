-- Порядок выступлений (жеребьёвка) на конкретное испытание.
CREATE TABLE evaluation_draw_entries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id       UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id UUID NOT NULL REFERENCES users (id),
    draw_number        INT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id),
    UNIQUE (challenge_id, draw_number)
);

CREATE INDEX idx_eval_draw_challenge ON evaluation_draw_entries (challenge_id, draw_number);
