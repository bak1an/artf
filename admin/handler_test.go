package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bak1an/artf/store"
	"github.com/bak1an/artf/store/mock"
)

func TestListKeysHandler(t *testing.T) {
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
}
