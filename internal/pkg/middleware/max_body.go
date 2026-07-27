package middleware

import "net/http"

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
