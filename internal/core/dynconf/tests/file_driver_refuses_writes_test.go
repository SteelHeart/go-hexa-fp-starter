package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestFileDriverRefusesWrites: writing into versioned values at run time would
// create a divergence between the repository and what is running, which the
// next deployment would overwrite without warning. The refusal is explicit,
// never silent.
func TestFileDriverRefusesWrites(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, nil)
	ctx := context.Background()

	valid := domain.Change{Kind: domain.KindFlag, Key: "mode", Value: "true"}
	if err := mod.Set(ctx, valid); !errors.Is(err, domain.ErrReadOnly) {
		t.Errorf("Set = %v, want ErrReadOnly", err)
	}
}

// TestFileDriverReportsMalformedChangeFirst: a malformed change is reported
// BEFORE the refusal to write.
//
// The order matters: the caller who is going to move to the `postgres` driver
// in order to write must first learn that their change was wrong. Otherwise
// they would change driver and discover the error only there.
func TestFileDriverReportsMalformedChangeFirst(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, nil)
	ctx := context.Background()

	cases := map[string]domain.Change{
		"without key":    {Kind: domain.KindFlag},
		"unknown nature": {Kind: "other", Key: "mode"},
		"nature missing": {Key: "mode"},
	}

	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := mod.Set(ctx, change); !errors.Is(err, domain.ErrInvalidChange) {
				t.Errorf("Set = %v, want ErrInvalidChange", err)
			}
		})
	}
}
