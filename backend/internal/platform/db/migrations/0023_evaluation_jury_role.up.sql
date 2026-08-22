-- Роль жюри (CONTEST-scope через user_roles) и назначения операторов испытания.
INSERT INTO roles (code, name) VALUES ('JURY', 'Член жюри')
    ON CONFLICT (code) DO NOTHING;

CREATE TABLE evaluation_staff_assignments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id   UUID NOT NULL REFERENCES contests (id) ON DELETE CASCADE,
    challenge_id UUID NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         VARCHAR(32) NOT NULL, -- JURY|TRIAL_OPERATOR|EXPERT|FOCUS_GROUP_ADMIN
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_eval_staff_contest_role
    ON evaluation_staff_assignments (contest_id, user_id, role)
    WHERE challenge_id IS NULL AND active;
CREATE UNIQUE INDEX idx_eval_staff_challenge_role
    ON evaluation_staff_assignments (challenge_id, user_id, role)
    WHERE challenge_id IS NOT NULL AND active;
CREATE INDEX idx_eval_staff_user ON evaluation_staff_assignments (user_id) WHERE active;
