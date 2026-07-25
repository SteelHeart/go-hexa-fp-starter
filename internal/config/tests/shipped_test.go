package tests

import (
	"encoding/base64"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// shippedConfigDir pointe sur le répertoire config/ RÉELLEMENT livré.
//
// Les autres tests de ce paquet valident des structures construites à la main.
// Ceux-ci valident les fichiers du dépôt : sans eux, une faute de frappe dans
// config/modules.yaml passerait `task check` et n'apparaîtrait qu'au premier
// démarrage. C'est exactement le faux vert que rules/README.md interdit.
func shippedConfigDir() string { return filepath.Join("..", "..", "..", "config") }

// testEncryptionKey fabrique une clé AES-256 nulle, encodée à l'exécution.
//
// Aucune clé n'est écrite en dur dans le dépôt, même de test : rules/securite.md
// interdit tout secret versionné, et une chaîne base64 de 32 octets dans un
// fichier est indiscernable d'une vraie fuite pour gitleaks comme pour un lecteur.
func testEncryptionKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

// withShippedConfig pointe le chargeur sur la configuration livrée, secret fourni.
func withShippedConfig(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvVarConfigDir, shippedConfigDir())
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", testEncryptionKey())
}

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

	_, err := config.Load()

	var missing config.ErrMissingSecret
	if !errors.As(err, &missing) {
		t.Fatalf("attendu ErrMissingSecret, reçu %v", err)
	}
	want := []string{"SECURITY_ENCRYPTION_KEY"}
	if !slices.Equal(missing.Variables, want) {
		t.Errorf("secrets obligatoires = %v, attendu %v", missing.Variables, want)
	}
}

// TestShippedConfigurationLoads : la configuration livrée doit charger telle
// quelle, le seul secret obligatoire étant fourni.
func TestShippedConfigurationLoads(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("la configuration livrée doit charger: %v", err)
	}
	if len(cfg.Modules) == 0 {
		t.Error("aucun module lu : le fichier modules.yaml n'a pas été pris en compte")
	}
}

// TestShippedConfigurationNeedsNoInfrastructure : avec les pilotes livrés, aucun
// service externe n'est requis. Ce test échoue le jour où quelqu'un active un
// pilote postgres ou redis par défaut — et c'est précisément le but.
func TestShippedConfigurationNeedsNoInfrastructure(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	if cfg.Modules.RequiresSQL() {
		t.Error("la configuration livrée exige une base SQL : la promesse « zéro prérequis » est rompue")
	}
	if cfg.Modules.RequiresCache() {
		t.Error("la configuration livrée exige un cache : la promesse « zéro prérequis » est rompue")
	}
}

// TestShippedDriverOptionsParse : une option de pilote n'est validée qu'à la
// construction du module. Sans ce test, `ttl: 24` — un entier au lieu d'une durée,
// donc 24 secondes au lieu de 24 heures — resterait invisible jusqu'à ce qu'un
// rejeu légitime soit refusé en production.
func TestShippedDriverOptionsParse(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}

	ttl, err := cfg.Modules.Get("idempotency").DurationOption("ttl", 0)
	if err != nil {
		t.Fatalf("options du module idempotency illisibles: %v", err)
	}
	if ttl < time.Hour {
		t.Errorf("ttl livré = %v : une fenêtre de rejeu si courte trahit une unité oubliée", ttl)
	}
}
