// Package middleware fournit les préoccupations transverses du transport HTTP.
//
// Ce sont les SEULS « middlewares » du dépôt : le mot est réservé au HTTP. Les
// préoccupations transverses du métier sont des décorateurs `func(P) P`
// (rules/references.md § vocabulaire imposé).
//
// Tous sont des `func(http.Handler) http.Handler`, donc composables avec
// n'importe quel routeur de l'écosystème net/http.
//
// # Un fichier par fonction publique
//
// Ce paquet applique au CODE la règle des tests (rules/tests.md §2) : chaque
// garde vit dans son propre fichier, nommé d'après elle. Ce n'est pas une
// préférence de rangement — le limiteur de débit de ce paquet n'a jamais rien
// limité, et le défaut a survécu parce qu'il était perdu au milieu d'un fichier
// que personne n'ouvrait pour lui.
package middleware

import "net/http"

// Middleware est une transformation d'un gestionnaire HTTP.
type Middleware = func(http.Handler) http.Handler

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
