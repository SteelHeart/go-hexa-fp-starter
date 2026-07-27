package middleware

import "net/http"

// hstsOneYear exige HTTPS pendant un an, sous-domaines compris.
//
// Un an est la valeur qui rend un domaine éligible au préchargement des
// navigateurs. La durée est volontairement longue : ce qu'elle protège, c'est la
// PREMIÈRE requête d'une visite ultérieure — celle qu'un attaquant présent sur le
// réseau détournerait avant tout échange chiffré.
const hstsOneYear = "max-age=31536000; includeSubDomains"

// SecurityHeaders pose les en-têtes de durcissement, HSTS compris.
//
// C'est le constructeur par DÉFAUT : obtenir la protection ne coûte rien, y
// renoncer doit se nommer.
func SecurityHeaders() Middleware {
	return hardeningHeaders(hstsOneYear)
}

// SecurityHeadersWithoutHSTS pose les mêmes en-têtes SANS Strict-Transport-Security.
//
// Réservé au développement en clair : sur `http://localhost`, HSTS inscrirait dans
// le navigateur une exigence de HTTPS que le poste ne peut pas satisfaire, et le
// développeur perdrait l'accès à son propre serveur jusqu'à purger le cache.
//
// Le nom porte la renonciation. C'était autrefois `SecurityHeaders(false)`, où le
// booléen ne disait ni ce qu'il désactivait, ni ce que ça coûtait.
func SecurityHeadersWithoutHSTS() Middleware {
	return hardeningHeaders("")
}

// hardeningHeaders reçoit la VALEUR de l'en-tête, pas un drapeau : vide = absent.
func hardeningHeaders(hsts string) Middleware {
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
			if hsts != "" {
				h.Set("Strict-Transport-Security", hsts)
			}
			next.ServeHTTP(w, r)
		})
	}
}
