-- Numeric social IDs for VK / Telegram login. URLs remain for display and first-time matching.
ALTER TABLE event_participants
    ADD COLUMN vk_user_id BIGINT NULL,
    ADD COLUMN telegram_user_id BIGINT NULL;

CREATE UNIQUE INDEX uq_event_participants_vk_user
    ON event_participants (contest_id, vk_user_id)
    WHERE vk_user_id IS NOT NULL;

CREATE UNIQUE INDEX uq_event_participants_telegram_user
    ON event_participants (contest_id, telegram_user_id)
    WHERE telegram_user_id IS NOT NULL;
