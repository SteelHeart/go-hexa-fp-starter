package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestPasswordNeverAppearsInALog est le test de sécurité le plus important du
// domaine.
//
// Un mot de passe en clair dans un journal est une fuite définitive : les journaux
// sont conservés longtemps, dupliqués vers des agrégateurs, et lus par des humains.
// Le rotationner ne suffit pas — il faut prévenir tous les utilisateurs.
//
// La protection est structurelle : `String()` rend un marqueur, donc `%v` et `%s`
// masquent la valeur. Ce test vérifie aussi le cas qui fait vraiment les fuites —
// un `%+v` sur une STRUCTURE qui contient le mot de passe, écrit sans y penser
// dans un journal de débogage.
func TestPasswordNeverAppearsInALog(t *testing.T) {
	t.Parallel()

	const secret = "correct cheval batterie agrafe"
	value, _, ok := domain.NewRawPassword(secret).Get()
	if !ok {
		t.Fatal("le mot de passe de test devait être accepté")
	}

	formats := map[string]string{
		"%v":  fmt.Sprintf("%v", value),
		"%s":  fmt.Sprintf("%s", value),
		"%+v": fmt.Sprintf("%+v", struct{ Password domain.RawPassword }{value}),
		"%v struct": fmt.Sprintf("%v", struct {
			Email    string
			Password domain.RawPassword
		}{"alice@example.com", value}),
	}

	for format, rendu := range formats {
		if strings.Contains(rendu, secret) {
			t.Errorf("%s a laissé fuir le mot de passe: %q", format, rendu)
		}
	}

	// Expose reste le seul chemin vers la valeur, et il est nommé pour se voir en
	// revue : il ne doit y avoir qu'un appel dans tout le dépôt, celui du hachage.
	if value.Expose() != secret {
		t.Error("Expose doit rendre la valeur réelle : c'est sa seule raison d'être")
	}
}
