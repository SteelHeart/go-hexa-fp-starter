// Package tests contient les tests en BOÎTE NOIRE du paquet config : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// shippedCatalog assemble le catalogue EXACTEMENT comme le composition root.
//
// Charger la configuration livrée contre un catalogue inventé ne prouverait
// rien : c'est l'accord entre les fichiers de `config/` et les modules
// réellement embarqués qui est en jeu (ADR 014). Si ces deux-là divergent, la
// configuration livrée refuse de se charger — et c'est ce que ces tests
// doivent attraper, pas un binaire au premier démarrage.
func shippedCatalog(t *testing.T) config.ModuleCatalog {
	t.Helper()
	coreCatalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("catalogue du noyau: %v", err)
	}
	catalog, err := config.MergeCatalogs(coreCatalog, userregistration.Catalog())
	if err != nil {
		t.Fatalf("fusion des catalogues: %v", err)
	}
	return catalog
}

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

// withCatalogTestConfig prépare une configuration chargeable où seule la section
// `modules:` est celle du test.
//
// Elle recopie tous les groupes livrés SAUF `modules.yaml`, qu'elle remplace.
// Recopier plutôt que réécrire à la main est délibéré : une configuration
// minimale écrite ici divergerait de la vraie au premier réglage obligatoire
// ajouté, et le test échouerait pour une raison sans rapport avec ce qu'il
// vérifie.
func withCatalogTestConfig(t *testing.T, modules string) {
	t.Helper()

	dir := t.TempDir()
	// La couche d'environnement compte aussi : c'est `env/development.yaml` qui
	// donne un défaut explicite à `DB_DSN`. L'oublier ferait échouer le test sur
	// un secret manquant, c'est-à-dire pour une raison sans aucun rapport avec
	// ce qu'il vérifie.
	if err := os.MkdirAll(filepath.Join(dir, "env"), 0o750); err != nil {
		t.Fatalf("création de env/: %v", err)
	}
	var copies int
	for _, motif := range []string{"*.yaml", filepath.Join("env", "*.yaml")} {
		livres, err := filepath.Glob(filepath.Join(shippedConfigDir(), motif))
		if err != nil {
			t.Fatalf("lecture de la configuration livrée: %v", err)
		}
		for _, chemin := range livres {
			relatif, err := filepath.Rel(shippedConfigDir(), chemin)
			if err != nil {
				t.Fatalf("chemin relatif de %s: %v", chemin, err)
			}
			if filepath.Base(chemin) == "modules.yaml" || strings.HasPrefix(filepath.Base(chemin), "local") {
				continue
			}
			contenu, err := os.ReadFile(chemin)
			if err != nil {
				t.Fatalf("lecture de %s: %v", relatif, err)
			}
			if err := os.WriteFile(filepath.Join(dir, relatif), contenu, 0o600); err != nil {
				t.Fatalf("écriture de %s: %v", relatif, err)
			}
			copies++
		}
	}
	if copies == 0 {
		t.Fatal("aucun fichier de configuration copié : le test ne vérifierait rien")
	}
	if err := os.WriteFile(filepath.Join(dir, "modules.yaml"), []byte(modules), 0o600); err != nil {
		t.Fatalf("écriture de modules.yaml: %v", err)
	}

	t.Setenv(config.EnvVarConfigDir, dir)
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", testEncryptionKey())
}

// applicationCatalog est le catalogue qu'une APPLICATION fournirait.
//
// `facturation` n'existe nulle part dans le socle : ni code, ni pilote, ni
// ligne dans `internal/config`. Il n'a qu'un catalogue — et c'est tout ce que
// l'ADR 014 exige.
func applicationCatalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		"facturation": {
			Default: "memory",
			Drivers: map[string]config.Resources{
				"memory": {},
				"sqlite": {SQL: true},
			},
		},
	}
}
