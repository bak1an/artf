package bblt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"go.etcd.io/bbolt"
)

func TestRepoStoreCreateGetList(t *testing.T) {
	_, st := newTestStore(t)
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
	if list[0].ID != repo.ID {
		t.Fatalf("unexpected list entry ID: got=%d want=%d", list[0].ID, repo.ID)
	}
}

func TestRepoStoreGetByNameAndPath(t *testing.T) {
	_, st := newTestStore(t)
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

func TestRepoStoreUpdateUpdatesIndexes(t *testing.T) {
	_, st := newTestStore(t)
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

}

func TestRepoStoreDeleteRemovesDataAndIndexes(t *testing.T) {
	_, st := newTestStore(t)
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
	_, st := newTestStore(t)
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
	if err := rs.Update(ctx, seedRepo("missing", "/tmp/missing", time.Now(), time.Now())); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Update expected ErrNotFound, got: %v", err)
	}
}

func TestRepoStoreCorruptedIndexEntry(t *testing.T) {
	db, st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	withWriteTx(t, db, func(tx *bbolt.Tx) error {
		idx := tx.Bucket(indexBucket)
		if err := idx.Put(repoNameIndexKey("bad-name"), []byte{1, 2, 3}); err != nil {
			return err
		}
		return nil
	})

	if _, err := rs.GetByName(ctx, "bad-name"); err == nil || errors.Is(err, store.ErrNotFound) || !strings.Contains(err.Error(), "corrupted index entry") {
		t.Fatalf("expected corrupted index error for name, got: %v", err)
	}
}

func TestRepoStoreMissingBucketErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("missing repos bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		rs := st.Repos()

		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(reposBucket)
		})

		_, err := rs.Get(ctx, 1)
		if err == nil || !strings.Contains(err.Error(), "repos bucket not found") {
			t.Fatalf("expected repos bucket error, got: %v", err)
		}
	})

	t.Run("missing index bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		rs := st.Repos()

		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(indexBucket)
		})

		_, err := rs.GetByName(ctx, "x")
		if err == nil || !strings.Contains(err.Error(), "index bucket not found") {
			t.Fatalf("expected index bucket error, got: %v", err)
		}
	})
}

func TestRepoStoreListSkipsUnexpectedNonRecordKeys(t *testing.T) {
	db, st := newTestStore(t)
	rs := st.Repos()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	repo := seedRepo("repo4", "/tmp/repo4", now, now)
	if err := rs.Create(ctx, repo); err != nil {
		t.Fatalf("create repo: %v", err)
	}

	withWriteTx(t, db, func(tx *bbolt.Tx) error {
		return tx.Bucket(reposBucket).Put([]byte("bad-key"), []byte("not-a-repo-record"))
	})

	list, err := rs.List(ctx)
	if err != nil {
		t.Fatalf("list repos: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected only valid repo records in list, got %d", len(list))
	}
	if list[0].ID != repo.ID {
		t.Fatalf("unexpected repo ID in list: got=%d want=%d", list[0].ID, repo.ID)
	}
}
