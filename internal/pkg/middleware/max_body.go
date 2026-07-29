package middleware

import (
	"fmt"
	"net/http"
)

// NoBodyLimit disarms huma's own body bound on an operation, so that MaxBody
// below stays the ONLY authority on request size.
//
// # Why a second bound had to be disarmed rather than synchronised
//
// huma carries a body limit on each `huma.Operation`, NOT on its Config, and
// `huma.Register` silently defaults it to 1 MiB whenever the input has a Body.
// Every business route goes through huma, so that undocumented 1 MiB was the
// bound that actually answered — while `http.max_body_bytes` was read,
// validated at startup, and had no effect whatsoever (#141).
//
// An operator raising the key saw their configuration accepted, the service
// start, and requests still fail with `413 … limit=1048576 bytes` — a number
// written nowhere in their files. **A configuration key that exists, is
// documented and is validated, yet does nothing, is worse than a hard-coded
// value**: a hard-coded value can be found and argued with; this one made the
// question look settled.
//
// Two bounds for one thing is what `rules/references.md` forbids, and for this
// exact reason. Synchronising them would have kept both, each able to drift.
// This constant removes one, and `MaxBody` — configured, and tested on its own
// — keeps the job.
//
// Any strictly negative value works: huma only overrides a bound left at zero,
// and only enforces one that is strictly positive.
//
// ⚠️ Disarming it is safe ONLY because `MaxBody` is mounted on the whole mux.
// A route served outside that stack would have no bound at all. The guard
// `tools/verifie-borne-de-corps.sh` refuses an operation that omits this
// constant; it does not — and cannot — check that the middleware is mounted.
// That property is held by
// `httpserver/tests/the_middleware_stack_is_actually_mounted_test.go`.
//
// It also drops a defect of huma's check, which refuses a body of EXACTLY the
// limit: it compares the count copied through an `io.LimitReader` to the limit
// itself, so the last acceptable byte reads as one too many.
const NoBodyLimit int64 = -1

// MaxBody bounds the size of the body read. Without that bound, a client can
// grow the server's memory at will.
//
// Two mechanisms, and they answer two different questions.
//
// `http.MaxBytesReader` is the standard library's answer to *how much can be
// buffered*: it caps the read and signals the overflow to the response writer,
// so a handler that ignores the error still cannot be made to buffer without
// limit. It is the one that holds under a chunked body, where nobody knows the
// size in advance.
//
// The declared-length check answers *what does the client get told*. Without
// it, an oversized body reaches the handler, fails mid-read, and the framework
// reports a generic read error — huma answers **500 “cannot read request
// body”**. A `500` on a body the operator deliberately bounded is a lie about
// whose fault it is: it sends someone looking for a server defect when the
// request simply exceeded a documented limit.
//
// So the bound refuses early, with `413`, and it names the limit — the number
// the operator actually wrote in `http.max_body_bytes` (#141).
//
// ⚠️ `Content-Length` is client-supplied and can lie. It is used only to refuse
// SOONER, never to accept: a body that understates its length still meets
// `MaxBytesReader` on the way in. Trusting it to accept would be the hole;
// trusting it to reject costs the liar nothing but a correct answer.
func MaxBody(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 && r.ContentLength > limit {
				refuseTooLarge(w, limit)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// refuseTooLarge answers 413 in the same problem+json shape as the rest of the
// API, so a client does not have to special-case this one refusal.
//
// The limit is named. An operator reading a 413 must be able to tell the
// configured bound from a bound imposed elsewhere — that ambiguity is exactly
// what cost #141 its months of invisibility.
func refuseTooLarge(w http.ResponseWriter, limit int64) {
	w.Header().Set("content-type", "application/problem+json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = fmt.Fprintf(w,
		`{"title":"Request Entity Too Large","status":413,`+
			`"detail":"request body is too large limit=%d bytes"}`, limit)
}
