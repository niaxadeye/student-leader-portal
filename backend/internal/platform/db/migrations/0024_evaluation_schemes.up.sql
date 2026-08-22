-- Схема оценивания испытания: тип двигателя, критерии, компоненты. Live-сессии — позже.
CREATE TABLE evaluation_schemes (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id       UUID NOT NULL UNIQUE REFERENCES contest_challenges (id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    type               VARCHAR(32) NOT NULL,
    scoring_unit       VARCHAR(32) NOT NULL,
    min_score          DOUBLE PRECISION NULL,
    max_score          DOUBLE PRECISION NULL,
    corridor_mode      VARCHAR(32) NOT NULL DEFAULT 'NONE',
    result_visibility  VARCHAR(32) NOT NULL DEFAULT 'ADMIN_ONLY',
    edit_policy        VARCHAR(32) NOT NULL DEFAULT 'WHILE_TRIAL_ACTIVE',
    settings_json      JSONB NOT NULL DEFAULT '{}',
    active             BOOLEAN NOT NULL DEFAULT TRUE,
    created_by         UUID NULL REFERENCES users (id),
    updated_by         UUID NULL REFERENCES users (id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE evaluation_scheme_versions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_scheme_id    UUID NOT NULL REFERENCES evaluation_schemes (id) ON DELETE CASCADE,
    version                 INT NOT NULL,
    configuration_snapshot  JSONB NOT NULL,
    created_by              UUID NULL REFERENCES users (id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (evaluation_scheme_id, version)
);

CREATE TABLE evaluation_criterion_groups (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_scheme_id UUID NOT NULL REFERENCES evaluation_schemes (id) ON DELETE CASCADE,
    title                TEXT NOT NULL,
    description          TEXT NULL,
    sort_order           INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_eval_groups_scheme ON evaluation_criterion_groups (evaluation_scheme_id, sort_order);

CREATE TABLE evaluation_criteria (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_scheme_id UUID NOT NULL REFERENCES evaluation_schemes (id) ON DELETE CASCADE,
    group_id             UUID NULL REFERENCES evaluation_criterion_groups (id) ON DELETE SET NULL,
    title                TEXT NOT NULL,
    description          TEXT NULL,
    min_score            DOUBLE PRECISION NOT NULL DEFAULT 1,
    max_score            DOUBLE PRECISION NOT NULL DEFAULT 10,
    weight               DOUBLE PRECISION NOT NULL DEFAULT 1,
    is_required          BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order           INT NOT NULL DEFAULT 0,
    active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_eval_criteria_scheme ON evaluation_criteria (evaluation_scheme_id, sort_order) WHERE active;

CREATE TABLE criterion_scale_bands (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    criterion_id UUID NOT NULL REFERENCES evaluation_criteria (id) ON DELETE CASCADE,
    min_score    DOUBLE PRECISION NOT NULL,
    max_score    DOUBLE PRECISION NOT NULL,
    description  TEXT NOT NULL,
    sort_order   INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_eval_bands_criterion ON criterion_scale_bands (criterion_id, sort_order);

CREATE TABLE evaluation_components (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_scheme_id UUID NOT NULL REFERENCES evaluation_schemes (id) ON DELETE CASCADE,
    code                 VARCHAR(64) NOT NULL,
    title                TEXT NOT NULL,
    type                 VARCHAR(32) NOT NULL,
    weight               DOUBLE PRECISION NULL,
    aggregation_method   VARCHAR(32) NOT NULL DEFAULT 'SUM',
    sort_order           INT NOT NULL DEFAULT 0,
    UNIQUE (evaluation_scheme_id, code)
);
