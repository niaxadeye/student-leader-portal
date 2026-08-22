-- Материалы испытания для кабинета конкурсанта: общий текст/файлы и персональные выдачи.
CREATE TABLE challenge_briefings (
    challenge_id UUID PRIMARY KEY REFERENCES contest_challenges (id) ON DELETE CASCADE,
    body_text    TEXT NOT NULL DEFAULT '',
    publish_at   TIMESTAMPTZ NULL,
    created_by   UUID NULL REFERENCES users (id),
    updated_by   UUID NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE challenge_briefing_overrides (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id        UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    custom_text         BOOLEAN NOT NULL DEFAULT FALSE,
    body_text           TEXT NOT NULL DEFAULT '',
    custom_publish      BOOLEAN NOT NULL DEFAULT FALSE,
    publish_at          TIMESTAMPTZ NULL,
    hidden              BOOLEAN NOT NULL DEFAULT FALSE,
    replace_files       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by          UUID NULL REFERENCES users (id),
    updated_by          UUID NULL REFERENCES users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id)
);

CREATE INDEX idx_briefing_overrides_challenge ON challenge_briefing_overrides (challenge_id);

CREATE TABLE challenge_briefing_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    override_id  UUID NULL REFERENCES challenge_briefing_overrides (id) ON DELETE CASCADE,
    file_id      UUID NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_briefing_files_challenge ON challenge_briefing_files (challenge_id, override_id);
CREATE UNIQUE INDEX idx_briefing_files_default
    ON challenge_briefing_files (challenge_id, file_id)
    WHERE override_id IS NULL;
CREATE UNIQUE INDEX idx_briefing_files_override
    ON challenge_briefing_files (override_id, file_id)
    WHERE override_id IS NOT NULL;
