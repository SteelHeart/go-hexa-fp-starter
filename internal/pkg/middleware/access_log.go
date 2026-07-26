package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder capture le code de statut pour la journalisation.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err //nolint:wrapcheck // pass-through de l'écrivain sous-jacent
}

// AccessLog journalise une ligne par requête terminée.
// Aucune donnée personnelle : ni corps, ni paramètre de requête, ni en-tête
// d'autorisation (rules/securite.md §5).
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.InfoContext(r.Context(), "requête traitée",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Duration("duration", time.Since(started)),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
		})
	}
}
