package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// RequestIDHeader est l'en-tête de corrélation, accepté en entrée et toujours
// renvoyé en sortie.
const RequestIDHeader = "X-Request-Id"

// contextKey est un type privé : il rend toute collision de clé impossible.
type contextKey struct{ name string }

// requestIDKey est une globale assumée : le type `contextKey` est privé au paquet,
// donc aucun autre paquet ne peut fabriquer une clé égale, même en copiant le
// littéral. C'est l'idiome Go de la clé de contexte, et il n'a pas d'équivalent
// local.
//
//nolint:gochecknoglobals // clé de contexte : le type privé au niveau paquet EST le remède aux collisions
var requestIDKey = &contextKey{name: "request-id"}

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

// RequestIDFrom lit l'identifiant de corrélation. Il est toujours présent si
// RequestID est monté.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
