package tests

import "testing"

// TestVerifyRejectsAWrongPassword : un mot de passe incorrect est refusé SANS
// erreur.
//
// La distinction compte. Une erreur signale un défaut de données — condensé
// illisible, format inconnu — et mérite une alerte. Un refus est le cas nominal
// d'une tentative de connexion ratée, et il s'en produit des milliers par jour.
// Les confondre remplirait les alertes de bruit, ou masquerait un vrai incident.
func TestVerifyRejectsAWrongPassword(t *testing.T) {
	t.Parallel()

	hasher := newHasher()
	encoded := hash(t, "correct cheval batterie agrafe")

	for _, wrong := range []string{
		"correct cheval batterie agraf",  // un caractère de moins
		"Correct cheval batterie agrafe", // casse différente
		"",
		"tout autre chose",
	} {
		ok, err := hasher.Verify(wrong, encoded)
		if err != nil {
			t.Errorf("Verify(%q) a rendu une erreur au lieu d'un refus: %v", wrong, err)
		}
		if ok {
			t.Errorf("le mot de passe %q ne doit PAS être accepté", wrong)
		}
	}
}
