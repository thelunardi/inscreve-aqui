-- Módulo notifications (ADR-0005, ADR-0011): e-mails enfileirados pelo padrão outbox.

-- +goose Up
CREATE TABLE outbox_messages (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Tipo da mensagem, ex.: "order_confirmed".
    kind            text NOT NULL,
    recipient       text NOT NULL,
    payload         jsonb NOT NULL,
    status          text NOT NULL DEFAULT 'pending',
    attempts        integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    sent_at         timestamptz,

    CONSTRAINT ck_outbox_messages_status CHECK (status IN ('pending', 'sent', 'failed')),
    CONSTRAINT ck_outbox_messages_attempts CHECK (attempts >= 0),
    CONSTRAINT ck_outbox_messages_sent_at CHECK ((status = 'sent') = (sent_at IS NOT NULL))
);

-- Fila do worker: mensagens pendentes, por ordem de próxima tentativa.
CREATE INDEX ix_outbox_messages_pending
    ON outbox_messages (next_attempt_at)
    WHERE status = 'pending';

-- +goose Down
DROP TABLE outbox_messages;
