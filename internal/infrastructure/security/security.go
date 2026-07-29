// Package security provides password hashing and encryption at rest.
//
// Nothing here knows the business: these are primitives, plugged onto ports by
// the composition root.
//
// # One file per public responsibility
//
// Three independent primitives, three files (rules/tests.md §2):
//
//	hasher.go       Argon2id — hash, verify, decide on rehashing
//	cipher.go       AES-256-GCM — encrypt and decrypt at rest
//	blind_index.go  deterministic index to search without decrypting
//
// The split is not cosmetic here: each of these three files carries at least one
// constant whose value IS a security guarantee — the AES key length, the bounds
// of the decoded digest. A guarantee drowned in the middle of two hundred lines
// about another subject reads badly, and gets modified without anyone measuring
// what is being relaxed.
package security
