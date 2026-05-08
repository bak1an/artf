package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/bblt"
	"go.etcd.io/bbolt"
)

func TestOpenAndMigrateFreshInstall(t *testing.T) {
	dir := t.TempDir()

	st, err := OpenAndMigrate(context.Background(), dir, discardLogger())
	if err != nil {
		t.Fatalf("OpenAndMigrate: %v", err)
	}
	defer st.Close()

	keys, err := st.APIKeys().List(context.Background())
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected empty keys on fresh install, got %d", len(keys))
	}

	if _, err := os.Stat(filepath.Join(dir, newDBFilename)); err != nil {
		t.Fatalf("expected new DB file to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, oldDBFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no legacy DB to be created, got err=%v", err)
	}
}

func TestOpenAndMigrateCopiesBboltData(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, oldDBFilename)

	// Seed a legacy bbolt DB with mixed data.
	seedLegacyBbolt(t, oldPath)

	st, err := OpenAndMigrate(context.Background(), dir, discardLogger())
	if err != nil {
		t.Fatalf("OpenAndMigrate: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	keys, err := st.APIKeys().List(ctx)
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys migrated, got %d", len(keys))
	}
	// Keys were inserted in this order; IDs should match.
	if keys[0].Name != "ci" || keys[0].ID != 1 {
		t.Fatalf("unexpected key[0]: %+v", keys[0])
	}
	if keys[1].Name != "deploy" || keys[1].ID != 2 {
		t.Fatalf("unexpected key[1]: %+v", keys[1])
	}
	if keys[0].LastUsedAt != nil {
		t.Fatalf("expected LastUsedAt nil, got %v", keys[0].LastUsedAt)
	}
	if keys[1].LastUsedAt == nil {
		t.Fatalf("expected LastUsedAt set on deploy key")
	}

	repos, err := st.Repos().List(ctx)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}
	if len(repos) != 1 || repos[0].ID != 1 || repos[0].Name != "main" {
		t.Fatalf("unexpected repos: %+v", repos)
	}

	artifacts, err := st.Artifacts().List(ctx)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].Name != "v1.tar.gz" || artifacts[0].ID != 1 || artifacts[0].RepoID != 1 {
		t.Fatalf("unexpected artifact[0]: %+v", artifacts[0])
	}

	// Legacy file must remain on disk.
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected legacy DB to remain: %v", err)
	}
}

func TestOpenAndMigrateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	seedLegacyBbolt(t, filepath.Join(dir, oldDBFilename))

	st1, err := OpenAndMigrate(context.Background(), dir, discardLogger())
	if err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("close st1: %v", err)
	}

	st2, err := OpenAndMigrate(context.Background(), dir, discardLogger())
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer st2.Close()

	keys, err := st2.APIKeys().List(context.Background())
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("second open should see migrated keys only, got %d (no duplicate copy)", len(keys))
	}
}

func TestOpenAndMigratePreservesIDsAndContinuesSequence(t *testing.T) {
	dir := t.TempDir()
	seedLegacyBbolt(t, filepath.Join(dir, oldDBFilename))

	st, err := OpenAndMigrate(context.Background(), dir, discardLogger())
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	// Previous max ID for api_keys was 2. New insert should be 3.
	now := time.Now().UTC().Truncate(time.Second)
	fresh := seedAPIKey("post-migration", []byte{0xff}, now, false)
	if err := st.APIKeys().Create(ctx, fresh); err != nil {
		t.Fatalf("create after migration: %v", err)
	}
	if fresh.ID != 3 {
		t.Fatalf("expected sequence to continue past legacy ids; got %d, want 3", fresh.ID)
	}
}

// seedLegacyBbolt writes a bbolt DB at path containing 2 api_keys (one with
// LastUsedAt set, one without), 1 repo, and 2 artifacts. Insertion order is
// preserved so ID assignments are deterministic (1, 2, ...).
func seedLegacyBbolt(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()

	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open bbolt: %v", err)
	}
	oldStore, err := bblt.NewBboltStore(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("new bbolt store: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	k1 := &store.APIKey{Name: "ci", KeyHash: []byte{0x01}, CreatedAt: now, ReadOnly: false}
	if err := oldStore.APIKeys().Create(ctx, k1); err != nil {
		t.Fatalf("create k1: %v", err)
	}
	lu := now.Add(time.Hour)
	k2 := &store.APIKey{Name: "deploy", KeyHash: []byte{0x02}, CreatedAt: now, ReadOnly: true, LastUsedAt: &lu}
	if err := oldStore.APIKeys().Create(ctx, k2); err != nil {
		t.Fatalf("create k2: %v", err)
	}

	repo := &store.Repo{Name: "main", Type: store.RepoTypeFile, Path: "main", KeepCount: 10, KeepDays: 30, CreatedAt: now, UpdatedAt: now}
	if err := oldStore.Repos().Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}

	a1 := &store.Artifact{Name: "v1.tar.gz", RepoID: repo.ID, Path: "main/v1.tar.gz", SHA256: "aaa", CreatedAt: now}
	if err := oldStore.Artifacts().Create(ctx, a1); err != nil {
		t.Fatalf("create a1: %v", err)
	}
	a2 := &store.Artifact{Name: "v2.tar.gz", RepoID: repo.ID, Path: "main/v2.tar.gz", SHA256: "bbb", CreatedAt: now}
	if err := oldStore.Artifacts().Create(ctx, a2); err != nil {
		t.Fatalf("create a2: %v", err)
	}

	if err := oldStore.Close(); err != nil {
		t.Fatalf("close legacy store: %v", err)
	}
}
