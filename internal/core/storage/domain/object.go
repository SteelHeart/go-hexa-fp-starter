// Package domain carries the vocabulary of object storage, with no dependency.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// Key is the location of an object in the store.
//
// Distinct from the name asked for by the caller: the name comes from outside,
// the key is derived and safe. Confusing the two is the directory traversal
// vulnerability.
type Key string

// String returns the key in raw form.
func (k Key) String() string { return string(k) }

// Object is an object to be stored.
type Object struct {
	// Name is the name asked for by the caller. It comes from a form, hence
	// from a stranger: it is NEVER used as a path as it stands.
	Name string
	// ContentType is declared by the caller, hence never trustworthy for a
	// security decision. A driver that returns it in a header must restrict it
	// to an allow-list.
	ContentType string
	Content     io.Reader
}

// Located is the result of a successful store.
type Located struct {
	Key Key
	// URL is the read address. Its shape depends on the driver — a path served
	// by the application for `disk`, the provider's URL for an object store.
	URL string
}

// ErrUnsafeName refuses a name that would try to escape the store.
var ErrUnsafeName = errors.New("object name refused")

// ErrEmptyContent refuses an object without content.
var ErrEmptyContent = errors.New("object without content")

// IsValid says whether the object carries the usable minimum.
func (o Object) IsValid() bool { return o.Name != "" && o.Content != nil }

// SafeKey derives a safe storage key from the name asked for.
//
// # This is the security function of the module
//
// It is PURE and lives in the domain precisely for that: it can be tested
// exhaustively, without a disk, without a network, and every driver calls it. A
// driver that built its key itself would reopen the vulnerability for itself
// alone.
//
// Three properties, each for a reason:
//
//  1. **The name is hashed.** The key therefore contains no fragment of a path
//     supplied by the caller: `../../etc/passwd` cannot survive the hashing.
//  2. **The key is spread over two levels.** A flat directory with a hundred
//     thousand entries becomes impractical on most file systems.
//  3. **The base name is kept as a suffix**, cleaned. An object stays
//     identifiable by a human inspecting the store — without which an incident
//     is diagnosed blind.
func SafeKey(name string) (Key, error) {
	base := path.Base(path.Clean("/" + strings.ReplaceAll(name, "\\", "/")))
	if base == "." || base == ".." || base == "/" || base == "" {
		return "", fmt.Errorf("%w: %q designates no file", ErrUnsafeName, name)
	}

	sum := sha256.Sum256([]byte(name))
	digest := hex.EncodeToString(sum[:])
	return Key(fmt.Sprintf("%s/%s/%s-%s", digest[0:2], digest[2:4], digest[4:16], base)), nil
}

// IsWithin says whether a key stays within the bounds of the store.
//
// Called on READ, where the key comes from outside — from a URL, hence from a
// stranger — and has not been derived by SafeKey. Without this check, a `..` in
// a key would allow any file on the host to be read.
func IsWithin(key Key) bool {
	raw := strings.ReplaceAll(key.String(), "\\", "/")
	if raw == "" || strings.HasPrefix(raw, "/") {
		return false
	}
	cleaned := path.Clean(raw)
	return cleaned == raw && !strings.HasPrefix(cleaned, "../") && cleaned != ".."
}
