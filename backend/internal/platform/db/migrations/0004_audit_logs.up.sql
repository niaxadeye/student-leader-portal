-- Append-only аудит (SITE.md §21.17).
CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    action        VARCHAR(64) NOT NULL,
    entity_type   VARCHAR(64) NOT NULL,
    entity_id     UUID NULL,
    contest_id    UUID NULL,
    request_id    TEXT NULL,
    ip_hash       TEXT NULL,
    user_agent    TEXT NULL,
    before_json   JSONB NULL,
    after_json    JSONB NULL,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_contest ON audit_logs (contest_id, created_at DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_user_id, created_at DESC);
