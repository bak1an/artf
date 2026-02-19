package server

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/bak1an/artf/internal/ctxlog"
)

// recovers panics
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())

		defer func() {
			if v := recover(); v != nil {
				logger.Error(
					"http panic recovered",
					"panic", v,
					"method", r.Method,
					"path", r.URL.RequestURI(),
					"remote", r.RemoteAddr,
					"trace", string(debug.Stack()),
				)

				fmt.Fprintln(os.Stderr, string(debug.Stack()))

				http.Error(
					w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
