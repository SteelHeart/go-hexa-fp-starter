// Package tests contient les tests en BOÎTE NOIRE du domaine de l'inscription :
// ils n'utilisent que l'API publique, exactement comme le ferait un cas d'usage.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
//
// # Pourquoi ces tests sont les plus rentables du dépôt
//
// Le domaine est PUR : ni I/O, ni horloge, ni aléa. Il se teste donc en
// microsecondes, sans conteneur, sans double, sans montage. C'est là que chaque
// règle métier et chaque cas limite doivent être couverts — pas plus haut, où
// chaque test coûte cent fois plus cher pour prouver la même chose.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// emailValide construit une adresse dont on sait qu'elle passe, ou fait échouer
// le test. Évite de répéter la discrimination du Result dans chaque fichier.
func emailValide(t *testing.T, raw string) domain.Email {
	t.Helper()
	value, err, ok := domain.NewEmail(raw).Get()
	if !ok {
		t.Fatalf("NewEmail(%q) devait réussir, refusée: %v", raw, err)
	}
	return value
}

// codeDe extrait le code d'erreur d'un Result en échec.
func codeDe[T any](t *testing.T, r result.Result[T, domain.Error]) domain.ErrorCode {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("un échec était attendu, reçu un succès")
	}
	return err.Code
}

// erreurDe extrait l'erreur complète d'un Result en échec.
func erreurDe[T any](t *testing.T, r result.Result[T, domain.Error]) domain.Error {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("un échec était attendu, reçu un succès")
	}
	return err
}
