// Package middleware provides the cross-cutting concerns of the HTTP transport.
//
// These are the ONLY "middlewares" in the repository: the word is reserved for
// HTTP. Cross-cutting business concerns are decorators, `func(P) P`
// (rules/references.md, imposed vocabulary).
//
// All of them are `func(http.Handler) http.Handler`, hence composable with any
// router of the net/http ecosystem.
//
// # One file per public function
//
// This package applies the test rule (rules/tests.md §2) to CODE: every guard
// lives in its own file, named after it. This is not a filing preference — the
// rate limiter in this package never limited anything, and the defect survived
// because it was lost in the middle of a file nobody opened for it.
//
//	middleware.go        the Middleware type and Chain
//	access_log.go        one log line per completed request
//	cors.go              allowed origins, deny by default
//	max_body.go          bounds the size of the body read
//	rate_limiter.go      per-client throttling, in memory
//	recover.go           turns a panic into a 500 without leaking the stack
//	request_id.go        correlation identifier, never trusted on input
//	security_headers.go  hardening headers, with and without HSTS
package middleware

import "net/http"

// Middleware is a transformation of an HTTP handler.
type Middleware = func(http.Handler) http.Handler

// Chain composes middlewares. The first one listed is the outermost: it sees the
// request first and the response last.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
