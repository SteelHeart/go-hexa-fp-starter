package tests

import (
	"errors"
	"slices"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationRequiresExactlyOneSecret verrouille la frontière entre
// deux promesses qu'on confond facilement.
//
// « Zéro prérequis » porte sur l'INFRASTRUCTURE : ni base, ni cache, ni Docker.
// Elle ne porte pas sur les secrets. La clé de chiffrement n'a délibérément
// AUCUNE valeur de repli : une clé par défaut chiffrerait les données de tout le
// monde avec une clé publiquement connue — une faille, pas une commodité.
//
// Ce test fixe la liste exacte. Le jour où un second secret devient obligatoire,
// il échoue et force une décision explicite au lieu d'allonger en silence ce
// qu'un nouveau venu doit fournir pour démarrer.
func TestShippedConfigurationRequiresExactlyOneSecret(t *testing.T) {
	t.Setenv(config.EnvVarConfigDir, shippedConfigDir())
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", "")

	_, err := config.Load(shippedCatalog(t))

	var missing config.ErrMissingSecret
	if !errors.As(err, &missing) {
		t.Fatalf("attendu ErrMissingSecret, reçu %v", err)
	}
	want := []string{"SECURITY_ENCRYPTION_KEY"}
	if !slices.Equal(missing.Variables, want) {
		t.Errorf("secrets obligatoires = %v, attendu %v", missing.Variables, want)
	}
}
