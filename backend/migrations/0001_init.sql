-- Admin users (login do painel)
CREATE TABLE IF NOT EXISTS admin_users (
    id              BIGSERIAL PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pixels (cada pixelName pode ter múltiplos pixel_id/access_token)
CREATE TABLE IF NOT EXISTS pixels (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,            -- pixelName usado pelo client (slug)
    label           TEXT NOT NULL DEFAULT '',
    pixel_ids       TEXT[] NOT NULL DEFAULT '{}',    -- ids do Meta
    access_tokens   TEXT[] NOT NULL DEFAULT '{}',    -- paralelo a pixel_ids
    test_codes      TEXT[] NOT NULL DEFAULT '{}',    -- paralelo a pixel_ids
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tracked users (visitantes/leads identificados pelo external_id)
CREATE TABLE IF NOT EXISTS tracked_users (
    id                 BIGSERIAL PRIMARY KEY,
    external_id        TEXT NOT NULL UNIQUE,
    pixel_name         TEXT NOT NULL DEFAULT '',
    fbp                TEXT NOT NULL DEFAULT '',
    fbc                TEXT NOT NULL DEFAULT '',
    client_ip          TEXT NOT NULL DEFAULT '',
    client_user_agent  TEXT NOT NULL DEFAULT '',

    country            TEXT NOT NULL DEFAULT '',
    state              TEXT NOT NULL DEFAULT '',
    city               TEXT NOT NULL DEFAULT '',
    postal_code        TEXT NOT NULL DEFAULT '',

    first_name         TEXT NOT NULL DEFAULT '',
    last_name          TEXT NOT NULL DEFAULT '',
    email              TEXT NOT NULL DEFAULT '',
    phone              TEXT NOT NULL DEFAULT '',

    url                TEXT NOT NULL DEFAULT '',

    first_utm_source   TEXT NOT NULL DEFAULT '',
    first_utm_medium   TEXT NOT NULL DEFAULT '',
    first_utm_campaign TEXT NOT NULL DEFAULT '',
    first_utm_content  TEXT NOT NULL DEFAULT '',
    first_utm_term     TEXT NOT NULL DEFAULT '',

    last_utm_source    TEXT NOT NULL DEFAULT '',
    last_utm_medium    TEXT NOT NULL DEFAULT '',
    last_utm_campaign  TEXT NOT NULL DEFAULT '',
    last_utm_content   TEXT NOT NULL DEFAULT '',
    last_utm_term      TEXT NOT NULL DEFAULT '',

    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tracked_users_phone ON tracked_users(phone) WHERE phone <> '';
CREATE INDEX IF NOT EXISTS idx_tracked_users_email ON tracked_users(email) WHERE email <> '';
CREATE INDEX IF NOT EXISTS idx_tracked_users_pixel ON tracked_users(pixel_name);

-- Eventos enviados (auditoria + dedup)
CREATE TABLE IF NOT EXISTS sent_events (
    id            BIGSERIAL PRIMARY KEY,
    event_id      TEXT NOT NULL,
    event_name    TEXT NOT NULL,
    pixel_name    TEXT NOT NULL,
    external_id   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL,           -- queued | sent | failed
    response_code INT NOT NULL DEFAULT 0,
    response_body TEXT NOT NULL DEFAULT '',
    error         TEXT NOT NULL DEFAULT '',
    payload       JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sent_events_event_id ON sent_events(event_id);
CREATE INDEX IF NOT EXISTS idx_sent_events_pixel ON sent_events(pixel_name, created_at DESC);
