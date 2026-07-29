package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnOptionOfAnotherDriverIsRefused: an option that is correctly spelt, but
// belongs to ANOTHER driver, is refused too.
//
// This is the costliest case, and the least obvious. The key exists, it is
// documented, it reads back without suspicion — but the driver retained never
// reads it. The setting has no effect, and nothing says so.
//
// The real case is in `dynconf`: `flags` and `settings` carry the VERSIONED
// values, so they only make sense for the file driver. Under the postgres
// driver the values live in the database; writing them in the repository would
// describe an intention that nothing executes.
//
// That is what the PER DRIVER declaration makes it possible to express, where a
// per-module list could not.
func TestAnOptionOfAnotherDriverIsRefused(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: memory
    options:
      path: /tmp/facturation.db
`)

	_, err := config.Load(catalogueWithOptions())
	if err == nil {
		t.Fatal("an option of another driver must refuse to start")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Errorf("the message must name the offending key: %v", err)
	}
	if !strings.Contains(err.Error(), `"memory"`) {
		t.Errorf("the message must name the driver that does not read it: %v", err)
	}
}

// TestADriverWithoutOptionsRefusesEveryOne: a driver that declares no option
// accepts none, and says so in those terms.
//
// The zero value of `Resources` admits nothing, and that is the right default:
// a driver that forgot to declare its options would see its configuration
// refused — which gets noticed. The opposite would reopen the hole for every
// driver at once, silently.
func TestADriverWithoutOptionsRefusesEveryOne(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: nu
    options:
      batch_size: 50
`)

	_, err := config.Load(catalogueWithOptions())
	if err == nil {
		t.Fatal("a driver without a declared option must refuse everything")
	}
	// "expected: " followed by nothing would read like a truncated message, and
	// would send people looking for a defect in the tool rather than for the
	// configuration mistake.
	if !strings.Contains(err.Error(), "accepts no option") {
		t.Errorf("the message must say that this driver has no option: %v", err)
	}
}
