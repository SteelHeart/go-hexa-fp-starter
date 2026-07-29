package tests

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestPostgresDriverRefusesWithoutLogger: this driver can NOT return its
// outages — the contract of ports.IsEnabled forbids it. Without a logger, an
// unreachable database would switch the hidden features off without leaving a
// trace. The logger is therefore not a comfort, it is the counterpart of the
// contract.
//
// The pool passed in is an empty shell and is never used: the factory only
// tests its presence, and verifying a refusal does not require a database.
func TestPostgresDriverRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(
		config.Module{Enabled: true, Driver: "postgres"},
		dynconf.Deps{Pool: &pgxpool.Pool{}},
	)
	if !errors.Is(err, dynconf.ErrLoggerRequired) {
		t.Errorf("error = %v, want ErrLoggerRequired", err)
	}
}
