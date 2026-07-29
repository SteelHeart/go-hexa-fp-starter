package tests

import (
	"errors"
	"slices"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationRequiresExactlyOneSecret locks down the boundary
// between two promises that are easily confused.
//
// "Zero prerequisite" is about INFRASTRUCTURE: no database, no cache, no
// Docker. It is not about secrets. The encryption key deliberately has NO
// fallback value: a default key would encrypt everybody's data with a publicly
// known key — a vulnerability, not a convenience.
//
// This test fixes the exact list. The day a second secret becomes mandatory, it
// fails and forces an explicit decision instead of silently lengthening what a
// newcomer must supply in order to start.
func TestShippedConfigurationRequiresExactlyOneSecret(t *testing.T) {
	t.Setenv(config.EnvVarConfigDir, shippedConfigDir())
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", "")

	_, err := config.Load(shippedCatalog(t))

	var missing config.ErrMissingSecret
	if !errors.As(err, &missing) {
		t.Fatalf("want ErrMissingSecret, got %v", err)
	}
	want := []string{"SECURITY_ENCRYPTION_KEY"}
	if !slices.Equal(missing.Variables, want) {
		t.Errorf("mandatory secrets = %v, want %v", missing.Variables, want)
	}
}
