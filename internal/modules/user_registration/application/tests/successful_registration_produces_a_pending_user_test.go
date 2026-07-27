package tests

import (
	"testing"
)

// TestSuccessfulRegistrationProducesAPendingUser : le chemin nominal, de bout en
// bout, sans base et sans conteneur.
//
// C'est ce que la pureté du cœur achète : ce test tourne en microsecondes et
// couvre pourtant l'intégralité de la règle métier. Les mêmes garanties par un test
// de bout en bout coûteraient une base, un courtier, et plusieurs secondes.
func TestSuccessfulRegistrationProducesAPendingUser(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	user := utilisateurDe(t, inscrit(depsNominales(observe)))

	if user.ID != identifiant {
		t.Errorf("identifiant = %q, attendu %q", user.ID, identifiant)
	}
	if user.Email.String() != adresseValide {
		t.Errorf("adresse = %q, attendu %q", user.Email.String(), adresseValide)
	}
	if user.PasswordHash.String() != condense {
		t.Errorf("condensé = %q, attendu celui rendu par le port", user.PasswordHash.String())
	}
	if !user.CreatedAt.Equal(instantFixe()) {
		t.Errorf("date de création = %v, attendu l'instant injecté", user.CreatedAt)
	}
	if user.CanAuthenticate() {
		t.Error("un compte fraîchement inscrit ne doit pas encore pouvoir se connecter")
	}
}
