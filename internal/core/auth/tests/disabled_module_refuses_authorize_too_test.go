package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestDisabledModuleRefusesAuthorizeToo guards the worst default value
// imaginable.
//
// # The case one forgets
//
// A disabled module mounts anyway — that is what lets a surface exist and
// answer a clear error, rather than failing the whole startup for a module
// nobody enabled.
//
// The temptation is then to write the refusals "that matter" — registering,
// signing in — and to leave `Authorize` at its zero value. But the zero of a
// function port is `nil`, and a called `nil` panics; worse, a "neutral"
// implementation returning `nil` as its error would AUTHORISE everything. A
// turned-off authentication module that authorises everything is exactly the
// opposite of deny by default.
func TestDisabledModuleRefusesAuthorizeToo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, err := auth.New(config.Module{Enabled: false}, auth.Deps{})
	if err != nil {
		t.Fatalf("a disabled module must mount: %v", err)
	}

	if err := mod.Authorize(ctx, "anyone-at-all", permission(t, "billing.invoice.cancel")); !errors.Is(err, auth.ErrDisabled) {
		t.Fatalf("Authorize on a disabled module: want ErrDisabled, got %v", err)
	}

	if _, err := mod.Register(ctx, subject, secret); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Register: want ErrDisabled, got %v", err)
	}
	if _, err := mod.Authenticate(ctx, subject, secret); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Authenticate: want ErrDisabled, got %v", err)
	}
	if _, err := mod.Verify(ctx, domain.Token{}); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Verify: want ErrDisabled, got %v", err)
	}
	if err := mod.Revoke(ctx, domain.Token{}); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Revoke: want ErrDisabled, got %v", err)
	}
	if err := mod.DefineRole(ctx, "accountant", nil); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("DefineRole: want ErrDisabled, got %v", err)
	}
	if err := mod.AssignRoles(ctx, "anyone-at-all", nil); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("AssignRoles: want ErrDisabled, got %v", err)
	}
}
