-- requires PRAGMA foreign_keys = ON on every connection for the FK to be enforced
CREATE TABLE IF NOT EXISTS stale_tokens (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    backend_id INTEGER  NOT NULL REFERENCES backends (id) ON DELETE CASCADE,
    token      TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL DEFAULT NULL
);

-- one row per (backend, token), including soft-deleted rows; re-recording revives the row
CREATE UNIQUE INDEX IF NOT EXISTS uq_stale_tokens_backend_token
    ON stale_tokens (backend_id, token);

-- keeps (backend_id, id) ordered for keyset pagination; the unique index is ordered by token instead
CREATE INDEX IF NOT EXISTS idx_stale_tokens_backend_id
    ON stale_tokens (backend_id);

-- partial so it only holds soft-deleted rows; serves the purge of old deleted tokens
CREATE INDEX IF NOT EXISTS idx_stale_tokens_deleted_at
    ON stale_tokens (deleted_at)
    WHERE deleted_at IS NOT NULL;
