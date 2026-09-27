CREATE TABLE IF NOT EXISTS backends (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL UNIQUE,
    description     TEXT,
    -- store a hash of the backend's API token, never the plaintext
    token_hash      TEXT     NOT NULL UNIQUE,
    -- no DB default: the app fills it from DEFAULT_WORKER_COUNT
    worker_count    INTEGER  NOT NULL CHECK (worker_count > 0),
    default_locale  TEXT     NOT NULL CHECK (length(default_locale) > 0),
    fcm_credentials TEXT     NOT NULL CHECK (length(fcm_credentials) > 0),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
