package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
)

func TestRepoStoreCreateGetList(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	repo := seedRepo("repo1", "/tmp/repo1", now, now)
	if err := rs.Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if repo.ID == 0 {
		t.Fatal("expected repo ID to be assigned")
	}

	got, err := rs.Get(ctx, repo.ID)
	if err != nil {
		t.Fatalf("get repo: %v", err)
	}
	if got.ID != repo.ID || got.Name != repo.Name || got.Path != repo.Path || got.Type != repo.Type {
		t.Fatalf("unexpected repo from Get: %+v", got)
	}
	if !got.CreatedAt.Equal(repo.CreatedAt) || !got.UpdatedAt.Equal(repo.UpdatedAt) {
		t.Fatalf("unexpected timestamps: got=%v/%v want=%v/%v", got.CreatedAt, got.UpdatedAt, repo.CreatedAt, repo.UpdatedAt)
	}

	list, err := rs.List(ctx)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 repo in list, got %d", len(list))
	}
}

func TestRepoStoreGetByName(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	repo := seedRepo("repo2", "/tmp/repo2", now, now)
	if err := rs.Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}

	byName, err := rs.GetByName(ctx, repo.Name)
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if byName.ID != repo.ID {
		t.Fatalf("GetByName returned wrong ID: got=%d want=%d", byName.ID, repo.ID)
	}
}

func TestRepoStoreUpdate(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	repo := seedRepo("old-repo", "/tmp/old-repo", now, now)
	if err := rs.Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}

	updated := *repo
	updated.Name = "new-repo"
	updated.Path = "/tmp/new-repo"
	updated.UpdatedAt = now.Add(time.Minute)
	if err := rs.Update(ctx, &updated); err != nil {
		t.Fatalf("update repo: %v", err)
	}

	if _, err := rs.GetByName(ctx, "old-repo"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected old name lookup to fail with ErrNotFound, got: %v", err)
	}

	byName, err := rs.GetByName(ctx, "new-repo")
	if err != nil {
		t.Fatalf("get by new name: %v", err)
	}
	if byName.ID != repo.ID || byName.Path != "/tmp/new-repo" {
		t.Fatalf("unexpected repo from GetByName: %+v", byName)
	}
	if !byName.UpdatedAt.Equal(updated.UpdatedAt) {
		t.Fatalf("unexpected UpdatedAt after update: got=%v want=%v", byName.UpdatedAt, updated.UpdatedAt)
	}
}

func TestRepoStoreDelete(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	repo := seedRepo("repo3", "/tmp/repo3", now, now)
	if err := rs.Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}

	if err := rs.Delete(ctx, repo.ID); err != nil {
		t.Fatalf("delete repo: %v", err)
	}

	if _, err := rs.Get(ctx, repo.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected Get to fail with ErrNotFound, got: %v", err)
	}
	if _, err := rs.GetByName(ctx, repo.Name); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected GetByName to fail with ErrNotFound, got: %v", err)
	}
}

func TestRepoStoreNotFoundCases(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	if _, err := rs.Get(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get expected ErrNotFound, got: %v", err)
	}
	if _, err := rs.GetByName(ctx, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetByName expected ErrNotFound, got: %v", err)
	}
	if err := rs.Delete(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete expected ErrNotFound, got: %v", err)
	}
	missing := seedRepo("missing", "/tmp/missing", time.Now(), time.Now())
	missing.ID = 999
	if err := rs.Update(ctx, missing); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Update expected ErrNotFound, got: %v", err)
	}
}

func TestRepoStoreDuplicateNameReturnsAlreadyExists(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	if err := rs.Create(ctx, seedRepo("dup", "/tmp/a", now, now)); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := rs.Create(ctx, seedRepo("dup", "/tmp/b", now, now))
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on duplicate name, got: %v", err)
	}
}

func TestRepoStoreUpdateNameCollisionReturnsAlreadyExists(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := seedRepo("alpha", "/tmp/a", now, now)
	b := seedRepo("beta", "/tmp/b", now, now)
	for _, r := range []*store.Repo{a, b} {
		if err := rs.Create(ctx, r); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	b.Name = "alpha"
	err := rs.Update(ctx, b)
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on rename collision, got: %v", err)
	}
}

func TestRepoStoreDeleteBlockedByArtifactFK(t *testing.T) {
	st := newTestStore(t)
	ensureRepos(t, st, 1)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	if err := st.Artifacts().Create(ctx, a); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	// ON DELETE RESTRICT: deleting a repo with artifacts must fail.
	if err := st.Repos().Delete(ctx, 1); err == nil {
		t.Fatal("expected FK constraint to block repo delete, got nil")
	}

	// After clearing artifacts, the delete should succeed.
	if err := st.Artifacts().DeleteByRepo(ctx, 1); err != nil {
		t.Fatalf("delete artifacts: %v", err)
	}
	if err := st.Repos().Delete(ctx, 1); err != nil {
		t.Fatalf("delete repo after clearing artifacts: %v", err)
	}
}

func TestRepoStoreIDsMonotonicAfterDelete(t *testing.T) {
	st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	a := seedRepo("a", "/tmp/a", now, now)
	b := seedRepo("b", "/tmp/b", now, now)
	for _, r := range []*store.Repo{a, b} {
		if err := rs.Create(ctx, r); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if err := rs.Delete(ctx, b.ID); err != nil {
		t.Fatalf("delete b: %v", err)
	}

	// Without AUTOINCREMENT, SQLite would reuse b.ID here. With AUTOINCREMENT,
	// the new row must get a strictly greater id than any ever issued.
	c := seedRepo("c", "/tmp/c", now, now)
	if err := rs.Create(ctx, c); err != nil {
		t.Fatalf("create c: %v", err)
	}
	if c.ID <= b.ID {
		t.Fatalf("expected new repo ID > %d (monotonic), got %d", b.ID, c.ID)
	}
}
