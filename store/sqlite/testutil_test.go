package sqlite

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(context.Background(), path, discardLogger())
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

func seedArtifact(name, path string, repoID uint64, createdAt time.Time) *store.Artifact {
	return &store.Artifact{
		Name:      name,
		Path:      path,
		RepoID:    repoID,
		CreatedAt: createdAt,
	}
}

// ensureRepos creates n repos so that subsequent artifact inserts with
// RepoID 1..n satisfy the repo_id foreign key. Relies on AUTOINCREMENT
// starting at 1 for an empty repos table.
func ensureRepos(t *testing.T, st store.Store, n int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	for i := 1; i <= n; i++ {
		r := seedRepo(fmt.Sprintf("repo%d", i), fmt.Sprintf("/tmp/repo%d", i), now, now)
		if err := st.Repos().Create(ctx, r); err != nil {
			t.Fatalf("seed repo %d: %v", i, err)
		}
		if r.ID != uint64(i) {
			t.Fatalf("seed repo %d: expected ID %d, got %d", i, i, r.ID)
		}
	}
}
