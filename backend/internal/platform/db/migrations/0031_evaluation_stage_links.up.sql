-- Связь заочного этапа с основным испытанием: веса и способ сведения рейтинга.
CREATE TABLE evaluation_stage_links (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    main_challenge_id    UUID NOT NULL UNIQUE REFERENCES contest_challenges (id) ON DELETE CASCADE,
    remote_challenge_id  UUID NOT NULL UNIQUE REFERENCES contest_challenges (id) ON DELETE CASCADE,
    main_weight          NUMERIC(12, 4) NOT NULL DEFAULT 1,
    remote_weight        NUMERIC(12, 4) NOT NULL DEFAULT 1,
    combine_mode         VARCHAR(16) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT evaluation_stage_links_distinct CHECK (main_challenge_id <> remote_challenge_id),
    CONSTRAINT evaluation_stage_links_weights CHECK (main_weight > 0 AND remote_weight > 0),
    CONSTRAINT evaluation_stage_links_mode CHECK (combine_mode IN ('RANK_SUM', 'SCORE_SUM'))
);

CREATE INDEX idx_eval_stage_links_contest ON evaluation_stage_links (contest_id);
