-- Optional public VK / Telegram profile links for event participants.
ALTER TABLE event_participants
    ADD COLUMN vk_url TEXT NULL,
    ADD COLUMN telegram_url TEXT NULL;

ALTER TABLE event_participants
    ADD CONSTRAINT chk_event_participant_vk_url
        CHECK (vk_url IS NULL OR btrim(vk_url) <> ''),
    ADD CONSTRAINT chk_event_participant_telegram_url
        CHECK (telegram_url IS NULL OR btrim(telegram_url) <> '');
