package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
)

// Migration is one step in the in-house schema-versioning system. The SQL runs
// once, inside a dedicated transaction, then the id is recorded in the
// versions table. The list is append-only — never edit or renumber a shipped
// migration; add a new one instead.
type Migration struct {
	ID  int
	SQL string
}

const migration1SQL = `
CREATE TABLE api_keys (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    key_hash     BLOB    NOT NULL,
    name         TEXT    NOT NULL,
    read_only    INTEGER NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER
);
CREATE UNIQUE INDEX api_keys_key_hash_idx ON api_keys(key_hash);

CREATE TABLE repos (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    type       TEXT    NOT NULL,
    path       TEXT    NOT NULL,
    keep_count INTEGER NOT NULL,
    keep_days  INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX repos_name_idx ON repos(name);

CREATE TABLE artifacts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    repo_id    INTEGER NOT NULL REFERENCES repos(id) ON DELETE RESTRICT,
    path       TEXT    NOT NULL,
    sha256     TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX artifacts_repo_name_idx ON artifacts(repo_id, name);
CREATE INDEX        artifacts_repo_id_idx   ON artifacts(repo_id, id DESC);
`

var migrations = []Migration{
	{ID: 1, SQL: migration1SQL},
}

func applyMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS versions (id INTEGER PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create versions table: %w", err)
	}

	applied, err := loadAppliedVersions(ctx, db)
	if err != nil {
		return err
	}

	known := make(map[int]struct{}, len(migrations))
	for _, m := range migrations {
		known[m.ID] = struct{}{}
	}
	for id := range applied {
		if _, ok := known[id]; !ok {
			return fmt.Errorf("database contains migration id %d which is unknown to this binary (downgrade?)", id)
		}
	}

	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	for _, m := range sorted {
		if _, ok := applied[m.ID]; ok {
			continue
		}
		if err := applyOne(ctx, db, m); err != nil {
			return fmt.Errorf("apply migration %d: %w", m.ID, err)
		}
		logger.Info("applied migration", "id", m.ID)
	}
	return nil
}

func loadAppliedVersions(ctx context.Context, db *sql.DB) (map[int]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT id FROM versions`)
	if err != nil {
		return nil, fmt.Errorf("select versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]struct{})
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan version id: %w", err)
		}
		applied[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func applyOne(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO versions(id) VALUES(?)`, m.ID); err != nil {
		return err
	}
	return tx.Commit()
}
