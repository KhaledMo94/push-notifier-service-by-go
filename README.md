# Push Notifier

A Go service that sends push notifications through Firebase Cloud Messaging (FCM) on behalf of registered backends. Each backend has its own FCM credentials, worker count and default locale, and the service keeps track of device tokens that FCM reports as invalid ("stale") so backends can clean them up.

> **Status:** early development. The data layer (schema, models, repositories) and configuration are in place and tested. The HTTP server, FCM sender and worker pool are not implemented yet.

## Requirements

- Go 1.27+
- A C compiler (`gcc`) with `CGO_ENABLED=1` — the SQLite driver (`mattn/go-sqlite3`) uses cgo
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI to apply migrations (built with the `sqlite3` tag)

## Getting started

```bash
git clone <repo-url> push-notifier
cd push-notifier

cp .env.example .env        # then adjust values
go mod download

migrate -path migration -database "sqlite3://notifier.db" up
go test ./...
```

## Configuration

Settings are read from environment variables. A `.env` file in the working directory is loaded on startup; variables already set in the environment take priority over it.

| Variable | Default | Description |
|---|---|---|
| `DEFAULT_LOCALE` | `en` | Locale assigned to a backend when none is provided |
| `DEFAULT_WORKER_COUNT` | `1` | Worker count assigned to a backend when none is provided; must be > 0 |

`.env` is git-ignored; keep `.env.example` up to date when adding variables.

## Project structure

```
cmd/server/            entry point (not implemented yet)
internal/config/       environment configuration loading
internal/storage/      SQLite connection setup (GORM)
internal/db/           DB wrapper embedding *gorm.DB
internal/models/       GORM models: Backend, StaleToken
internal/repository/   data access: backends and stale tokens
migration/             SQL migrations (golang-migrate format)
```

## Database

SQLite, accessed through GORM. The schema is owned by the SQL files in `migration/` — GORM's `AutoMigrate` is not used.

### `backends`

A service allowed to send notifications.

| Column | Notes |
|---|---|
| `name` | unique |
| `description` | optional |
| `token_hash` | unique; hash of the backend's API token, never the plaintext |
| `worker_count` | required, > 0 |
| `default_locale` | required |
| `fcm_credentials` | required; the Firebase service-account JSON |

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

## Testing

```bash
go test ./...
```

Repository tests run against a temporary SQLite database with the real migrations applied.

## Security notes

- `fcm_credentials` is currently stored as plaintext. Anyone who can read the database file can send pushes as that Firebase project — encrypt it before storing (planned in `internal/crypto`).
- `token_hash` and `fcm_credentials` are excluded from JSON output.
