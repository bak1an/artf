package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/mock"
)

func TestListKeysHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		keys := []*store.APIKey{
			{ID: 1, Name: "ci", ReadOnly: false, CreatedAt: now},
			{ID: 2, Name: "deploy", ReadOnly: true, CreatedAt: now},
		}

		ks := &mock.MockAPIKeyStore{
			ListFn: func(_ context.Context) ([]*store.APIKey, error) {
				return keys, nil
			},
		}

		h := listKeysHandler(ks)

		req := httptest.NewRequest(http.MethodGet, "/keys", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp KeyListResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(resp.Keys) != len(keys) {
			t.Fatalf("expected %d keys, got %d", len(keys), len(resp.Keys))
		}

		for i, k := range resp.Keys {
			if k.ID != keys[i].ID {
				t.Errorf("key[%d].ID: want %d, got %d", i, keys[i].ID, k.ID)
			}
			if k.Name != keys[i].Name {
				t.Errorf("key[%d].Name: want %q, got %q", i, keys[i].Name, k.Name)
			}
			if k.ReadOnly != keys[i].ReadOnly {
				t.Errorf("key[%d].ReadOnly: want %v, got %v", i, keys[i].ReadOnly, k.ReadOnly)
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{
			ListFn: func(_ context.Context) ([]*store.APIKey, error) {
				return nil, errors.New("database error")
			},
		}

		h := listKeysHandler(ks)

		req := httptest.NewRequest(http.MethodGet, "/keys", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestCreateKeyHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var createdKey *store.APIKey
		ks := &mock.MockAPIKeyStore{
			CreateFn: func(_ context.Context, key *store.APIKey) error {
				createdKey = key
				key.ID = 42
				return nil
			},
		}

		h := createKeyHandler(ks)

		body, _ := json.Marshal(KeyCreateRequest{Name: "test-key", ReadOnly: true})
		req := httptest.NewRequest(http.MethodPost, "/keys", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		var resp Key
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if resp.ID != 42 {
			t.Errorf("ID: want 42, got %d", resp.ID)
		}
		if resp.Name != "test-key" {
			t.Errorf("Name: want %q, got %q", "test-key", resp.Name)
		}
		if !resp.ReadOnly {
			t.Errorf("ReadOnly: want true, got false")
		}
		if resp.Key == nil || !strings.HasPrefix(*resp.Key, "artf_") {
			t.Errorf("Key: expected non-nil key with artf_ prefix, got %v", resp.Key)
		}
		if createdKey == nil {
			t.Fatal("expected key to be created in store")
		}
		if !bytes.Equal(createdKey.KeyHash, createdKey.KeyHash) {
			t.Error("KeyHash should be set")
		}
	})

	t.Run("bad request", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{}
		h := createKeyHandler(ks)

		req := httptest.NewRequest(http.MethodPost, "/keys", bytes.NewReader([]byte("invalid json")))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{
			CreateFn: func(_ context.Context, _ *store.APIKey) error {
				return errors.New("database error")
			},
		}

		h := createKeyHandler(ks)

		body, _ := json.Marshal(KeyCreateRequest{Name: "test-key", ReadOnly: false})
		req := httptest.NewRequest(http.MethodPost, "/keys", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestDeleteKeyHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var deletedID uint64
		ks := &mock.MockAPIKeyStore{
			DeleteFn: func(_ context.Context, id uint64) error {
				deletedID = id
				return nil
			},
		}

		h := deleteKeyHandler(ks)

		req := httptest.NewRequest(http.MethodDelete, "/keys/42", nil)
		req.SetPathValue("id", "42")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if deletedID != 42 {
			t.Errorf("deleted ID: want 42, got %d", deletedID)
		}
	})

	t.Run("bad request", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{}
		h := deleteKeyHandler(ks)

		req := httptest.NewRequest(http.MethodDelete, "/keys/invalid", nil)
		req.SetPathValue("id", "invalid")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{
			DeleteFn: func(_ context.Context, _ uint64) error {
				return store.ErrNotFound
			},
		}

		h := deleteKeyHandler(ks)

		req := httptest.NewRequest(http.MethodDelete, "/keys/999", nil)
		req.SetPathValue("id", "999")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		ks := &mock.MockAPIKeyStore{
			DeleteFn: func(_ context.Context, _ uint64) error {
				return errors.New("database error")
			},
		}

		h := deleteKeyHandler(ks)

		req := httptest.NewRequest(http.MethodDelete, "/keys/42", nil)
		req.SetPathValue("id", "42")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}
