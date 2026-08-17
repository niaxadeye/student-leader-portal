-- Merch catalog, participant saving targets, atomic stock/points reservations and orders.

CREATE TABLE merch_products (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id            UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    title                 TEXT NOT NULL,
    slug                  VARCHAR(180) NOT NULL,
    description           TEXT NOT NULL,
    price_points          BIGINT NOT NULL,
    discount_price_points BIGINT NULL,
    stock_quantity        INT NOT NULL DEFAULT 0,
    reserved_quantity     INT NOT NULL DEFAULT 0,
    status                VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_merch_products_contest_id UNIQUE (contest_id, id),
    CONSTRAINT uq_merch_products_slug UNIQUE (contest_id, slug),
    CONSTRAINT chk_merch_product_title CHECK (btrim(title) <> ''),
    CONSTRAINT chk_merch_product_description CHECK (btrim(description) <> ''),
    CONSTRAINT chk_merch_product_price CHECK (price_points > 0),
    CONSTRAINT chk_merch_product_discount CHECK (
        discount_price_points IS NULL
        OR (discount_price_points > 0 AND discount_price_points < price_points)
    ),
    CONSTRAINT chk_merch_product_stock CHECK (
        stock_quantity >= 0 AND reserved_quantity >= 0
        AND reserved_quantity <= stock_quantity
    ),
    CONSTRAINT chk_merch_product_status CHECK (
        status IN ('DRAFT','ACTIVE','HIDDEN','SOLD_OUT')
    )
);

CREATE INDEX idx_merch_products_contest_status
    ON merch_products(contest_id, status, created_at DESC);

CREATE TABLE merch_product_images (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id    UUID NOT NULL,
    product_id    UUID NOT NULL,
    object_key    TEXT NOT NULL,
    original_name TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_merch_product_image_product
        FOREIGN KEY (contest_id, product_id)
        REFERENCES merch_products(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_merch_product_image_object UNIQUE (object_key),
    CONSTRAINT chk_merch_product_image_size CHECK (size_bytes > 0)
);

CREATE INDEX idx_merch_product_images_product
    ON merch_product_images(product_id, sort_order, created_at);

CREATE TABLE merch_saving_targets (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL,
    event_participant_id UUID NOT NULL,
    product_id           UUID NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_merch_target_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_merch_target_product
        FOREIGN KEY (contest_id, product_id)
        REFERENCES merch_products(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_merch_target_participant UNIQUE (event_participant_id)
);

CREATE INDEX idx_merch_saving_targets_product ON merch_saving_targets(product_id);

CREATE TABLE merch_orders (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL,
    event_participant_id UUID NOT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'RESERVED',
    points_total         BIGINT NOT NULL,
    rejection_reason     TEXT NULL,
    idempotency_key      VARCHAR(200) NOT NULL,
    request_fingerprint  CHAR(64) NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    issued_at            TIMESTAMPTZ NULL,
    rejected_at          TIMESTAMPTZ NULL,
    cancelled_at         TIMESTAMPTZ NULL,
    issued_by_user_id    UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    rejected_by_user_id  UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_merch_orders_contest_id UNIQUE (contest_id, id),
    CONSTRAINT uq_merch_order_idempotency UNIQUE (contest_id, idempotency_key),
    CONSTRAINT fk_merch_order_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_merch_order_points CHECK (points_total > 0),
    CONSTRAINT chk_merch_order_idempotency CHECK (btrim(idempotency_key) <> ''),
    CONSTRAINT chk_merch_order_status CHECK (
        status IN ('RESERVED','ISSUED','REJECTED','CANCELLED')
    ),
    CONSTRAINT chk_merch_order_terminal_state CHECK (
        (status='RESERVED' AND issued_at IS NULL AND rejected_at IS NULL
          AND cancelled_at IS NULL AND issued_by_user_id IS NULL
          AND rejected_by_user_id IS NULL AND rejection_reason IS NULL)
        OR
        (status='ISSUED' AND issued_at IS NOT NULL AND issued_by_user_id IS NOT NULL
          AND rejected_at IS NULL AND cancelled_at IS NULL
          AND rejected_by_user_id IS NULL AND rejection_reason IS NULL)
        OR
        (status='REJECTED' AND rejected_at IS NOT NULL AND rejected_by_user_id IS NOT NULL
          AND rejection_reason IS NOT NULL AND btrim(rejection_reason) <> ''
          AND issued_at IS NULL AND cancelled_at IS NULL AND issued_by_user_id IS NULL)
        OR
        (status='CANCELLED' AND cancelled_at IS NOT NULL
          AND issued_at IS NULL AND rejected_at IS NULL
          AND issued_by_user_id IS NULL AND rejected_by_user_id IS NULL
          AND rejection_reason IS NULL)
    )
);

CREATE INDEX idx_merch_orders_participant
    ON merch_orders(event_participant_id, created_at DESC);
CREATE INDEX idx_merch_orders_contest_status
    ON merch_orders(contest_id, status, created_at DESC);

CREATE TABLE merch_order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id   UUID NOT NULL,
    order_id     UUID NOT NULL,
    product_id   UUID NOT NULL,
    product_title TEXT NOT NULL,
    quantity     INT NOT NULL,
    price_points BIGINT NOT NULL,
    total_points BIGINT NOT NULL,
    CONSTRAINT fk_merch_order_item_order
        FOREIGN KEY (contest_id, order_id)
        REFERENCES merch_orders(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_merch_order_item_product
        FOREIGN KEY (contest_id, product_id)
        REFERENCES merch_products(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_merch_order_item_product UNIQUE (order_id, product_id),
    CONSTRAINT chk_merch_order_item_values CHECK (
        quantity > 0 AND price_points > 0
        AND total_points = price_points * quantity
        AND btrim(product_title) <> ''
    )
);

CREATE INDEX idx_merch_order_items_order ON merch_order_items(order_id);

CREATE TABLE points_holds (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_id           UUID NOT NULL,
    event_participant_id UUID NOT NULL,
    merch_order_id       UUID NOT NULL,
    amount               BIGINT NOT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    captured_at          TIMESTAMPTZ NULL,
    released_at          TIMESTAMPTZ NULL,
    CONSTRAINT fk_points_hold_participant
        FOREIGN KEY (contest_id, event_participant_id)
        REFERENCES event_participants(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_points_hold_order
        FOREIGN KEY (contest_id, merch_order_id)
        REFERENCES merch_orders(contest_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_points_hold_order UNIQUE (merch_order_id),
    CONSTRAINT chk_points_hold_amount CHECK (amount > 0),
    CONSTRAINT chk_points_hold_status CHECK (status IN ('ACTIVE','CAPTURED','RELEASED')),
    CONSTRAINT chk_points_hold_terminal_state CHECK (
        (status='ACTIVE' AND captured_at IS NULL AND released_at IS NULL)
        OR (status='CAPTURED' AND captured_at IS NOT NULL AND released_at IS NULL)
        OR (status='RELEASED' AND released_at IS NOT NULL AND captured_at IS NULL)
    )
);

CREATE INDEX idx_points_holds_participant_active
    ON points_holds(event_participant_id, status)
    WHERE status='ACTIVE';
