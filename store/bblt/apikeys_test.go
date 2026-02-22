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

func TestAPIKeyStoreCreateGetList(t *testing.T) {
	_, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	key := seedAPIKey("ci", []byte{0x01, 0x02, 0x03}, now, false)
	if err := ks.Create(ctx, key); err != nil {
		t.Fatalf("create API key: %v", err)
	}
	if key.ID == 0 {
		t.Fatal("expected API key ID to be assigned")
	}

	got, err := ks.Get(ctx, key.ID)
	if err != nil {
		t.Fatalf("get API key: %v", err)
	}
	if got.ID != key.ID || got.Name != key.Name || got.ReadOnly != key.ReadOnly || !got.CreatedAt.Equal(key.CreatedAt) {
		t.Fatalf("unexpected API key from Get: %+v", got)
	}

	list, err := ks.List(ctx)
	if err != nil {
		t.Fatalf("list API keys: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 key in list, got %d", len(list))
	}
	if list[0].ID != key.ID {
		t.Fatalf("unexpected list entry ID: got=%d want=%d", list[0].ID, key.ID)
	}
}

func TestAPIKeyStoreGetByKey(t *testing.T) {
	_, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	key := seedAPIKey("deploy", []byte{0xaa, 0xbb, 0xcc}, now, true)
	if err := ks.Create(ctx, key); err != nil {
		t.Fatalf("create API key: %v", err)
	}

	got, err := ks.GetByKey(ctx, key.KeyHash)
	if err != nil {
		t.Fatalf("get API key by key hash: %v", err)
	}
	if got.ID != key.ID || got.Name != key.Name {
		t.Fatalf("unexpected API key from GetByKey: %+v", got)
	}
}

func TestAPIKeyStoreUpdateLastUsed(t *testing.T) {
	_, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	key := seedAPIKey("ops", []byte{0x10, 0x20}, now, false)
	if err := ks.Create(ctx, key); err != nil {
		t.Fatalf("create API key: %v", err)
	}

	lastUsed := now.Add(15 * time.Minute)
	if err := ks.UpdateLastUsed(ctx, key.ID, lastUsed); err != nil {
		t.Fatalf("update last used: %v", err)
	}

	got, err := ks.Get(ctx, key.ID)
	if err != nil {
		t.Fatalf("get API key: %v", err)
	}
	if got.LastUsedAt == nil {
		t.Fatal("expected LastUsedAt to be set")
	}
	if !got.LastUsedAt.Equal(lastUsed) {
		t.Fatalf("unexpected LastUsedAt: got=%v want=%v", *got.LastUsedAt, lastUsed)
	}
}

func TestAPIKeyStoreDeleteRemovesDataAndIndex(t *testing.T) {
	_, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	key := seedAPIKey("temp", []byte{0x44, 0x55}, now, false)
	if err := ks.Create(ctx, key); err != nil {
		t.Fatalf("create API key: %v", err)
	}

	if err := ks.Delete(ctx, key.ID); err != nil {
		t.Fatalf("delete API key: %v", err)
	}

	if _, err := ks.Get(ctx, key.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected Get to fail with ErrNotFound, got: %v", err)
	}

	if _, err := ks.GetByKey(ctx, key.KeyHash); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected GetByKey to fail with ErrNotFound, got: %v", err)
	}
}

func TestAPIKeyStoreNotFoundCases(t *testing.T) {
	_, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	if _, err := ks.Get(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get expected ErrNotFound, got: %v", err)
	}
	if err := ks.Delete(ctx, 999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete expected ErrNotFound, got: %v", err)
	}
	if err := ks.UpdateLastUsed(ctx, 999, time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("UpdateLastUsed expected ErrNotFound, got: %v", err)
	}
	if _, err := ks.GetByKey(ctx, []byte("missing")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetByKey expected ErrNotFound, got: %v", err)
	}
}

func TestAPIKeyStoreCorruptedIndexEntry(t *testing.T) {
	db, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	withWriteTx(t, db, func(tx *bbolt.Tx) error {
		return tx.Bucket(indexBucket).Put(apiKeyIndexKey([]byte("bad-key")), []byte{9, 9})
	})

	if _, err := ks.GetByKey(ctx, []byte("bad-key")); err == nil || errors.Is(err, store.ErrNotFound) || !strings.Contains(err.Error(), "corrupted index entry") {
		t.Fatalf("expected corrupted index error, got: %v", err)
	}
}

func TestAPIKeyStoreMissingBucketErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("missing API keys bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		ks := st.APIKeys()

		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(apiKeysBucket)
		})

		_, err := ks.Get(ctx, 1)
		if err == nil || !strings.Contains(err.Error(), "API keys bucket not found") {
			t.Fatalf("expected API keys bucket error, got: %v", err)
		}
	})

	t.Run("missing index bucket", func(t *testing.T) {
		db, st := newTestStore(t)
		ks := st.APIKeys()

		withWriteTx(t, db, func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(indexBucket)
		})

		_, err := ks.GetByKey(ctx, []byte("any"))
		if err == nil || !strings.Contains(err.Error(), "index bucket not found") {
			t.Fatalf("expected index bucket error, got: %v", err)
		}
	})
}

func TestAPIKeyStoreListSkipsUnexpectedNonRecordKeys(t *testing.T) {
	db, st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	key := seedAPIKey("auditor", []byte{0x99}, now, true)
	if err := ks.Create(ctx, key); err != nil {
		t.Fatalf("create API key: %v", err)
	}

	withWriteTx(t, db, func(tx *bbolt.Tx) error {
		return tx.Bucket(apiKeysBucket).Put([]byte("bad-key"), []byte("not-an-apikey-record"))
	})

	list, err := ks.List(ctx)
	if err != nil {
		t.Fatalf("list API keys: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected only valid API key records in list, got %d", len(list))
	}
	if list[0].ID != key.ID {
		t.Fatalf("unexpected key ID in list: got=%d want=%d", list[0].ID, key.ID)
	}
}
