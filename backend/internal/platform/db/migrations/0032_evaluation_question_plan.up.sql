-- План вопросов «2 к 1»: сколько вопросов в испытании (0 = не задано).
ALTER TABLE evaluation_sessions
    ADD COLUMN question_count INT NOT NULL DEFAULT 0;

ALTER TABLE evaluation_sessions
    ADD CONSTRAINT evaluation_sessions_question_count
    CHECK (question_count >= 0);
