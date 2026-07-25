package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestHashFailurePreventsPersistence : un hachage en échec arrête tout.
//
// Le danger serait un utilisateur écrit en base avec un condensé vide : il
// existerait, occuperait son adresse, et ne pourrait jamais se connecter. Pire, un
// condensé vide comparé naïvement pourrait laisser passer n'importe quel mot de
// passe selon l'implémentation de vérification.
//
// Le court-circuit rend ce scénario structurellement impossible.
func TestHashFailurePreventsPersistence(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	deps := depsNominales(observe)
	deps.HashPassword = func(domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
		observe.note("HashPassword")
		return echoue[domain.PasswordHash](domain.CodeInternal, "hachage indisponible")
	}

	if got := codeDe(t, inscrit(deps)); got != domain.CodeInternal {
		t.Errorf("code = %q, attendu %q", got, domain.CodeInternal)
	}
	if observe.aAppele("SaveUser") {
		t.Error("aucun utilisateur ne doit être écrit sans condensé valide")
	}
	if observe.aAppele("PublishEvent") {
		t.Error("aucun événement ne doit être publié")
	}
}
