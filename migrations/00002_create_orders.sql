-- Módulo orders (ADR-0002, ADR-0011): pedidos e seu histórico de status.

-- +goose Up
CREATE TABLE orders (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Código público exibido no comprovante.
    code            text NOT NULL,
    event_id        bigint NOT NULL REFERENCES events (id),
    buyer_name      text NOT NULL,
    buyer_email     text NOT NULL,
    total_cents     bigint NOT NULL,
    status          text NOT NULL DEFAULT 'awaiting_payment',
    -- Gerada pelo frontend a cada tentativa de checkout (ADR-0003).
    idempotency_key uuid NOT NULL,
    expires_at      timestamptz NOT NULL,
    paid_at         timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_orders_status CHECK (status IN ('awaiting_payment', 'paid', 'expired')),
    CONSTRAINT ck_orders_total CHECK (total_cents >= 0),
    CONSTRAINT ck_orders_paid_at CHECK ((status = 'paid') = (paid_at IS NOT NULL)),
    CONSTRAINT uq_orders_code UNIQUE (code),
    CONSTRAINT uq_orders_idempotency_key UNIQUE (idempotency_key),
    -- Alvo da chave estrangeira composta de registrations.
    CONSTRAINT uq_orders_id_event UNIQUE (id, event_id)
);

CREATE INDEX ix_orders_awaiting_expires ON orders (expires_at) WHERE status = 'awaiting_payment';

CREATE TABLE order_status_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    bigint NOT NULL REFERENCES orders (id),
    -- NULL na criação do pedido.
    from_status text,
    to_status   text NOT NULL,
    -- Quem causou a transição, ex.: "buyer", "webhook", "reconciliation", "expiration_job".
    actor       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_order_status_history_order ON order_status_history (order_id);

-- +goose Down
DROP TABLE order_status_history;
DROP TABLE orders;
