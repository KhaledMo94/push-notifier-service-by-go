# Push Notifier

A Go service that sends push notifications through Firebase Cloud Messaging (FCM) on behalf of registered backends. Each backend has its own FCM credentials, worker count and default locale, and the service keeps track of device tokens that FCM reports as invalid ("stale") so backends can clean them up.

> **Status:** early development. Configuration, logging, the data layer (schema, models, repositories) and the crypto helpers are in place. The server currently starts up, connects to the database and validates the encryption key, then exits. The HTTP API, FCM sender, worker pool and backend admin command are not implemented yet.

## Requirements

- Go 1.27+
- A C compiler (`gcc`) with `CGO_ENABLED=1` — the SQLite driver (`mattn/go-sqlite3`) uses cgo
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI to apply migrations (built with the `sqlite3` tag)

## Getting started

```bash
git clone https://github.com/KhaledMo94/push-notifier-service-by-go.git push-notifier
cd push-notifier

cp .env.example .env
# set ENCRYPTION_KEY in .env:
openssl rand -base64 32

go mod download
migrate -path migration -database "sqlite3://notifier.db" up
go run ./cmd/server
```

Migrations are applied manually for now; the `-database` path must match `DB_PATH`.

## Configuration

Settings are read from environment variables. A `.env` file in the working directory is loaded on startup; variables already set in the environment take priority over it.

| Variable | Default | Description |
|---|---|---|
| `DB_PATH` | `notifier.db` | Path to the SQLite database file |
| `ENCRYPTION_KEY` | — (required) | Base64-encoded 32-byte key used to encrypt FCM credentials. Generate with `openssl rand -base64 32` |
| `DEFAULT_LOCALE` | `en` | Locale assigned to a backend when none is provided |
| `DEFAULT_WORKER_COUNT` | `1` | Worker count assigned to a backend when none is provided; must be > 0 |

The server refuses to start if `ENCRYPTION_KEY` is missing or is not a valid 32-byte key.

`.env` is git-ignored; keep `.env.example` up to date when adding variables.

## Project structure

```
cmd/server/            entry point: config, logging, database, cipher setup
internal/config/       environment configuration loading
internal/logger/       slog JSON logger (stdout + rotating file)
internal/crypto/       API token hashing, AES-GCM encryption
internal/storage/      SQLite connection setup (GORM)
internal/db/           DB wrapper embedding *gorm.DB
internal/models/       GORM models: Backend, StaleToken
internal/repository/   data access: backends and stale tokens
migration/             SQL migrations (golang-migrate format)
```

Empty placeholders for upcoming work: `internal/fcm`, `internal/pool`, `internal/aggregator`, `internal/transport`, `api/`, `deploy/`.

## Logging

`logger.Setup()` sets the default `slog` logger to write JSON at debug level to both stdout and `logs/app.log`. The file is rotated at 10 MB, rotated files are kept for 30 days and compressed.

`logs/app.log` is relative to the working directory the server is started from. The `logs/` directory is git-ignored.

## Database

SQLite, accessed through GORM. The schema is owned by the SQL files in `migration/` — GORM's `AutoMigrate` is not used.

### `backends`

A service allowed to send notifications.

| Column | Notes |
|---|---|
| `name` | unique |
| `description` | optional |
| `token_hash` | unique; SHA-256 of the backend's API token, never the plaintext |
| `worker_count` | required, > 0 |
| `default_locale` | required |
| `fcm_credentials` | required; the Firebase service-account JSON, meant to be stored encrypted |

### `stale_tokens`

Device tokens FCM reported as invalid, per backend.

- `(backend_id, token)` is unique across all rows, including soft-deleted ones.
- Soft-deleted via `deleted_at`; GORM excludes deleted rows from queries automatically.
- Deleting a backend deletes its stale tokens (`ON DELETE CASCADE`).

### Connection settings

`storage.OpenSQLite` enables these on every pooled connection: foreign keys, WAL journal mode, a 5-second busy timeout and `synchronous=NORMAL`.

Do not run migration files through the GORM connection: with prepared statements enabled, only the first statement of each file would run.

## Repositories

Each repository exposes small role interfaces, so consumers depend only on what they use.

**Backends** — `repository.NewBackendRepository(db)`

| Interface | Methods |
|---|---|
| `BackendLoader` | `LoadAll` |
| `BackendFinder` | `GetByID`, `GetByName`, `GetByTokenHash` |
| `BackendWriter` | `Create`, `Update`, `Delete` |

**Stale tokens** — `repository.NewStaleTokenRepository(db)`

| Interface | Methods |
|---|---|
| `StaleTokenRecorder` | `Record` — insert new tokens, revive soft-deleted ones, skip active ones |
| `StaleTokenReader` | `ListByBackend` (keyset pagination), `FindStale` |
| `StaleTokenRemover` | `MarkDeleted` (soft delete), `Purge` (hard-delete old soft-deleted rows) |

Repositories return `repository.ErrNotFound`, `ErrDuplicate` and `ErrReferenceNotFound` instead of GORM errors.

## Crypto

### API tokens

Each backend authenticates with an API token. The database stores only its SHA-256 hash; the plain token is shown once when created.

- `crypto.GenerateToken()` returns a new random token (`pn_` + 32 random bytes, base64url) and its hash.
- `crypto.HashToken(plain)` hashes an incoming token so it can be looked up by `token_hash`.

A fast unsalted hash is used on purpose: tokens are 256-bit random values, so they can't be guessed, and a deterministic hash allows lookup through the unique index.

### FCM credentials

`crypto.Cipher` is the encryption interface; `crypto.AESGCM` implements it with AES-256-GCM using `ENCRYPTION_KEY`.

- `Encrypt(plaintext)` returns `aesgcmv1:` + base64(nonce ‖ ciphertext). A new random nonce is used every time.
- `Decrypt(encoded)` returns the original bytes, or `crypto.ErrInvalidCiphertext` for a wrong key, altered data or an unknown format.

The `aesgcmv1:` prefix identifies the format so keys can be rotated later.

## Security notes

- **Back up `ENCRYPTION_KEY` outside the server.** Losing it makes every stored FCM credential unrecoverable.
- Never log plain API tokens, decrypted credentials or the encryption key.
- `token_hash` and `fcm_credentials` are excluded from JSON output.

## Testing

There are no tests at the moment.
