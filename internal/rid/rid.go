package rid

// request id context helpers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func From(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKey{})
	s, ok := v.(string)
	return s, ok && s != ""
}

func MustFrom(ctx context.Context) string {
	if s, ok := From(ctx); ok {
		return s
	}
	return ""
}

// New generates a reasonably unique ID without external deps.
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// extremely unlikely; fallback to something non-empty
		return "rid-unknown"
	}
	return hex.EncodeToString(b[:]) // 32 hex chars
}
