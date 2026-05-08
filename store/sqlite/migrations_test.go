package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestApplyMigrationsCreatesVersionsAndRunsAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := sql.Open("sqlite", "file:"+path+dsnSuffix)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := applyMigrations(ctx, db, discardLogger()); err != nil {
		t.Fatalf("applyMigrations: %v", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT id FROM versions ORDER BY id`)
	if err != nil {
		t.Fatalf("query versions: %v", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) != len(migrations) {
		t.Fatalf("expected %d version rows, got %d", len(migrations), len(ids))
	}
	for i, m := range migrations {
		if ids[i] != m.ID {
			t.Fatalf("version[%d]=%d; want %d", i, ids[i], m.ID)
		}
	}
}

func TestApplyMigrationsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := sql.Open("sqlite", "file:"+path+dsnSuffix)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := applyMigrations(ctx, db, discardLogger()); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := applyMigrations(ctx, db, discardLogger()); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM versions`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != len(migrations) {
		t.Fatalf("expected %d rows after re-apply, got %d", len(migrations), n)
	}
}

func TestApplyMigrationsRejectsUnknownVersionInDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := sql.Open("sqlite", "file:"+path+dsnSuffix)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := applyMigrations(ctx, db, discardLogger()); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// Simulate a version row from a newer binary that this build doesn't know about.
	if _, err := db.ExecContext(ctx, `INSERT INTO versions(id) VALUES (999999)`); err != nil {
		t.Fatalf("insert future version: %v", err)
	}

	err = applyMigrations(ctx, db, discardLogger())
	if err == nil || !strings.Contains(err.Error(), "unknown to this binary") {
		t.Fatalf("expected downgrade guard error, got: %v", err)
	}
}

func TestApplyOneRollbackOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := sql.Open("sqlite", "file:"+path+dsnSuffix)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS versions (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create versions: %v", err)
	}

	bad := Migration{ID: 42, SQL: `CREATE TABLE ok (id INTEGER); CREATE TABLE definitely not valid sql;`}
	err = applyOne(ctx, db, bad)
	if err == nil {
		t.Fatal("expected applyOne to fail on bad SQL")
	}

	// The "ok" table should not persist, and the version row should not be recorded.
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='ok'`).Scan(&n); err != nil {
		t.Fatalf("query master: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected rollback to drop ok table, but %d remain", n)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM versions WHERE id=42`).Scan(&n); err != nil {
		t.Fatalf("query versions: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected versions row NOT written after rollback, but found %d", n)
	}
}
