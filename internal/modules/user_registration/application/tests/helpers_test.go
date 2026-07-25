// Package tests contient les tests en BOÎTE NOIRE des cas d'usage de
// l'inscription : ils n'utilisent que l'API publique, exactement comme le ferait
// un adaptateur primaire.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
//
// # Aucune bibliothèque de mock, et ce n'est pas une privation
//
// Chaque port est un type FONCTION : un double de test est une closure de trois
// lignes. Pas de génération de code, pas d'assertions d'appel magiques, pas de
// `.On("Method").Return(...)` qui casse au premier changement de signature. Le
// compilateur vérifie les doubles comme il vérifie le reste
// (rules/dependances.md).
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Constantes de scénario, pour que les tests se lisent comme des phrases.
const (
	adresseValide  = "alice@example.com"
	motDePasseFort = "correct cheval batterie agrafe"
	condense       = "$argon2id$v=19$m=65536,t=3,p=2$c2Vs$Y29uZGVuc2U"
	identifiant    = domain.UserID("user-42")
)

// instantFixe est l'horloge des tests : aucun test ne lit l'heure réelle.
func instantFixe() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// journal enregistre ce que le pipeline a réellement appelé, et dans quel ordre.
//
// L'ordre compte autant que le contenu : hacher avant de vérifier la disponibilité
// de l'adresse serait correct fonctionnellement et coûteux en pratique.
type journal struct {
	appels     []string
	sauvegarde domain.User
	evenement  any
	typeEvent  string
	agregat    string
	hache      string
}

func (j *journal) note(nom string) { j.appels = append(j.appels, nom) }

func (j *journal) aAppele(nom string) bool {
	for _, appel := range j.appels {
		if appel == nom {
			return true
		}
	}
	return false
}

// commande forge une intention d'inscription valide.
func commande() domain.RegistrationCommand {
	return domain.RegistrationCommand{Email: adresseValide, Password: motDePasseFort}
}

// depsNominales construit un jeu de ports où tout réussit, et qui note ses appels.
//
// Chaque test part de là et ne remplace QUE le port qui l'intéresse : ce qui
// change dans le test est exactement ce que le test vérifie.
func depsNominales(j *journal) application.Deps {
	return application.Deps{
		EmailIsTaken: func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			j.note("EmailIsTaken")
			return result.Ok[bool, domain.Error](false)
		},
		HashPassword: func(p domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
			j.note("HashPassword")
			j.hache = p.Expose()
			return result.Ok[domain.PasswordHash, domain.Error](domain.NewPasswordHash(condense))
		},
		SaveUser: func(_ context.Context, u domain.User) result.Result[domain.User, domain.Error] {
			j.note("SaveUser")
			j.sauvegarde = u
			return result.Ok[domain.User, domain.Error](u)
		},
		PublishEvent: func(
			_ context.Context, eventType, aggregateID string, payload any,
		) result.Result[struct{}, domain.Error] {
			j.note("PublishEvent")
			j.typeEvent, j.agregat, j.evenement = eventType, aggregateID, payload
			return result.Ok[struct{}, domain.Error](struct{}{})
		},
		GenerateID: func() domain.UserID { j.note("GenerateID"); return identifiant },
		Now:        func() time.Time { j.note("Now"); return instantFixe() },
	}
}

// echoue fabrique un port en erreur, pour n'importe quel type de succès.
func echoue[T any](code domain.ErrorCode, message string) result.Result[T, domain.Error] {
	return result.Err[T, domain.Error](domain.NewError(code, message))
}

// inscrit exécute le cas d'usage et rend son Result.
func inscrit(deps application.Deps) result.Result[domain.User, domain.Error] {
	return application.NewRegisterUser(deps)(context.Background(), commande())
}

// utilisateurDe extrait l'utilisateur d'un Result attendu en succès.
func utilisateurDe(t *testing.T, r result.Result[domain.User, domain.Error]) domain.User {
	t.Helper()
	user, err, ok := r.Get()
	if !ok {
		t.Fatalf("inscription refusée alors qu'elle devait réussir: %v", err)
	}
	return user
}

// codeDe extrait le code d'erreur d'un Result attendu en échec.
func codeDe(t *testing.T, r result.Result[domain.User, domain.Error]) domain.ErrorCode {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("un échec était attendu, reçu un succès")
	}
	return err.Code
}
