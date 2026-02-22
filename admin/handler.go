package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/bak1an/artf/internal/auth"
	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

// admin specific handlers

func listKeysHandler(keyStore store.APIKeyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())

		keys, err := keyStore.List(r.Context())
		if err != nil {
			logger.Error("failed to list keys", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := KeyListResponse{
			Keys: make([]*Key, len(keys)),
		}
		for i, key := range keys {
			resp.Keys[i] = &Key{
				ID:         key.ID,
				Name:       key.Name,
				ReadOnly:   key.ReadOnly,
				CreatedAt:  key.CreatedAt,
				LastUsedAt: key.LastUsedAt,
			}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func createKeyHandler(keyStore store.APIKeyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())

		var req KeyCreateRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logger.Error("failed to decode request", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		key := auth.GenerateKey()
		keyHash := auth.HashKey(key)

		now := time.Now()
		keyRecord := &store.APIKey{
			Name:       req.Name,
			ReadOnly:   req.ReadOnly,
			KeyHash:    keyHash,
			CreatedAt:  now,
			LastUsedAt: nil,
		}

		err = keyStore.Create(r.Context(), keyRecord)

		if err != nil {
			logger.Error("failed to create key", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(Key{
			ID:         keyRecord.ID,
			Name:       keyRecord.Name,
			ReadOnly:   keyRecord.ReadOnly,
			CreatedAt:  keyRecord.CreatedAt,
			LastUsedAt: keyRecord.LastUsedAt,
			Key:        &key,
		})

		if err != nil {
			logger.Error("failed to encode response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func deleteKeyHandler(keyStore store.APIKeyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())
		idPath := r.PathValue("id")
		id, err := strconv.ParseUint(idPath, 10, 64)
		if err != nil {
			logger.Error("failed to parse key ID", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = keyStore.Delete(r.Context(), id)
		if err != nil {
			logger.Error("failed to delete key", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
