// Package tests contient les tests en BOÎTE NOIRE du paquet config : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"encoding/base64"
	"path/filepath"
	"testing"

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
