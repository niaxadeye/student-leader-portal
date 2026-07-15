-- Транзакционный outbox для внешних уведомлений (SITE.md §15, §21.16, Этап 5).
-- Событие пишется в одной транзакции с бизнес-операцией (отправка формы);
-- диспетчер отдельно опрашивает PENDING и доставляет в Telegram с ретраями.
CREATE TABLE outbox_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type     VARCHAR(64) NOT NULL,        -- submission.submitted | submission.resubmitted | ...
    aggregate_type VARCHAR(32) NOT NULL,        -- submission | file | ...
    aggregate_id   UUID NOT NULL,
    payload        JSONB NOT NULL DEFAULT '{}',
    status         VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING|SENT|DEAD
    attempts       INT NOT NULL DEFAULT 0,
    available_at   TIMESTAMPTZ NOT NULL DEFAULT now(),      -- когда событие снова можно брать (backoff)
    locked_at      TIMESTAMPTZ NULL,
    locked_by      TEXT NULL,
    last_error     TEXT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at   TIMESTAMPTZ NULL
);

-- Частичный индекс под выборку диспетчера: только необработанные, по времени доступности.
CREATE INDEX idx_outbox_pending ON outbox_events (available_at)
    WHERE status = 'PENDING';
