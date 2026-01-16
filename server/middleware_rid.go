package server

import (
	"net/http"
	"strings"

	"github.com/bak1an/artf/server/rid"
)

const RequestIDHeader = "X-Request-Id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = rid.New()
		}

		// make it available everywhere downstream
		r = r.WithContext(rid.With(r.Context(), id))

		// echo back so clients can correlate
		w.Header().Set(RequestIDHeader, id)

		next.ServeHTTP(w, r)
	})
}
