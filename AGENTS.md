# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project

`artf` is a minimal home-use artifact management system with two HTTP servers (main API + admin Unix socket) and a CLI.

## Commands

```bash
make build          # Build binary for local OS (default target: clean + build)
make test           # Run tests with -v and -race
make fmt            # Format with go fmt + goimports
make vet            # Run go vet
make check          # Run vet + nils (nilaway) + test
make dist           # Full cross-platform distribution build
```

Run a single test:

```bash
go test -v -run TestName ./path/to/package/...
```

Integration coverage also lives in `tests/integration_test.py`.

CI enforces: `make fmt` (no diff), then `make check`.

## Architecture

Two servers run concurrently from `cmd/serve.go`:

- **Main server** (`server/`): TCP on `127.0.0.1:8365` (or systemd socket). Requires API key auth via `Authorization: Bearer <api-key>` header.
- **Admin server** (`admin/server.go`): Unix socket at `~/.artf/artf.sock`. Local-only, no auth. Used by CLI subcommands.

Both support graceful shutdown via SIGINT/SIGTERM with a 10-second timeout and systemd notifications.

### Data Layer (`store/`)

Interface-based storage defined in `store/store.go`. Current implementation: SQLite via pure-Go `modernc.org/sqlite` (`store/sqlite/`) at `~/.artf/artf1.db` (0600 perms, with WAL/SHM sidecars). Tables: `api_keys`, `repos`, `artifacts`, plus `versions` for schema migrations. IDs are `INTEGER PRIMARY KEY` (cast to/from `uint64` at the Go boundary). Timestamps stored as unix nanoseconds. PRAGMAs applied on open: `journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=NORMAL`.

Schema changes go through an in-house migration framework in `store/sqlite/migrations.go`: an append-only `[]Migration{ID, SQL}` list is applied on every `Open` inside per-migration transactions, tracked in the `versions` table. Never edit or renumber a shipped migration — add a new one.

The legacy BBolt implementation in `store/bblt/` is deprecated and retained only so `store/sqlite/migrate.go` can perform the one-time bbolt→sqlite copy when `artf0.db` exists and `artf1.db` does not. After migration the old file is left on disk untouched.

### Authentication (`internal/auth/`, `server/middleware_auth.go`)

API keys are prefixed with `artf_`, stored as SHA256 hashes. Auth middleware extracts the raw key, hashes it, looks it up in the store, enforces read-only constraints, and async-updates `LastUsedAt` (rate-limited to once per minute).

### Middleware chain (`server/middleware_*.go`)

Applied outermost-first: `RecoverMiddleware` → `RequestID` → `RequestContextLogger` → `LoggingMiddleware` → `AuthMiddleware` (main server only).

### CLI (`cmd/`)

Cobra commands. Admin commands (`keys list`, `status`) connect to the Unix socket via `admin/client.go`. Config via Viper with `ARTF_` env prefix; default data dir: `$HOME/.artf`.

### Internal utilities (`internal/`)

- `ctxlog/`: Attach/retrieve `slog.Logger` from context (fallback to `slog.Default()`).
- `rid/`: Generate 32-char hex request IDs, store/retrieve from context.
- `auth/`: `GenerateKey()` and `HashKey()` (strips `artf_` prefix before hashing).

### Build info

`make build` injects timestamp, git revision, branch, and tag via ldflags into `version/`.
