// Package pagination provides cursor pagination, the contract shared by every
// surface.
//
// Why not OFFSET: past a few thousand rows Postgres has to walk then discard
// everything before, and — worse — a concurrent insert shifts the pages, so the
// caller silently skips rows. A cursor names a stable position, not a rank.
package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Page size bounds. An unbounded page is a denial of service, offered.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// ErrInvalidCursor reports an unreadable or tampered cursor.
var ErrInvalidCursor = errors.New("invalid cursor")

// Cursor names a stable position in a (CreatedAt, ID) ordering.
//
// The timestamp + identifier pair is required: the timestamp alone is not
// unique, and two rows created within the same microsecond would make
// pagination loop forever.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// Encode serialises the cursor for public transport.
//
// The encoding is reversible and UNSIGNED: a cursor is not a secret and must
// never carry authorisation information. The query re-checks rights, always.
func (c Cursor) Encode() string {
	raw := strconv.FormatInt(c.CreatedAt.UTC().UnixMicro(), 10) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reads an encoded cursor.
func DecodeCursor(encoded string) (Cursor, error) {
	if encoded == "" {
		return Cursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: base64: %w", ErrInvalidCursor, err)
	}
	micros, id, found := strings.Cut(string(raw), "|")
	if !found || id == "" {
		return Cursor{}, fmt.Errorf("%w: format", ErrInvalidCursor)
	}
	parsed, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: timestamp: %w", ErrInvalidCursor, err)
	}
	return Cursor{CreatedAt: time.UnixMicro(parsed).UTC(), ID: id}, nil
}

// Request carries a page request.
type Request struct {
	After Cursor
	Limit int
	// HasAfter tells "first page" apart from "zero cursor".
	HasAfter bool
}

// NewRequest builds a page request, bounding the size.
// An empty cursor means "first page"; that is not an error.
func NewRequest(encodedCursor string, limit int) (Request, error) {
	req := Request{Limit: clampLimit(limit)}
	if encodedCursor == "" {
		return req, nil
	}
	cursor, err := DecodeCursor(encodedCursor)
	if err != nil {
		return Request{}, err
	}
	req.After = cursor
	req.HasAfter = true
	return req, nil
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

// FetchLimit is the size to ask the database for: one item more than needed, so
// we know whether a next page exists without running a COUNT.
func (r Request) FetchLimit() int { return r.Limit + 1 }

// Page carries a slice of results and what is needed to ask for the next one.
type Page[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}

// NewPage builds a page from the rows actually read (FetchLimit items at most)
// and the function extracting the cursor from an item.
func NewPage[T any](fetched []T, req Request, cursorOf func(T) Cursor) Page[T] {
	hasMore := len(fetched) > req.Limit
	items := fetched
	if hasMore {
		items = fetched[:req.Limit]
	}
	page := Page[T]{Items: items, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		page.NextCursor = cursorOf(items[len(items)-1]).Encode()
	}
	return page
}
