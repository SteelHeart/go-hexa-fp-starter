package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestTraversalNamesAreRefused est le test le plus important du module.
//
// Le nom d'un objet vient d'un formulaire, donc d'un inconnu. Utilisé tel quel
// comme chemin, il permet d'écrire n'importe où sur l'hôte — c'est la faille de
// traversée de répertoire, et elle a suffi à compromettre des serveurs entiers.
//
// Deux protections se cumulent, et le test vérifie les deux :
//   - les noms qui ne désignent aucun fichier sont REFUSÉS ;
//   - les autres sont HACHÉS, donc aucun fragment de chemin ne survit.
func TestTraversalNamesAreRefused(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"remontée simple":    "..",
		"remontée profonde":  "../../..",
		"racine":             "/",
		"répertoire courant": ".",
		"nom vide":           "",
		"remontée Windows":   "..\\..",
		"séparateurs seuls":  "///",
		"remontée déguisée":  "foo/../..",
		"racine puis parent": "/..",
	}

	for name, candidate := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.SafeKey(candidate); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("SafeKey(%q) = %v, attendu ErrUnsafeName", candidate, err)
			}
		})
	}
}

// TestTraversalNamesAreNeutralised : les noms qui contiennent un chemin mais
// désignent bien un fichier sont acceptés, et le chemin ne survit PAS.
//
// C'est la deuxième moitié de la protection : `../../etc/passwd` ne doit pas être
// refusé pour la forme, il doit être rendu inoffensif.
func TestTraversalNamesAreNeutralised(t *testing.T) {
	t.Parallel()

	dangerous := []string{
		"../../etc/passwd",
		"/etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"foo/../../../bar.txt",
		"./../secret.pem",
	}

	for _, candidate := range dangerous {
		key, err := domain.SafeKey(candidate)
		if err != nil {
			continue // refusé, c'est déjà sûr
		}
		if !domain.IsWithin(key) {
			t.Errorf("SafeKey(%q) = %q sort du magasin", candidate, key)
		}
		if got := key.String(); got == candidate {
			t.Errorf("SafeKey(%q) a rendu le nom tel quel", candidate)
		}
	}
}
