package middleware

import (
	"log/slog"
	"net/http"
)

// Recover intercepte une panique, la journalise et répond 500 sans divulguer la
// pile à l'appelant.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Le contexte est capturé AVANT le defer, jamais lu dedans : un
			// gestionnaire qui panique peut avoir remplacé `r`, et la panique serait
			// alors journalisée avec un contexte qui n'est pas celui de la requête —
			// donc sans son identifiant de corrélation, précisément quand il sert.
			ctx := r.Context()
			method, path := r.Method, r.URL.Path

			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				// http.ErrAbortHandler est un signal volontaire, pas un défaut.
				if recovered == http.ErrAbortHandler { //nolint:errorlint // sentinelle levée par panic, pas enveloppée
					panic(recovered)
				}
				logger.ErrorContext(ctx, "panique dans un gestionnaire HTTP",
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
