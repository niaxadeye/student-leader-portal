-- Ключ «да/нет» на вопрос и отметки жюри по ответам конкурсантов.
CREATE TABLE evaluation_question_keys (
    challenge_id     UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    question_number  INT NOT NULL,
    correct_answer   VARCHAR(8) NOT NULL,
    set_by_user_id   UUID NOT NULL REFERENCES users (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (challenge_id, question_number),
    CONSTRAINT evaluation_question_keys_number CHECK (question_number >= 1),
    CONSTRAINT evaluation_question_keys_answer CHECK (correct_answer IN ('YES', 'NO'))
);

CREATE TABLE life_answer_marks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id         UUID NOT NULL REFERENCES contest_challenges (id) ON DELETE CASCADE,
    contestant_user_id   UUID NOT NULL REFERENCES users (id),
    jury_user_id         UUID NOT NULL REFERENCES users (id),
    question_number      INT NOT NULL,
    answer               VARCHAR(8) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (challenge_id, contestant_user_id, jury_user_id, question_number),
    CONSTRAINT life_answer_marks_number CHECK (question_number >= 1),
    CONSTRAINT life_answer_marks_answer CHECK (answer IN ('YES', 'NO'))
);

CREATE INDEX idx_life_answer_marks_challenge ON life_answer_marks (challenge_id, question_number);
CREATE INDEX idx_life_answer_marks_jury ON life_answer_marks (challenge_id, jury_user_id);
