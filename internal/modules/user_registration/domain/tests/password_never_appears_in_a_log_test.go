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

	// Le passage par fmt est DÉLIBÉRÉ, et gocritic a tort de proposer
	// `value.String()` : c'est précisément le chemin `fmt` qu'on teste. Un
	// développeur qui laisse fuir un mot de passe ne le fait jamais en appelant
	// String() — il le fait en passant la valeur à un `%v` de journal. Appeler
	// String() ici testerait la méthode au lieu du chemin de fuite, et le test
	// resterait vert le jour où quelqu'un retire l'implémentation de Stringer.
	formats := map[string]string{
		"%v":  fmt.Sprintf("%v", value), //nolint:gocritic // le chemin fmt EST l'objet du test, pas un détour
		"%s":  fmt.Sprintf("%s", value), //nolint:gocritic,staticcheck // idem : String() contournerait la fuite testée
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
