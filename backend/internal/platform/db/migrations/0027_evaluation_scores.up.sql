-- Баллы жюри: лист на выступление × оценщик, значение на критерий, история правок.
CREATE TABLE score_sheets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    performance_id      UUID NOT NULL REFERENCES performances (id) ON DELETE CASCADE,
    evaluator_user_id   UUID NOT NULL REFERENCES users (id),
    total_score_cache   DOUBLE PRECISION NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (performance_id, evaluator_user_id)
);

CREATE INDEX idx_score_sheets_evaluator ON score_sheets (evaluator_user_id, updated_at DESC);

CREATE TABLE score_values (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    score_sheet_id     UUID NOT NULL REFERENCES score_sheets (id) ON DELETE CASCADE,
    criterion_id       UUID NOT NULL REFERENCES evaluation_criteria (id),
    score              DOUBLE PRECISION NOT NULL,
    comment            TEXT NULL,
    revision           INT NOT NULL DEFAULT 1,
    last_mutation_id   UUID NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (score_sheet_id, criterion_id)
);

CREATE UNIQUE INDEX idx_score_values_mutation ON score_values (last_mutation_id) WHERE last_mutation_id IS NOT NULL;
CREATE INDEX idx_score_values_sheet ON score_values (score_sheet_id);

CREATE TABLE score_value_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    score_value_id  UUID NOT NULL REFERENCES score_values (id) ON DELETE CASCADE,
    score           DOUBLE PRECISION NOT NULL,
    comment         TEXT NULL,
    revision        INT NOT NULL,
    mutation_id     UUID NULL,
    actor_user_id   UUID NOT NULL REFERENCES users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_score_history_value ON score_value_history (score_value_id, created_at);
