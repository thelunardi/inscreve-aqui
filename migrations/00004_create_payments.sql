-- Módulo payments (ADR-0005, ADR-0011): pagamentos, histórico de status e webhooks recebidos.

-- +goose Up
CREATE TABLE payments (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Um pedido pode ter vários pagamentos, ex.: um Pix expirado e um novo.
    order_id     bigint NOT NULL REFERENCES orders (id),
    gateway      text NOT NULL,
    external_id  text NOT NULL,
    amount_cents bigint NOT NULL,
    status       text NOT NULL DEFAULT 'pending',
    -- Expiração da cobrança no gateway; deve ser anterior à do pedido (ADR-0004).
    expires_at   timestamptz NOT NULL,
    approved_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_payments_amount CHECK (amount_cents > 0),
    CONSTRAINT ck_payments_status CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT ck_payments_approved_at CHECK ((status = 'approved') = (approved_at IS NOT NULL)),
    CONSTRAINT uq_payments_gateway_external_id UNIQUE (gateway, external_id)
);

CREATE INDEX ix_payments_order ON payments (order_id);

CREATE TABLE payment_status_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payment_id  bigint NOT NULL REFERENCES payments (id),
    from_status text,
    to_status   text NOT NULL,
    actor       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_payment_status_history_payment ON payment_status_history (payment_id);

-- Webhooks são gravados antes de processar (ADR-0005).
CREATE TABLE received_webhooks (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    gateway           text NOT NULL,
    external_event_id text NOT NULL,
    payload           jsonb NOT NULL,
    status            text NOT NULL DEFAULT 'received',
    attempts          integer NOT NULL DEFAULT 0,
    next_attempt_at   timestamptz NOT NULL DEFAULT now(),
    last_error        text,
    received_at       timestamptz NOT NULL DEFAULT now(),
    processed_at      timestamptz,

    CONSTRAINT ck_received_webhooks_status CHECK (status IN ('received', 'processed', 'failed')),
    CONSTRAINT ck_received_webhooks_attempts CHECK (attempts >= 0),
    CONSTRAINT uq_received_webhooks_gateway_event UNIQUE (gateway, external_event_id)
);

-- Fila do worker: webhooks ainda não processados, por ordem de próxima tentativa.
CREATE INDEX ix_received_webhooks_pending
    ON received_webhooks (next_attempt_at)
    WHERE status = 'received';

-- +goose Down
DROP TABLE received_webhooks;
DROP TABLE payment_status_history;
DROP TABLE payments;
