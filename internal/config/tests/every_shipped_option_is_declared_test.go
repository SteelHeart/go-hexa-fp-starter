package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestEveryShippedOptionIsDeclared: the SHIPPED configuration passes the new
// check.
//
// Without it, this batch could be green having only added a guard that the
// repository itself does not clear — and the defect would only appear at
// somebody else's first `go run`.
//
// It is the positive counterpart of the failure witness: a guard must refuse
// what is wrong AND accept what is right. Both halves are necessary, and the
// experience of this repository is that it is the first one people forget to
// verify.
func TestEveryShippedOptionIsDeclared(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("the shipped configuration must load: %v", err)
	}

	// And the check must have something to check: if `modules.yaml` stopped
	// carrying the slightest option, this test would stay green while proving
	// nothing.
	options := 0
	for name := range cfg.Modules {
		options += len(cfg.Modules.Get(name).Options)
	}
	if options == 0 {
		t.Fatal("the shipped configuration carries no option — this test then proves nothing")
	}
	t.Logf("%d options declared in the shipped configuration", options)
}
