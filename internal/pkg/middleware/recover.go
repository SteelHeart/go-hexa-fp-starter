package middleware

import (
	"log/slog"
	"net/http"
)

// Recover catches a panic, logs it, and answers 500 without disclosing the
// stack to the caller.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The context is captured BEFORE the defer, never read inside it: a
			// handler that panics may have replaced `r`, and the panic would then be
			// logged with a context that is not the request's — hence without its
			// correlation identifier, exactly when it matters.
			ctx := r.Context()
			method, path := r.Method, r.URL.Path

			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				// http.ErrAbortHandler is a deliberate signal, not a defect.
				if recovered == http.ErrAbortHandler { //nolint:errorlint // sentinel raised by panic, not wrapped
					panic(recovered)
				}
				logger.ErrorContext(ctx, "panic in an HTTP handler",
					slog.Any("recovered", recovered),
					slog.String("method", method),
					slog.String("path", path),
					slog.String("request_id", RequestIDFrom(ctx)),
				)
				http.Error(w, `{"title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
