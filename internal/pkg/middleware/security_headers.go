package middleware

import "net/http"

// hstsOneYear requires HTTPS for a year, subdomains included.
//
// One year is the value that makes a domain eligible for browser preloading. The
// duration is deliberately long: what it protects is the FIRST request of a later
// visit — the one an attacker sitting on the network would hijack before any
// encrypted exchange.
const hstsOneYear = "max-age=31536000; includeSubDomains"

// SecurityHeaders sets the hardening headers, HSTS included.
//
// This is the DEFAULT constructor: getting the protection costs nothing, opting
// out has to be named.
func SecurityHeaders() Middleware {
	return hardeningHeaders(hstsOneYear)
}

// SecurityHeadersWithoutHSTS sets the same headers WITHOUT
// Strict-Transport-Security.
//
// Reserved for cleartext development: on `http://localhost`, HSTS would record in
// the browser an HTTPS requirement the machine cannot satisfy, and the developer
// would lose access to their own server until clearing the cache.
//
// The name carries the opt-out. It used to be `SecurityHeaders(false)`, where the
// boolean said neither what it disabled nor what that cost.
func SecurityHeadersWithoutHSTS() Middleware {
	return hardeningHeaders("")
}

// hardeningHeaders receives the header VALUE, not a flag: empty means absent.
func hardeningHeaders(hsts string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
			// JSON API: no active resource is served.
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			if hsts != "" {
				h.Set("Strict-Transport-Security", hsts)
			}
			next.ServeHTTP(w, r)
		})
	}
}
