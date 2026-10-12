-- Módulo registrations (ADR-0002, ADR-0003, ADR-0011): inscrições e seu histórico de status.

-- +goose Up
CREATE TABLE registrations (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    bigint NOT NULL,
    event_id    bigint NOT NULL,
    category_id bigint NOT NULL,
    name        text NOT NULL,
    -- Somente dígitos; a validação dos dígitos verificadores fica no domínio.
    cpf         char(11) NOT NULL,
    birth_date  date NOT NULL,
    shirt_size  text,
    -- Atribuído uma única vez, na confirmação (ADR-0003).
    bib_number  integer,
    status      text NOT NULL DEFAULT 'reserved',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    -- Chaves compostas: pedido, categoria e inscrição são sempre do mesmo evento.
    CONSTRAINT fk_registrations_order FOREIGN KEY (order_id, event_id) REFERENCES orders (id, event_id),
    CONSTRAINT fk_registrations_category FOREIGN KEY (category_id, event_id) REFERENCES categories (id, event_id),
    CONSTRAINT ck_registrations_name CHECK (btrim(name) <> ''),
    CONSTRAINT ck_registrations_cpf CHECK (cpf ~ '^[0-9]{11}$'),
    CONSTRAINT ck_registrations_bib_number CHECK (bib_number > 0),
    CONSTRAINT ck_registrations_status CHECK (status IN ('reserved', 'confirmed', 'canceled')),
    CONSTRAINT ck_registrations_confirmed_has_bib CHECK (status <> 'confirmed' OR bib_number IS NOT NULL),
    CONSTRAINT ck_registrations_reserved_no_bib CHECK (status <> 'reserved' OR bib_number IS NULL)
);

-- Um CPF ativo por evento; inscrições canceladas não bloqueiam uma nova (ADR-0003).
CREATE UNIQUE INDEX uq_registrations_event_cpf
    ON registrations (event_id, cpf)
    WHERE status <> 'canceled';

-- Número de peito único por evento. Inclui canceladas, então um número nunca é reutilizado.
CREATE UNIQUE INDEX uq_registrations_event_bib
    ON registrations (event_id, bib_number)
    WHERE bib_number IS NOT NULL;

CREATE INDEX ix_registrations_order ON registrations (order_id);
-- Listas operacionais: inscrições confirmadas de um evento.
CREATE INDEX ix_registrations_event_status ON registrations (event_id, status);

CREATE TABLE registration_status_history (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    registration_id bigint NOT NULL REFERENCES registrations (id),
    from_status     text,
    to_status       text NOT NULL,
    actor           text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_registration_status_history_registration ON registration_status_history (registration_id);

-- +goose Down
DROP TABLE registration_status_history;
DROP TABLE registrations;
