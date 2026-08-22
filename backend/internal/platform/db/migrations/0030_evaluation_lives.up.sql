-- «2 к 1»: номер текущего вопроса и журнал жизней (append-only).
ALTER TABLE evaluation_sessions
    ADD COLUMN current_question_number INT NOT NULL DEFAULT 1;

ALTER TABLE evaluation_sessions
    ADD CONSTRAINT evaluation_sessions_question_positive
    CHECK (current_question_number >= 1);

CREATE TABLE life_events (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id            UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id      UUID NOT NULL REFERENCES users (id),
    question_number         INT NOT NULL,
    delta                   INT NOT NULL,
    reason                  VARCHAR(64) NOT NULL,
    created_by_user_id      UUID NOT NULL REFERENCES users (id),
    reverses_life_event_id  UUID NULL REFERENCES life_events (id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT life_events_question_positive CHECK (question_number >= 1),
    CONSTRAINT life_events_delta CHECK (delta IN (-1, 1))
);

CREATE INDEX idx_life_events_challenge ON life_events (challenge_id, created_at);
CREATE INDEX idx_life_events_contestant ON life_events (challenge_id, contestant_user_id, created_at);
CREATE INDEX idx_life_events_author ON life_events (challenge_id, created_by_user_id, created_at);
CREATE INDEX idx_life_events_reverse ON life_events (reverses_life_event_id)
    WHERE reverses_life_event_id IS NOT NULL;
