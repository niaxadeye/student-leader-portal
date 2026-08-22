-- Фактическая длительность выступления конкурсанта (секунды), фиксируется кнопкой «Аплодисменты».
ALTER TABLE performances
    ADD COLUMN speech_duration_seconds DOUBLE PRECISION NULL;
