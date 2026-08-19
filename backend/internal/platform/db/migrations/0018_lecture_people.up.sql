-- Speakers and moderators of a lecture. Names are free text: guests need not be participants.
CREATE TABLE lecture_people (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id  UUID NOT NULL,
    lecture_id  UUID NOT NULL,
    role        VARCHAR(16) NOT NULL,
    name        TEXT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    CONSTRAINT fk_lecture_people_lecture
        FOREIGN KEY (contest_id, lecture_id)
        REFERENCES lectures (contest_id, id)
        ON DELETE CASCADE,
    CONSTRAINT chk_lecture_people_role CHECK (role IN ('SPEAKER', 'MODERATOR')),
    CONSTRAINT chk_lecture_people_name CHECK (btrim(name) <> '')
);

CREATE INDEX idx_lecture_people_lecture
    ON lecture_people (lecture_id, role, sort_order);
