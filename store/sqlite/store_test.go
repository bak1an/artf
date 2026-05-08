package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesFileWith0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "perm.db")

	st, err := Open(context.Background(), path, discardLogger())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 perms, got %v", info.Mode().Perm())
	}
}

func TestOpenReopenExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")

	st1, err := Open(context.Background(), path, discardLogger())
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	st2, err := Open(context.Background(), path, discardLogger())
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	if err := st2.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestStoreProvidesSubStores(t *testing.T) {
	st := newTestStore(t)

	if st.APIKeys() == nil {
		t.Fatal("APIKeys() returned nil")
	}
	if st.Repos() == nil {
		t.Fatal("Repos() returned nil")
	}
	if st.Artifacts() == nil {
		t.Fatal("Artifacts() returned nil")
	}
}
