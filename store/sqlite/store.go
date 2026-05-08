// Package sqlite is the SQLite-backed implementation of store.Store, using
// the pure-Go modernc.org/sqlite driver via database/sql.
//
// IDs stored in [store] types are uint64, but SQLite's INTEGER PRIMARY KEY is
// int64. Since bbolt-produced and SQLite-produced ids start at 1 and realistic
// deployments never approach 2^63, conversions between the two are safe.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bak1an/artf/store"
	_ "modernc.org/sqlite"
)

const (
	dbFileMode = 0600
	// DSN pragmas applied by the driver on connection open.
	dsnSuffix = "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
)

type sqliteStore struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at dbPath, applies PRAGMAs and
// migrations, enforces 0600 perms on the main file and WAL/SHM sidecars, and
// returns a store.Store backed by it.
func Open(ctx context.Context, dbPath string, logger *slog.Logger) (store.Store, error) {
	if err := ensureFile(dbPath); err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}
	dsn := "file:" + abs + dsnSuffix

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := applyMigrations(ctx, db, logger); err != nil {
		_ = db.Close()
		return nil, err
	}

	chmodSidecars(dbPath)

	return &sqliteStore{db: db}, nil
}

// ensureFile makes sure the db file exists with 0600 perms before the driver
// opens it — SQLite would otherwise honor the process umask on create.
func ensureFile(dbPath string) error {
	if _, err := os.Stat(dbPath); err == nil {
		return os.Chmod(dbPath, dbFileMode)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat db file: %w", err)
	}
	f, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, dbFileMode)
	if err != nil {
		return fmt.Errorf("create db file: %w", err)
	}
	return f.Close()
}

func chmodSidecars(dbPath string) {
	for _, suffix := range []string{"-wal", "-shm"} {
		p := dbPath + suffix
		if _, err := os.Stat(p); err == nil {
			_ = os.Chmod(p, dbFileMode)
		}
	}
}

func (s *sqliteStore) Close() error { return s.db.Close() }
func (s *sqliteStore) APIKeys() store.APIKeyStore {
	return &sqliteAPIKeyStore{db: s.db}
}
func (s *sqliteStore) Repos() store.RepoStore {
	return &sqliteRepoStore{db: s.db}
}
func (s *sqliteStore) Artifacts() store.ArtifactStore {
	return &sqliteArtifactStore{db: s.db}
}
