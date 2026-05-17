package server

import (
	"context"
	"net/http"
	"time"
)

type ctxKey string

const accentCtxKey ctxKey = "accent"

const accentCookieName = "ipcalc_accent"

var allAccents = []string{"cyan", "amber", "green"}

const defaultAccent = "cyan"

// wrapMiddleware composes the global middleware chain. Outermost is recovery,
// then request log, then accent-cookie reader.
func (s *server) wrapMiddleware(h http.Handler) http.Handler {
	return s.recover(s.requestLog(s.withAccent(h)))
}

func (s *server) withAccent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), accentCtxKey, readAccentCookie(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.deps.Logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.deps.Logger.Error("panic", "path", r.URL.Path, "err", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(s int) {
	w.status = s
	w.ResponseWriter.WriteHeader(s)
}

func readAccentCookie(r *http.Request) string {
	c, err := r.Cookie(accentCookieName)
	if err != nil {
		return defaultAccent
	}
	if isValidAccent(c.Value) {
		return c.Value
	}
	return defaultAccent
}

func isValidAccent(v string) bool {
	for _, a := range allAccents {
		if v == a {
			return true
		}
	}
	return false
}

func accentFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(accentCtxKey).(string); ok {
		return v
	}
	return defaultAccent
}
