package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEmailIsMaskedBeforeLogging : un courriel est une donnée personnelle et ne se
// journalise JAMAIS en clair (rules/securite.md §5).
//
// La forme masquée doit rester utile au diagnostic — le domaine reste visible, ce
// qui suffit à distinguer une panne de messagerie d'entreprise d'un problème
// général — sans permettre de reconstituer l'adresse.
//
// Un journal est conservé longtemps, souvent hors du périmètre de la base, et lu
// par des humains en incident. C'est précisément l'endroit où une fuite ne se
// remarque pas.
func TestEmailIsMaskedBeforeLogging(t *testing.T) {
	t.Parallel()

	masque := emailValide(t, "alice.martin@example.com").Masked()

	if strings.Contains(masque, "alice.martin") {
		t.Errorf("forme masquée = %q : la partie locale doit être cachée", masque)
	}
	if !strings.Contains(masque, "example.com") {
		t.Errorf("forme masquée = %q : le domaine doit rester lisible pour le diagnostic", masque)
	}
	if !strings.HasPrefix(masque, "a") {
		t.Errorf("forme masquée = %q : la première lettre aide à distinguer deux comptes", masque)
	}

	// Une adresse non construite ne doit rien révéler non plus.
	var jamaisConstruite domain.Email
	if got := jamaisConstruite.Masked(); strings.Contains(got, "@") {
		t.Errorf("adresse vide masquée = %q, attendu un marqueur opaque", got)
	}
}
