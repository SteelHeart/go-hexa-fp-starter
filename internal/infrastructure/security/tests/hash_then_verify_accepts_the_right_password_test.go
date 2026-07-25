package tests

import "testing"

// TestHashThenVerifyAcceptsTheRightPassword : le chemin nominal.
//
// Il paraît trivial et ne l'est pas : c'est lui qui échouerait si le format encodé
// changeait d'un côté sans l'autre, et ce défaut-là verrouillerait TOUS les comptes
// existants au prochain déploiement.
func TestHashThenVerifyAcceptsTheRightPassword(t *testing.T) {
	t.Parallel()

	const secret = "correct cheval batterie agrafe"
	hasher := newHasher()
	encoded := hash(t, secret)

	ok, err := hasher.Verify(secret, encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("le bon mot de passe doit être accepté")
	}
}
