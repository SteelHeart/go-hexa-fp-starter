package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnUnknownDriverOptionIsRefused: an option key that the driver does not
// read refuses to start.
//
// # The defect this test would have caught
//
// Deny-by-default stopped at the name of the driver. Measured before the fix
// (#93): with `bath_size` instead of `batch_size`, the server STARTED, mounted
// the module and said nothing about it. The dispatcher ran with a default batch
// nobody had asked for.
//
// The cause is that the option accessors return the default value when the key
// is absent — correct taken in isolation — and that nothing enumerated the
// known keys. "Absent" and "misspelt" were therefore indistinguishable: exactly
// the shape of defect this repository meets over and over.
func TestAnUnknownDriverOptionIsRefused(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: memory
    options:
      bath_size: 50
`)

	_, err := config.Load(catalogueWithOptions())
	if err == nil {
		t.Fatal("an unknown option must refuse to start")
	}

	message := err.Error()
	if !strings.Contains(message, "bath_size") {
		t.Errorf("the message must name the offending key: %v", err)
	}
	// Refusing without saying what is admitted turns a guard into a dead end:
	// the person facing the error has to guess the exact spelling.
	if !strings.Contains(message, "batch_size") {
		t.Errorf("the message must list the admitted keys: %v", err)
	}
}

// catalogueWithOptions declares a module whose drivers admit DIFFERENT options.
//
// The difference is the subject: it is what makes it possible to say that an
// option valid for one driver is not valid for another.
func catalogueWithOptions() config.ModuleCatalog {
	return config.ModuleCatalog{
		"facturation": {
			Default: "memory",
			Drivers: map[string]config.Resources{
				"memory": {Options: []string{"batch_size", "interval"}},
				"sqlite": {SQL: true, Options: []string{"batch_size", "path"}},
				"nu":     {},
			},
		},
	}
}
