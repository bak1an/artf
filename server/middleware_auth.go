package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bak1an/artf/internal/auth"
	"github.com/bak1an/artf/internal/ctxlog"
	"github.com/bak1an/artf/store"
)

// bearerToken returns the token from "Authorization: Bearer <token>" or "" if missing/invalid.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	s := r.Header.Get("Authorization")
	if s == "" || !strings.HasPrefix(s, prefix) {
		return ""
	}
	return strings.TrimSpace(s[len(prefix):])
}

func AuthMiddleware(keys store.APIKeyStore, readonly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := ctxlog.From(r.Context())

			rawKey := bearerToken(r)
			if rawKey == "" {
				logger.Warn("auth: missing api key")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			keyHash := auth.HashKey(rawKey)

			key, err := keys.GetByKey(r.Context(), keyHash)
			if err != nil {
				logger.Warn("auth: invalid api key", "error", err)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			if !readonly && key.ReadOnly {
				logger.Warn("auth: read-only key used on write endpoint")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)

			if key.LastUsedAt == nil || time.Since(*key.LastUsedAt) > time.Minute {
				go func() {
					if err := keys.UpdateLastUsed(context.Background(), key.ID, time.Now()); err != nil {
						logger.Warn("auth: failed to update last used", "error", err)
					}
				}()
			}
		})
	}
}
