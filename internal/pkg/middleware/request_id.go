package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// RequestIDHeader is the correlation header, accepted on input and always
// echoed on output.
const RequestIDHeader = "X-Request-Id"

// contextKey is a private type: it makes any key collision impossible.
type contextKey struct{ name string }

// requestIDKey is an assumed global: the `contextKey` type is private to the
// package, so no other package can build an equal key, even by copying the
// literal. This is the Go idiom for a context key, and it has no local
// equivalent.
//
//nolint:gochecknoglobals // context key: the package-private type IS the cure for collisions
var requestIDKey = &contextKey{name: "request-id"}

// RequestID propagates a correlation identifier, reusing the caller's one when
// it is plausible.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			// An incoming header is untrusted: it ends up in the logs.
			if id == "" || len(id) > 64 || strings.ContainsAny(id, "\r\n") {
				id = uuid.NewString()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
		})
	}
}

// RequestIDFrom reads the correlation identifier. It is always present when
// RequestID is mounted.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
