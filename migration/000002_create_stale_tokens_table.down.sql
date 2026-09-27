DROP INDEX IF EXISTS idx_stale_tokens_deleted_at;
DROP INDEX IF EXISTS idx_stale_tokens_backend_id;
DROP INDEX IF EXISTS uq_stale_tokens_backend_token;
DROP TABLE IF EXISTS stale_tokens;
