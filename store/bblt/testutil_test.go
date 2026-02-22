package bblt

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

func newTestBoltDB(t *testing.T) *bbolt.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "artf-test.db")
	db, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("open bbolt db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func newTestStore(t *testing.T) (*bbolt.DB, store.Store) {
	t.Helper()

	db := newTestBoltDB(t)
	st, err := NewBboltStore(db)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	return db, st
}

func withWriteTx(t *testing.T, db *bbolt.DB, fn func(tx *bbolt.Tx) error) {
	t.Helper()

	if err := db.Update(fn); err != nil {
		t.Fatalf("db update: %v", err)
	}
}

func seedRepo(name, path string, createdAt, updatedAt time.Time) *store.Repo {
	return &store.Repo{
		Name:      name,
		Type:      store.RepoTypeFile,
		Path:      path,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func seedAPIKey(name string, keyHash []byte, createdAt time.Time, readOnly bool) *store.APIKey {
	return &store.APIKey{
		Name:      name,
		KeyHash:   keyHash,
		CreatedAt: createdAt,
		ReadOnly:  readOnly,
	}
}
