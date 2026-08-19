-- Optional venue of a lecture (room, hall, building). Empty stays NULL.
ALTER TABLE lectures
    ADD COLUMN location TEXT NULL;

ALTER TABLE lectures
    ADD CONSTRAINT chk_lecture_location CHECK (location IS NULL OR btrim(location) <> '');
