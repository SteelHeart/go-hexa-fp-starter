// Package middleware fournit les préoccupations transverses du transport HTTP.
//
// Ce sont les SEULS « middlewares » du dépôt : le mot est réservé au HTTP. Les
// préoccupations transverses du métier sont des décorateurs `func(P) P`
// (rules/references.md § vocabulaire imposé).
//
// Tous sont des `func(http.Handler) http.Handler`, donc composables avec
// n'importe quel routeur de l'écosystème net/http.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// Middleware est une transformation d'un gestionnaire HTTP.
type Middleware = func(http.Handler) http.Handler

// contextKey est un type privé : il rend toute collision de clé impossible.
type contextKey struct{ name string }

var requestIDKey = &contextKey{name: "request-id"}

// RequestIDHeader est l'en-tête de corrélation, accepté en entrée et toujours
// renvoyé en sortie.
const RequestIDHeader = "X-Request-Id"

// Chain compose des middlewares. Le premier listé est le plus externe : il voit
// la requête en premier et la réponse en dernier.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// RequestIDFrom lit l'identifiant de corrélation. Il est toujours présent si
// RequestID est monté.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID propage un identifiant de corrélation, en réutilisant celui fourni
// par l'appelant s'il est plausible.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			// Un en-tête entrant n'est pas de confiance : il finit dans les logs.
			if id == "" || len(id) > 64 || strings.ContainsAny(id, "\r\n") {
				id = uuid.NewString()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
		})
	}
}

// Recover intercepte une panique, la journalise et répond 500 sans divulguer la
// pile à l'appelant.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				// http.ErrAbortHandler est un signal volontaire, pas un défaut.
				if recovered == http.ErrAbortHandler { //nolint:errorlint // sentinelle levée par panic, pas enveloppée
					panic(recovered)
				}
				logger.ErrorContext(r.Context(), "panique dans un gestionnaire HTTP",
					slog.Any("recovered", recovered),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("request_id", RequestIDFrom(r.Context())),
				)
				http.Error(w, `{"title":"Internal Server Error","status":500}`, http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders pose les en-têtes de durcissement sur toute réponse.
// `secure` active les protections qui exigent HTTPS.
func SecurityHeaders(secure bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
			// API JSON : aucune ressource active n'est servie.
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			if secure {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS n'autorise que les origines explicitement listées.
// Une liste vide refuse tout — deny par défaut.
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowedOrigins, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)
				h.Set("Access-Control-Max-Age", "600")
				h.Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBody borne la taille du corps lu. Sans cette borne, un client peut faire
// grossir la mémoire du serveur à volonté.
func MaxBody(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter limite le débit par client.
//
// ⚠️ En mémoire, donc PAR INSTANCE : derrière N répliques la limite effective
// est multipliée par N (voir SECURITY.md § ce que ce socle ne fournit pas).
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
	ttl      time.Duration
}

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

// NewRateLimiter construit un limiteur. `ttl` est la durée au bout de laquelle
// un client inactif est oublié, pour que la table ne croisse pas sans fin.
func NewRateLimiter(rps float64, burst int, ttl time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
		ttl:      ttl,
	}
}

// Middleware retourne le middleware de limitation.
func (l *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientKey(r)) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"title":"Too Many Requests","status":429}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, found := l.visitors[key]
	if !found {
		v = &visitor{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.visitors[key] = v
		l.evictLocked(now)
	}
	v.seen = now
	return v.limiter.Allow()
}

// evictLocked purge les clients inactifs. Appelée sous verrou, et seulement à
// l'apparition d'un nouveau client : pas de goroutine de nettoyage à arrêter.
func (l *RateLimiter) evictLocked(now time.Time) {
	for key, v := range l.visitors {
		if now.Sub(v.seen) > l.ttl {
			delete(l.visitors, key)
		}
	}
}

// clientKey identifie l'appelant. RemoteAddr uniquement : un en-tête
// X-Forwarded-For non validé serait falsifiable, donc la limite serait
// contournable. Le mandataire de confiance devra le réécrire lui-même.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

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
