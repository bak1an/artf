package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
)

func TestAPIKeyStoreCreateGetList(t *testing.T) {
	st := newTestStore(t)
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
	if got.LastUsedAt != nil {
		t.Fatalf("expected LastUsedAt nil, got %v", got.LastUsedAt)
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
	st := newTestStore(t)
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
	if got.ID != key.ID || got.Name != key.Name || got.ReadOnly != key.ReadOnly {
		t.Fatalf("unexpected API key from GetByKey: %+v", got)
	}
}

func TestAPIKeyStoreUpdateLastUsed(t *testing.T) {
	st := newTestStore(t)
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

func TestAPIKeyStoreDeleteRemovesData(t *testing.T) {
	st := newTestStore(t)
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
	st := newTestStore(t)
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

func TestAPIKeyStoreDuplicateHashReturnsAlreadyExists(t *testing.T) {
	st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	hash := []byte{0xde, 0xad, 0xbe, 0xef}
	if err := ks.Create(ctx, seedAPIKey("a", hash, now, false)); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := ks.Create(ctx, seedAPIKey("b", hash, now, true))
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on duplicate key_hash, got: %v", err)
	}
}

func TestAPIKeyStoreIDsMonotonicAfterDelete(t *testing.T) {
	st := newTestStore(t)
	ks := st.APIKeys()
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	k1 := seedAPIKey("a", []byte{0x01}, now, false)
	k2 := seedAPIKey("b", []byte{0x02}, now, false)
	for _, k := range []*store.APIKey{k1, k2} {
		if err := ks.Create(ctx, k); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if err := ks.Delete(ctx, k2.ID); err != nil {
		t.Fatalf("delete k2: %v", err)
	}

	// AUTOINCREMENT must not reuse the freed id.
	k3 := seedAPIKey("c", []byte{0x03}, now, false)
	if err := ks.Create(ctx, k3); err != nil {
		t.Fatalf("create k3: %v", err)
	}
	if k3.ID <= k2.ID {
		t.Fatalf("expected new key ID > %d (monotonic), got %d", k2.ID, k3.ID)
	}
}
