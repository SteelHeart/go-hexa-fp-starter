package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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
		// La purge a lieu AVANT l'insertion, et le nouveau visiteur naît
		// horodaté. L'ordre inverse — insérer, puis purger — supprimait le
		// visiteur qu'on venait de créer : son `seen` valait encore la date
		// zéro, et `now.Sub(zéro)` dépasse n'importe quel TTL.
		//
		// Conséquence, restée invisible jusqu'à ce qu'un test la cherche : la
		// table restait vide, chaque requête repartait avec un limiteur neuf,
		// et la limitation de débit ne limitait RIEN. Aucun symptôme — un
		// limiteur qui laisse tout passer se comporte exactement comme un
		// service peu sollicité.
		l.evictLocked(now)
		v = &visitor{limiter: rate.NewLimiter(l.rps, l.burst), seen: now}
		l.visitors[key] = v
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
