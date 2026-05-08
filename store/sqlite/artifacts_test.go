package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
)

func TestArtifactStoreCreateGetList(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 1)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	artifact := seedArtifact("v1.0.tar.gz", "/tmp/repo1/v1.0.tar.gz", 1, now)
	if err := as.Create(ctx, artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}
	if artifact.ID == 0 {
		t.Fatal("expected artifact ID to be assigned")
	}

	got, err := as.Get(ctx, artifact.ID)
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if got.ID != artifact.ID || got.Name != artifact.Name || got.Path != artifact.Path || got.RepoID != artifact.RepoID {
		t.Fatalf("unexpected artifact from Get: %+v", got)
	}
	if !got.CreatedAt.Equal(artifact.CreatedAt) {
		t.Fatalf("unexpected CreatedAt: got=%v want=%v", got.CreatedAt, artifact.CreatedAt)
	}

	list, err := as.List(ctx)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(list))
	}
}

func TestArtifactStoreListByRepoAndCount(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 2)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a1 := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	a2 := seedArtifact("v2.tar.gz", "/tmp/repo1/v2.tar.gz", 1, now)
	a3 := seedArtifact("other.tar.gz", "/tmp/repo2/other.tar.gz", 2, now)
	for _, a := range []*store.Artifact{a1, a2, a3} {
		if err := as.Create(ctx, a); err != nil {
			t.Fatalf("create artifact: %v", err)
		}
	}

	repo1List, err := as.ListByRepo(ctx, 1)
	if err != nil {
		t.Fatalf("list by repo 1: %v", err)
	}
	if len(repo1List) != 2 {
		t.Fatalf("expected 2 artifacts for repo 1, got %d", len(repo1List))
	}

	if n, err := as.CountByRepo(ctx, 1); err != nil || n != 2 {
		t.Fatalf("CountByRepo(1)=%d,err=%v; want=2", n, err)
	}
	if n, err := as.CountByRepo(ctx, 2); err != nil || n != 1 {
		t.Fatalf("CountByRepo(2)=%d,err=%v; want=1", n, err)
	}
	if n, err := as.CountByRepo(ctx, 99); err != nil || n != 0 {
		t.Fatalf("CountByRepo(99)=%d,err=%v; want=0", n, err)
	}

	emptyList, err := as.ListByRepo(ctx, 99)
	if err != nil {
		t.Fatalf("list by repo 99: %v", err)
	}
	if len(emptyList) != 0 {
		t.Fatalf("expected empty list for unknown repo, got %d", len(emptyList))
	}
}

func TestArtifactStoreGetByRepoAndName(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 2)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a1 := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	a2 := seedArtifact("v2.tar.gz", "/tmp/repo1/v2.tar.gz", 1, now)
	a3 := seedArtifact("v1.tar.gz", "/tmp/repo2/v1.tar.gz", 2, now)
	for _, a := range []*store.Artifact{a1, a2, a3} {
		if err := as.Create(ctx, a); err != nil {
			t.Fatalf("create artifact: %v", err)
		}
	}

	got, err := as.GetByRepoAndName(ctx, 1, "v1.tar.gz")
	if err != nil {
		t.Fatalf("GetByRepoAndName: %v", err)
	}
	if got.ID != a1.ID {
		t.Fatalf("expected artifact ID %d, got %d", a1.ID, got.ID)
	}

	got, err = as.GetByRepoAndName(ctx, 2, "v1.tar.gz")
	if err != nil {
		t.Fatalf("GetByRepoAndName repo2: %v", err)
	}
	if got.ID != a3.ID {
		t.Fatalf("expected artifact ID %d, got %d", a3.ID, got.ID)
	}

	if _, err = as.GetByRepoAndName(ctx, 1, "nonexistent"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
	if _, err = as.GetByRepoAndName(ctx, 99, "v1.tar.gz"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown repo, got: %v", err)
	}
}

func TestArtifactStoreGetLatestByRepo(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 2)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a1 := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	a2 := seedArtifact("v2.tar.gz", "/tmp/repo1/v2.tar.gz", 1, now)
	a3 := seedArtifact("other.tar.gz", "/tmp/repo2/other.tar.gz", 2, now)
	for _, a := range []*store.Artifact{a1, a2, a3} {
		if err := as.Create(ctx, a); err != nil {
			t.Fatalf("create artifact: %v", err)
		}
	}

	got, err := as.GetLatestByRepo(ctx, 1)
	if err != nil {
		t.Fatalf("GetLatestByRepo: %v", err)
	}
	if got.ID != a2.ID {
		t.Fatalf("expected latest to be %d (v2), got %d", a2.ID, got.ID)
	}

	got, err = as.GetLatestByRepo(ctx, 2)
	if err != nil {
		t.Fatalf("GetLatestByRepo repo2: %v", err)
	}
	if got.ID != a3.ID {
		t.Fatalf("expected latest to be %d, got %d", a3.ID, got.ID)
	}

	if _, err = as.GetLatestByRepo(ctx, 99); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty repo, got: %v", err)
	}
}

func TestArtifactStoreDeleteAndDeleteByRepo(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 2)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a1 := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	a2 := seedArtifact("v2.tar.gz", "/tmp/repo1/v2.tar.gz", 1, now)
	a3 := seedArtifact("v3.tar.gz", "/tmp/repo2/v3.tar.gz", 2, now)
	for _, a := range []*store.Artifact{a1, a2, a3} {
		if err := as.Create(ctx, a); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	if err := as.Delete(ctx, a1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := as.Get(ctx, a1.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}

	if err := as.DeleteByRepo(ctx, 1); err != nil {
		t.Fatalf("delete by repo: %v", err)
	}
	if _, err := as.Get(ctx, a2.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected a2 to be gone after DeleteByRepo, got: %v", err)
	}
	if _, err := as.Get(ctx, a3.ID); err != nil {
		t.Fatalf("expected a3 (different repo) to remain, got err: %v", err)
	}
}

func TestArtifactStoreNotFoundCases(t *testing.T) {
	st := newTestStore(t)
	as := st.Artifacts()
	ctx := context.Background()

	if _, err := as.Get(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get expected ErrNotFound, got: %v", err)
	}
	if err := as.Delete(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete expected ErrNotFound, got: %v", err)
	}
}

func TestArtifactStoreDuplicateNameInRepoReturnsAlreadyExists(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 2)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	if err := as.Create(ctx, a); err != nil {
		t.Fatalf("first create: %v", err)
	}
	dup := seedArtifact("v1.tar.gz", "/tmp/repo1/v1-again.tar.gz", 1, now)
	err := as.Create(ctx, dup)
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on duplicate (repo_id,name), got: %v", err)
	}

	// Same name in a different repo should succeed.
	otherRepo := seedArtifact("v1.tar.gz", "/tmp/repo2/v1.tar.gz", 2, now)
	if err := as.Create(ctx, otherRepo); err != nil {
		t.Fatalf("create in other repo: %v", err)
	}
}

func TestArtifactStoreRejectsUnknownRepoID(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 1)
	as := st.Artifacts()
	ctx := context.Background()

	// repo_id 999 doesn't exist; FK must reject.
	orphan := seedArtifact("v1.tar.gz", "/tmp/orphan/v1.tar.gz", 999, time.Now().UTC())
	if err := as.Create(ctx, orphan); err == nil {
		t.Fatal("expected FK constraint to block insert with unknown repo_id, got nil")
	}
}

func TestArtifactStoreIDsMonotonicAfterDelete(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 1)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a1 := seedArtifact("a1", "/tmp/repo1/a1", 1, now)
	a2 := seedArtifact("a2", "/tmp/repo1/a2", 1, now)
	for _, a := range []*store.Artifact{a1, a2} {
		if err := as.Create(ctx, a); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if err := as.Delete(ctx, a2.ID); err != nil {
		t.Fatalf("delete a2: %v", err)
	}

	a3 := seedArtifact("a3", "/tmp/repo1/a3", 1, now)
	if err := as.Create(ctx, a3); err != nil {
		t.Fatalf("create a3: %v", err)
	}
	if a3.ID <= a2.ID {
		t.Fatalf("expected new artifact ID > %d (monotonic), got %d", a2.ID, a3.ID)
	}
}
