# Push Notifier

A Go service that sends push notifications through Firebase Cloud Messaging (FCM) on behalf of registered backends. Each backend has its own FCM credentials, worker count and default locale, and the service keeps track of device tokens that FCM reports as invalid ("stale") so backends can clean them up.

> **Status:** early development. Configuration, logging, the data layer (schema, models, repositories) and the crypto helpers are in place. The server currently starts up, connects to the database, validates the encryption key and applies pending migrations, then exits. The `admin` command can create, list and delete backends and rotate their API tokens. The HTTP API, FCM sender and worker pool are not implemented yet.



## Requirements

- Go 1.27+
- A C compiler (`gcc`) with `CGO_ENABLED=1` — the SQLite driver (`mattn/go-sqlite3`) uses cgo
- Optional: the [golang-migrate](https://github.com/golang-migrate/migrate) CLI (built with the `sqlite3` tag) to inspect or roll back migrations by hand



## Getting started

```bash
git clone https://github.com/KhaledMo94/push-notifier-service-by-go.git push-notifier
cd push-notifier

cp .env.example .env
# set the key for ENCRYPTION_ALGORITHM in .env (AES_GCM_KEY or CHACHA20_POLY1305_KEY):
openssl rand -base64 32

go mod download
go run ./cmd/server
```

### Migrations

The SQL files in `internal/db/migration/` are embedded into the server binary. On every start, `db.Migrate()` applies any pending migrations and logs the schema version:

```json
{"level":"INFO","msg":"database migrated","version":2}
```

- When the schema is already up to date nothing runs, so restarting is safe.
- The server refuses to start if a migration fails or the schema is marked dirty.
- The `admin` command does **not** run migrations: on a new database, start the server once before using it.
- The version is tracked in the `schema_migrations` table, the same one the `migrate` CLI uses, so both can be used on the same database.
- Add a migration as a new numbered pair (`000003_<name>.up.sql` / `.down.sql`); never edit one that has already been applied.

### Installing the `migrate` CLI (optional)

Only needed to check the version, roll back or clear a dirty state by hand.

```bash
CGO_ENABLED=1 go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

- `go install` puts binaries in `$(go env GOPATH)/bin` (usually `~/go/bin`). Add it to your `PATH`, e.g. `echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.bashrc`.
- If `go env GOBIN` prints a custom folder, binaries go there instead; reset it with `go env -u GOBIN`.
- `go get github.com/golang-migrate/migrate/v4` only adds the library to `go.mod`; it does not install the CLI.
- If the shell suggests `apt install python3-migrate`, ignore it — that is an unrelated Python tool.



### Checking migrations

```bash
migrate -path internal/db/migration -database "sqlite3://notifier.db" version   # 2 = all applied
sqlite3 notifier.db ".tables"                                                    # backends, schema_migrations, stale_tokens
sqlite3 notifier.db "SELECT * FROM schema_migrations;"                           # 2|0 = version 2, not dirty
```

A version printed as `N (dirty)` means a migration failed partway; fix the database, then clear the flag with `migrate ... force <last-good-version>`. Roll back one step with `migrate ... down 1`.

The database runs in WAL mode, so recent writes may live in `notifier.db-wal` until SQLite checkpoints them. Keep `notifier.db`, `notifier.db-wal` and `notifier.db-shm` together, and only copy or delete the database while the server is stopped.

To browse the data in a GUI, use a **SQLite** connection to the absolute path of `notifier.db` (no host, user or password). DuckDB-based connections need the `sqlite_scanner` extension and may fail to download it.

## Configuration

Settings are read from environment variables. A `.env` file in the working directory is loaded on startup; variables already set in the environment take priority over it.


| Variable                | Default       | Description                                                                        |
| ----------------------- | ------------- | ---------------------------------------------------------------------------------- |
| `DB_PATH`               | `notifier.db` | Path to the SQLite database file                                                   |
| `ENCRYPTION_ALGORITHM`  | `aes-gcm`     | Cipher for FCM credentials: `aes-gcm` or `chacha20-poly1305`                       |
| `AES_GCM_KEY`           | —             | Base64-encoded 32-byte key; required when `ENCRYPTION_ALGORITHM=aes-gcm`           |
| `CHACHA20_POLY1305_KEY` | —             | Base64-encoded 32-byte key; required when `ENCRYPTION_ALGORITHM=chacha20-poly1305` |
| `DEFAULT_LOCALE`        | `en`          | Locale assigned to a backend when none is provided                                 |
| `DEFAULT_WORKER_COUNT`  | `1`           | Worker count assigned to a backend when none is provided; must be > 0              |


Generate keys with `openssl rand -base64 32`. The server refuses to start if the algorithm is unknown, or if the selected algorithm's key is missing or is not a valid 32-byte key.

`.env` is git-ignored; keep `.env.example` up to date when adding variables.

## Project structure

```
cmd/server/            entry point: config, logging, database, cipher setup, migrations
cmd/admin/             admin CLI entry point: picks a subcommand and runs it
cmd/admin/command/     admin subcommands (create, list, delete, rotate-token) and their registry
internal/config/       environment configuration loading
internal/logger/       slog JSON logger (stdout + rotating file)
internal/crypto/       API token hashing, AES-GCM and ChaCha20-Poly1305 encryption
internal/storage/      SQLite connection setup (GORM)
internal/db/           DB wrapper embedding *gorm.DB; Migrate() applies the embedded migrations
internal/db/migration/ SQL migrations (golang-migrate format), embedded into the binary
internal/models/       GORM models: Backend, StaleToken
internal/repository/   data access: backends and stale tokens
```

Empty placeholders for upcoming work: `internal/fcm`, `internal/pool`, `internal/aggregator`, `internal/transport`, `api/`, `deploy/`.

## Admin command

`cmd/admin` manages backends from the command line. It reads `.env` and opens `DB_PATH` like the server, so **run it from the project root**; from another folder it won't find `.env` and may create a new empty database there. It doesn't create tables, so run the server once on a new database first.

```bash
go run ./cmd/admin                    # list the available commands
go run ./cmd/admin <command> -h       # show one command's flags
```


| Command        | Example                                                                                       | What it does                                                                                                                                                                          |
| -------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `create`       | `create -name my-app -fcm-credentials sa.json [-description "..."] [-workers 4] [-locale ar]` | Encrypts the service-account JSON, stores the backend and prints its API token **once**. `-workers` and `-locale` default to `DEFAULT_WORKER_COUNT` and `DEFAULT_LOCALE`.             |
| `list`         | `list [-search app] [-sort]`                                                                  | Prints id, name, workers, locale, created time and description. `-search` filters names (case-insensitive); `-sort` shows the newest first. Never prints token hashes or credentials. |
| `delete`       | `delete -name my-app [-yes]` or `delete -id 3`                                                | Asks for confirmation (skip with `-yes`), then deletes the backend and its stale tokens.                                                                                              |
| `rotate-token` | `rotate-token -name my-app` or `rotate-token -id 3`                                           | Replaces the backend's API token and prints the new one once; the old token stops working.                                                                                            |


`delete` and `rotate-token` take `-id` or `-name`; if both are given, `-id` is used.

Notes:

- Flags must come before any other argument: in `create foo -name x`, parsing stops at `foo` and `-name` is ignored.
- Exit codes: `0` success or `-h`, `1` command error, `2` missing or unknown command.
- Ctrl+C cancels the command's context, so a running database operation stops instead of being killed midway.
- Store the printed API token right away; only its SHA-256 hash is saved, so it can't be shown again (use `rotate-token` if it's lost).
- Keep service-account files out of git (`.gitignore` covers `*-sa.json`).



### Adding a command

Each command registers itself, so `main.go` and `command.go` never change when commands are added:

1. Create `cmd/admin/command/<name>_command.go` with a type implementing `command.Command` (`Name`, `Usage`, `Run`).
2. Add `func init() { Register(&myCmd{}) }` in that file.
3. Parse flags with its own `flag.NewFlagSet`, and pass the command logic only the repository interfaces it uses (e.g. `BackendFinder` + `BackendWriter`).

`Register` panics on a duplicate name, so a clash is caught at startup.

### `go run` vs. a built binary


| Goal                         | Command                                                     | Notes                                                                                                   |
| ---------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Develop and try commands     | `go run ./cmd/admin list`                                   | Always compiles the current code. Reports a non-zero exit as `exit status N` and itself exits with `1`. |
| Check real exit codes        | `go build -o /tmp/admin ./cmd/admin && /tmp/admin; echo $?` | `/tmp` is cleared on reboot; rebuild when needed.                                                       |
| Keep a binary in the project | `go build -o bin/admin ./cmd/admin`                         | `/bin/` is git-ignored.                                                                                 |
| Use it as a normal command   | `go install ./cmd/admin`, then `admin list`                 | Installed to `~/go/bin`.                                                                                |


A built binary is a snapshot: it does not pick up code changes until you build or install it again.

## Logging

`logger.Setup()` sets the default `slog` logger to write JSON at debug level to both stdout and `logs/app.log`. The file is rotated at 10 MB, rotated files are kept for 30 days and compressed.

`logs/app.log` is relative to the working directory the server is started from. The `logs/` directory is git-ignored.

## Database

SQLite, accessed through GORM. The schema is owned by the SQL files in `internal/db/migration/`, applied by golang-migrate when the server starts — GORM's `AutoMigrate` is not used.

### `backends`

A service allowed to send notifications.


| Column            | Notes                                                                     |
| ----------------- | ------------------------------------------------------------------------- |
| `name`            | unique                                                                    |
| `description`     | optional                                                                  |
| `token_hash`      | unique; SHA-256 of the backend's API token, never the plaintext           |
| `worker_count`    | required, > 0                                                             |
| `default_locale`  | required                                                                  |
| `fcm_credentials` | required; the Firebase service-account JSON, meant to be stored encrypted |




### `stale_tokens`

Device tokens FCM reported as invalid, per backend.

- `(backend_id, token)` is unique across all rows, including soft-deleted ones.
- Soft-deleted via `deleted_at`; GORM excludes deleted rows from queries automatically.
- Deleting a backend deletes its stale tokens (`ON DELETE CASCADE`).



### Connection settings

`storage.OpenSQLite` enables these on every pooled connection: foreign keys, WAL journal mode, a 5-second busy timeout and `synchronous=NORMAL`.

Do not run migration files through GORM's `Exec`: with prepared statements enabled, only the first statement of each file would run. `db.Migrate()` avoids this by handing golang-migrate the underlying `*sql.DB`, which executes each file in full.

## Repositories

Each repository exposes small role interfaces, so consumers depend only on what they use.

**Backends** — `repository.NewBackendRepository(db)`


| Interface       | Methods                                  |
| --------------- | ---------------------------------------- |
| `BackendLoader` | `LoadAll`                                |
| `BackendFinder` | `GetByID`, `GetByName`, `GetByTokenHash` |
| `BackendWriter` | `Create`, `Update`, `Delete`             |


**Stale tokens** — `repository.NewStaleTokenRepository(db)`


| Interface            | Methods                                                                  |
| -------------------- | ------------------------------------------------------------------------ |
| `StaleTokenRecorder` | `Record` — insert new tokens, revive soft-deleted ones, skip active ones |
| `StaleTokenReader`   | `ListByBackend` (keyset pagination), `FindStale`                         |
| `StaleTokenRemover`  | `MarkDeleted` (soft delete), `Purge` (hard-delete old soft-deleted rows) |


Repositories return `repository.ErrNotFound`, `ErrDuplicate` and `ErrReferenceNotFound` instead of GORM errors.

## Crypto



### API tokens

Each backend authenticates with an API token. The database stores only its SHA-256 hash; the plain token is shown once when created.

- `crypto.GenerateToken()` returns a new random token (`pn_` + 32 random bytes, base64url) and its hash.
- `crypto.HashToken(plain)` hashes an incoming token so it can be looked up by `token_hash`.

A fast unsalted hash is used on purpose: tokens are 256-bit random values, so they can't be guessed, and a deterministic hash allows lookup through the unique index.

### FCM credentials

`crypto.Cipher` is the encryption interface. `crypto.NewCipher(algorithm, key)` returns the implementation selected by `ENCRYPTION_ALGORITHM`; the rest of the code only uses the interface.


| Algorithm           | Type                                           | Nonce    | Prefix      |
| ------------------- | ---------------------------------------------- | -------- | ----------- |
| `aes-gcm`           | `crypto.AESGCM` (AES-256-GCM)                  | 12 bytes | `aesgcmv1:` |
| `chacha20-poly1305` | `crypto.Chacha20poly1305` (XChaCha20-Poly1305) | 24 bytes | `chachav1:` |


- `Encrypt(plaintext)` returns the prefix + base64(nonce ‖ ciphertext). A new random nonce is used every time.
- `Decrypt(encoded)` returns the original bytes, or `crypto.ErrInvalidCiphertext` for a wrong key, altered data or another algorithm's prefix.

Each cipher only decrypts values with its own prefix, so changing `ENCRYPTION_ALGORITHM` after credentials are stored requires re-encrypting them.

## Security notes

- **Back up the encryption keys outside the server.** Losing the active key makes every stored FCM credential unrecoverable.

## Testing

There are no tests at the moment.