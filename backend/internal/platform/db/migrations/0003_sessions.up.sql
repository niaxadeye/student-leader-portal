-- Сессии и refresh-токены (SITE.md §21.4–21.5, §16).
CREATE TABLE auth_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_family_id UUID NOT NULL,
    user_agent      TEXT NULL,
    ip_hash         TEXT NULL,
    last_used_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ NULL,
    revoke_reason   TEXT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_sessions_user ON auth_sessions (user_id, revoked_at, expires_at);

CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES auth_sessions (id) ON DELETE CASCADE,
    jti             UUID UNIQUE NOT NULL,
    token_hash      TEXT NOT NULL,
    rotated_from_id UUID NULL REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ NULL,
    revoked_at      TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_session ON refresh_tokens (session_id, revoked_at, expires_at);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash);
