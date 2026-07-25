package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTakenEmailIsRefusedBeforeHashing : l'adresse déjà prise est refusée AVANT le
// hachage.
//
// L'ordre n'est pas cosmétique. Argon2id est délibérément lent et gourmand en
// mémoire — c'est sa raison d'être. Hacher avant de vérifier la disponibilité
// offrirait à quiconque un moyen de saturer le serveur en soumettant en boucle une
// adresse dont il sait qu'elle existe déjà.
//
// Le test vérifie donc les deux choses : le bon code d'erreur, ET le fait que le
// port coûteux n'a pas été appelé.
func TestTakenEmailIsRefusedBeforeHashing(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	deps := depsNominales(observe)
	deps.EmailIsTaken = func(context.Context, domain.Email) result.Result[bool, domain.Error] {
		observe.note("EmailIsTaken")
		return result.Ok[bool, domain.Error](true)
	}

	if got := codeDe(t, inscrit(deps)); got != domain.CodeEmailAlreadyExists {
		t.Errorf("code = %q, attendu %q", got, domain.CodeEmailAlreadyExists)
	}
	if observe.aAppele("HashPassword") {
		t.Error("le hachage ne doit PAS être payé pour une adresse déjà prise")
	}
	if observe.aAppele("SaveUser") {
		t.Error("aucune écriture ne doit avoir lieu")
	}
}
