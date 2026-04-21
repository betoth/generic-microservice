CREATE TYPE entry_status AS ENUM ('created', 'processing', 'published');

CREATE TABLE entries (
    id         UUID         PRIMARY KEY,
    date       TIMESTAMPTZ  NOT NULL,
    subject    TEXT         NOT NULL,
    content    TEXT         NOT NULL,
    status     entry_status NOT NULL DEFAULT 'created',
    created_at TIMESTAMPTZ  NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL
);

CREATE TABLE outbox (
    id           UUID        PRIMARY KEY,
    business_key UUID        NOT NULL,
    event_type   TEXT        NOT NULL,
    topic        TEXT        NOT NULL,
    event_data   TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_outbox_business_key ON outbox (business_key);
