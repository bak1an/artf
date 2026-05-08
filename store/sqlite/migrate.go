package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/bblt"
	"go.etcd.io/bbolt"
)

const (
	oldDBFilename = "artf0.db"
	newDBFilename = "artf1.db"

	bboltOpenTimeout = 5 * time.Second
)

// OpenAndMigrate is the entry point used by the server. It:
//
//  1. Opens the SQLite DB at <dataDir>/artf1.db (creating it + schema if needed).
//  2. If the legacy <dataDir>/artf0.db exists and artf1.db did not exist
//     before this call, copies api_keys, repos, and artifacts from the bbolt
//     file into the new SQLite DB (IDs preserved) inside one transaction.
//  3. Leaves artf0.db on disk untouched; never re-reads it after migration.
//
// On a partial failure in step 2, artf1.db and its WAL/SHM sidecars are
// removed so the next start retries cleanly.
func OpenAndMigrate(ctx context.Context, dataDir string, logger *slog.Logger) (store.Store, error) {
	oldPath := filepath.Join(dataDir, oldDBFilename)
	newPath := filepath.Join(dataDir, newDBFilename)

	newExists, err := fileExists(newPath)
	if err != nil {
		return nil, err
	}
	oldExists, err := fileExists(oldPath)
	if err != nil {
		return nil, err
	}

	if newExists {
		if oldExists {
			logger.Info("legacy artf0.db present alongside artf1.db; migration already complete; artf0.db may be archived", "old", oldPath)
		}
		return Open(ctx, newPath, logger)
	}

	if !oldExists {
		return Open(ctx, newPath, logger)
	}

	logger.Info("legacy artf0.db detected, starting one-time migration to artf1.db", "old", oldPath, "new", newPath)

	newStore, err := Open(ctx, newPath, logger)
	if err != nil {
		return nil, fmt.Errorf("open new sqlite db: %w", err)
	}
	ss := newStore.(*sqliteStore)

	if err := runBboltMigration(ctx, ss.db, oldPath, logger); err != nil {
		_ = newStore.Close()
		removeSQLiteFiles(newPath, logger)
		return nil, err
	}

	return newStore, nil
}

func runBboltMigration(ctx context.Context, sqliteDB *sql.DB, oldPath string, logger *slog.Logger) error {
	oldDB, err := bbolt.Open(oldPath, dbFileMode, &bbolt.Options{
		Timeout:  bboltOpenTimeout,
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("open legacy bbolt db: %w", err)
	}
	defer func() { _ = oldDB.Close() }()

	oldStore := bblt.NewBboltStoreReadOnly(oldDB)

	tx, err := sqliteDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	keys, err := oldStore.APIKeys().List(ctx)
	if err != nil {
		return fmt.Errorf("list legacy api keys: %w", err)
	}
	for _, k := range keys {
		var lastUsed sql.NullInt64
		if k.LastUsedAt != nil {
			lastUsed = sql.NullInt64{Int64: k.LastUsedAt.UnixNano(), Valid: true}
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO api_keys (id, key_hash, name, read_only, created_at, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			int64(k.ID), k.KeyHash, k.Name, boolToInt(k.ReadOnly), k.CreatedAt.UnixNano(), lastUsed,
		)
		if err != nil {
			return fmt.Errorf("insert api key id=%d: %w", k.ID, err)
		}
	}

	repos, err := oldStore.Repos().List(ctx)
	if err != nil {
		return fmt.Errorf("list legacy repos: %w", err)
	}
	for _, r := range repos {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO repos (id, name, type, path, keep_count, keep_days, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			int64(r.ID), r.Name, string(r.Type), r.Path, r.KeepCount, r.KeepDays,
			r.CreatedAt.UnixNano(), r.UpdatedAt.UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert repo id=%d: %w", r.ID, err)
		}
	}

	artifacts, err := oldStore.Artifacts().List(ctx)
	if err != nil {
		return fmt.Errorf("list legacy artifacts: %w", err)
	}
	for _, a := range artifacts {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO artifacts (id, name, repo_id, path, sha256, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			int64(a.ID), a.Name, int64(a.RepoID), a.Path, a.SHA256, a.CreatedAt.UnixNano(),
		)
		if err != nil {
			return fmt.Errorf("insert artifact id=%d: %w", a.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration tx: %w", err)
	}

	logger.Info("migration complete", "keys", len(keys), "repos", len(repos), "artifacts", len(artifacts))
	return nil
}

func fileExists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat %s: %w", p, err)
}

func removeSQLiteFiles(dbPath string, logger *slog.Logger) {
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Warn("failed to remove partial sqlite file", "path", p, "error", err)
		}
	}
}
