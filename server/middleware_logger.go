package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bak1an/artf/server/ctxlog"
	"github.com/bak1an/artf/server/rid"
)

// quickly intercept response status code.
// not implementing all the possible interface on it, not required now
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Logs requests duration and metadata
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.From(r.Context())
		start := time.Now()

		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}

		logger.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"duration", time.Since(start),
			"status", status,
		)
	})
}

func RequestContextLogger(base *slog.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid, _ := rid.From(r.Context())
			l := base
			if rid != "" {
				l = base.With("request_id", rid)
			}

			r = r.WithContext(ctxlog.WithLogger(r.Context(), l))
			next.ServeHTTP(w, r)
		})
	}
}
