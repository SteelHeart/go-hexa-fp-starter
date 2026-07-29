// Package tests holds the BLACK BOX tests of cursor pagination: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import (
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// row is a paginable test entity.
type row struct {
	ID        string
	CreatedAt time.Time
}

// base is the reference instant.
func base() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// cursorOf extracts the cursor of a row.
func cursorOf(r row) pagination.Cursor {
	return pagination.Cursor{CreatedAt: r.CreatedAt, ID: r.ID}
}

// rows builds n rows one second apart, in order.
func rows(n int) []row {
	out := make([]row, 0, n)
	for i := range n {
		out = append(out, row{
			ID:        string(rune('a' + i)),
			CreatedAt: base().Add(time.Duration(i) * time.Second),
		})
	}
	return out
}
