-- Durable idempotency receipts for local-first jury score synchronization.
-- A receipt is intentionally independent from score_values: destructive result
-- resets must not allow an old offline mutation to resurrect a deleted score.
CREATE TABLE evaluation_score_mutations (
    mutation_id       UUID PRIMARY KEY,
    challenge_id      UUID NOT NULL,
    performance_id    UUID NOT NULL,
    evaluator_user_id UUID NOT NULL,
    criterion_id      UUID NOT NULL,
    score             DOUBLE PRECISION NOT NULL,
    score_sheet_id    UUID NULL,
    score_value_id    UUID NULL,
    revision          INT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at        TIMESTAMPTZ NULL
);

CREATE INDEX idx_evaluation_score_mutations_context
    ON evaluation_score_mutations (challenge_id, evaluator_user_id, created_at DESC);
