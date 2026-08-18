-- Tracks/directions of an event: one per participant, many per lecture.
-- Empty lecture_directions means the lecture is for every participant (backward compatible).

CREATE TABLE event_directions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id       UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    name             TEXT NOT NULL,
    name_normalized  TEXT NOT NULL,
    sort_order       INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_directions_contest_id UNIQUE (contest_id, id),
    CONSTRAINT uq_event_directions_name UNIQUE (contest_id, name_normalized),
    CONSTRAINT chk_event_direction_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_event_direction_name_normalized CHECK (btrim(name_normalized) <> '')
);

CREATE INDEX idx_event_directions_contest
    ON event_directions (contest_id, sort_order, name);

ALTER TABLE event_participants
    ADD COLUMN direction_id UUID NULL;

ALTER TABLE event_participants
    ADD CONSTRAINT fk_event_participants_direction
        FOREIGN KEY (contest_id, direction_id)
        REFERENCES event_directions (contest_id, id)
        ON DELETE RESTRICT;

CREATE INDEX idx_event_participants_direction
    ON event_participants (contest_id, direction_id)
    WHERE direction_id IS NOT NULL;

CREATE TABLE lecture_directions (
    contest_id   UUID NOT NULL,
    lecture_id   UUID NOT NULL,
    direction_id UUID NOT NULL,
    PRIMARY KEY (lecture_id, direction_id),
    CONSTRAINT fk_lecture_directions_lecture
        FOREIGN KEY (contest_id, lecture_id)
        REFERENCES lectures (contest_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_lecture_directions_direction
        FOREIGN KEY (contest_id, direction_id)
        REFERENCES event_directions (contest_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX idx_lecture_directions_direction
    ON lecture_directions (direction_id);
