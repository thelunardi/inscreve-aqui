-- Módulo events (ADR-0011): eventos e categorias.

-- +goose Up
CREATE TABLE events (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            text NOT NULL,
    starts_at       timestamptz NOT NULL,
    -- Fuso usado só na apresentação; datas são gravadas em UTC.
    timezone        text NOT NULL DEFAULT 'America/Sao_Paulo',
    -- Contador de números de peito (ADR-0003): só aumenta, nunca reutiliza.
    last_bib_number integer NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_events_name CHECK (btrim(name) <> ''),
    CONSTRAINT ck_events_last_bib_number CHECK (last_bib_number >= 0)
);

CREATE TABLE categories (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id    bigint NOT NULL REFERENCES events (id),
    name        text NOT NULL,
    price_cents bigint NOT NULL,
    capacity    integer NOT NULL,
    -- Reservadas mais confirmadas (ADR-0004).
    slots_taken integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_categories_name CHECK (btrim(name) <> ''),
    CONSTRAINT ck_categories_price CHECK (price_cents >= 0),
    CONSTRAINT ck_categories_capacity CHECK (capacity >= 0),
    -- Última barreira contra overbooking, por qualquer caminho (ADR-0004).
    CONSTRAINT ck_categories_slots CHECK (slots_taken >= 0 AND slots_taken <= capacity),
    CONSTRAINT uq_categories_event_name UNIQUE (event_id, name),
    -- Alvo da chave estrangeira composta de registrations.
    CONSTRAINT uq_categories_id_event UNIQUE (id, event_id)
);

-- +goose Down
DROP TABLE categories;
DROP TABLE events;
