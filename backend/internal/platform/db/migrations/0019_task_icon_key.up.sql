-- Square task icon is stored separately from the wide cover image.
ALTER TABLE event_tasks
    ADD COLUMN icon_key TEXT NULL;
