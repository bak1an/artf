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

func TestArtifactStoreCreateGetList(t *testing.T) {
	_, st := newTestStore(t)
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
		t.Fatalf("expected 1 artifact in list, got %d", len(list))
	}
	if list[0].ID != artifact.ID {
		t.Fatalf("unexpected list entry ID: got=%d want=%d", list[0].ID, artifact.ID)
	}
}

func TestArtifactStoreListByRepo(t *testing.T) {
	_, st := newTestStore(t)
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

	repo2List, err := as.ListByRepo(ctx, 2)
	if err != nil {
		t.Fatalf("list by repo 2: %v", err)
	}
	if len(repo2List) != 1 {
		t.Fatalf("expected 1 artifact for repo 2, got %d", len(repo2List))
	}
	if repo2List[0].ID != a3.ID {
		t.Fatalf("unexpected artifact in repo 2 list: got=%d want=%d", repo2List[0].ID, a3.ID)
	}

	emptyList, err := as.ListByRepo(ctx, 99)
	if err != nil {
		t.Fatalf("list by repo 99: %v", err)
	}
	if len(emptyList) != 0 {
		t.Fatalf("expected empty list for unknown repo, got %d", len(emptyList))
	}
}

func TestArtifactStoreDeleteRemovesData(t *testing.T) {
	_, st := newTestStore(t)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	artifact := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	if err := as.Create(ctx, artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	if err := as.Delete(ctx, artifact.ID); err != nil {
		t.Fatalf("delete artifact: %v", err)
	}

	if _, err := as.Get(ctx, artifact.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected Get to fail with ErrNotFound after delete, got: %v", err)
	}
}

func TestArtifactStoreNotFoundCases(t *testing.T) {
	_, st := newTestStore(t)
	as := st.Artifacts()
	ctx := context.Background()

	if _, err := as.Get(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get expected ErrNotFound, got: %v", err)
	}
	if err := as.Delete(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete expected ErrNotFound, got: %v", err)
	}
}

func TestArtifactStoreMissingBucketErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("Create missing bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		as := st.Artifacts()
		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(artifactsBucket)
		})
		err := as.Create(ctx, seedArtifact("x", "/x", 1, time.Now()))
		if err == nil || !strings.Contains(err.Error(), "artifacts bucket not found") {
			t.Fatalf("expected artifacts bucket error, got: %v", err)
		}
	})

	t.Run("Get missing bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		as := st.Artifacts()
		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(artifactsBucket)
		})
		_, err := as.Get(ctx, 1)
		if err == nil || !strings.Contains(err.Error(), "artifacts bucket not found") {
			t.Fatalf("expected artifacts bucket error, got: %v", err)
		}
	})

	t.Run("List missing bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		as := st.Artifacts()
		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(artifactsBucket)
		})
		_, err := as.List(ctx)
		if err == nil || !strings.Contains(err.Error(), "artifacts bucket not found") {
			t.Fatalf("expected artifacts bucket error, got: %v", err)
		}
	})

	t.Run("ListByRepo missing bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		as := st.Artifacts()
		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(artifactsBucket)
		})
		_, err := as.ListByRepo(ctx, 1)
		if err == nil || !strings.Contains(err.Error(), "artifacts bucket not found") {
			t.Fatalf("expected artifacts bucket error, got: %v", err)
		}
	})

	t.Run("Delete missing bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		as := st.Artifacts()
		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(artifactsBucket)
		})
		err := as.Delete(ctx, 1)
		if err == nil || !strings.Contains(err.Error(), "artifacts bucket not found") {
			t.Fatalf("expected artifacts bucket error, got: %v", err)
		}
	})
}

func TestArtifactStoreGetByRepoAndName(t *testing.T) {
	_, st := newTestStore(t)
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

	_, err = as.GetByRepoAndName(ctx, 1, "nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}

	_, err = as.GetByRepoAndName(ctx, 99, "v1.tar.gz")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown repo, got: %v", err)
	}
}

func TestArtifactStoreGetLatestByRepo(t *testing.T) {
	_, st := newTestStore(t)
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

	_, err = as.GetLatestByRepo(ctx, 99)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty repo, got: %v", err)
	}
}

func TestArtifactStoreListSkipsUnexpectedKeys(t *testing.T) {
	db, st := newTestStore(t)
	as := st.Artifacts()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	artifact := seedArtifact("v1.tar.gz", "/tmp/repo1/v1.tar.gz", 1, now)
	if err := as.Create(ctx, artifact); err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	withWriteTx(t, db, func(tx *bbolt.Tx) error {
		return tx.Bucket(artifactsBucket).Put([]byte("bad-key"), []byte("not-a-record"))
	})

	list, err := as.List(ctx)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected only valid artifact records in list, got %d", len(list))
	}
}
