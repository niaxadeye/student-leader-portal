-- Immutable журнал баллов участников мероприятий.
-- Текущий баланс всегда вычисляется как SUM(amount); прямого поля balance нет.

CREATE TABLE points_ledger (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL,
    event_participant_id UUID NOT NULL,
    amount               BIGINT NOT NULL,
    type                 VARCHAR(32) NOT NULL,
    source_type          VARCHAR(64) NULL,
    source_id            UUID NULL,
    description          TEXT NOT NULL,
    created_by_user_id   UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    idempotency_key      VARCHAR(200) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_points_ledger_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_points_ledger_amount CHECK (amount <> 0),
    CONSTRAINT chk_points_ledger_type CHECK (type IN (
        'LECTURE_ATTENDANCE',
        'TASK_REWARD',
        'MERCH_PURCHASE',
        'ADMIN_ADJUSTMENT',
        'REFUND'
    )),
    CONSTRAINT chk_points_ledger_description CHECK (btrim(description) <> ''),
    CONSTRAINT chk_points_ledger_idempotency_key CHECK (btrim(idempotency_key) <> ''),
    CONSTRAINT chk_points_ledger_source_pair CHECK (
        (source_type IS NULL AND source_id IS NULL)
        OR (source_type IS NOT NULL AND source_id IS NOT NULL)
    ),
    CONSTRAINT chk_points_ledger_source_type CHECK (
        source_type IS NULL OR btrim(source_type) <> ''
    ),
    CONSTRAINT chk_points_ledger_source_by_type CHECK (
        (type = 'ADMIN_ADJUSTMENT' AND source_type IS NULL AND source_id IS NULL)
        OR (type <> 'ADMIN_ADJUSTMENT' AND source_type IS NOT NULL AND source_id IS NOT NULL)
    ),
    CONSTRAINT chk_points_ledger_admin_actor CHECK (
        type <> 'ADMIN_ADJUSTMENT' OR created_by_user_id IS NOT NULL
    ),
    CONSTRAINT chk_points_ledger_reward_sign CHECK (
        type NOT IN ('LECTURE_ATTENDANCE', 'TASK_REWARD', 'REFUND') OR amount > 0
    ),
    CONSTRAINT chk_points_ledger_purchase_sign CHECK (
        type <> 'MERCH_PURCHASE' OR amount < 0
    ),
    CONSTRAINT uq_points_ledger_idempotency UNIQUE (contest_id, idempotency_key)
);

CREATE INDEX idx_points_ledger_participant_created
    ON points_ledger(event_participant_id, created_at DESC, id DESC);
CREATE INDEX idx_points_ledger_contest_created
    ON points_ledger(contest_id, created_at DESC);
CREATE INDEX idx_points_ledger_source
    ON points_ledger(source_type, source_id)
    WHERE source_type IS NOT NULL AND source_id IS NOT NULL;
CREATE UNIQUE INDEX uq_points_ledger_source_operation
    ON points_ledger(
        contest_id, event_participant_id, type, source_type, source_id
    )
    WHERE type <> 'ADMIN_ADJUSTMENT';

-- Ledger физически append-only даже при ошибочном прямом SQL из приложения.
CREATE FUNCTION reject_points_ledger_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'points_ledger is append-only; create a compensating entry instead';
END;
$$;

CREATE TRIGGER trg_points_ledger_append_only
BEFORE UPDATE OR DELETE ON points_ledger
FOR EACH ROW EXECUTE FUNCTION reject_points_ledger_mutation();
