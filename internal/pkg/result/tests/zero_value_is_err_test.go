package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestZeroValueIsErr est le test le plus important de la primitive.
//
// La valeur zéro d'un Result est un Err. C'est « deny par défaut » jusque dans le
// typage : un Result oublié — champ non initialisé, retour d'une branche qu'on
// croyait inatteignable — ÉCHOUE, au lieu de réussir silencieusement en portant la
// valeur zéro de T.
//
// Si cette propriété tombait, un `var r Result[User, Error]` non affecté
// ressemblerait à une inscription réussie portant un utilisateur vide.
func TestZeroValueIsErr(t *testing.T) {
	t.Parallel()

	var oublie result.Result[int, erreur]

	if oublie.IsOk() {
		t.Fatal("la valeur zéro d'un Result doit être une erreur, jamais un succès")
	}
	if !oublie.IsErr() {
		t.Error("IsErr doit être vrai sur la valeur zéro")
	}
	if _, _, ok := oublie.Get(); ok {
		t.Error("Get doit rendre ok=false sur la valeur zéro")
	}
	if got := oublie.ValueOr(42); got != 42 {
		t.Errorf("ValueOr = %d, attendu le repli 42", got)
	}
}
