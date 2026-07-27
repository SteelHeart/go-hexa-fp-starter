package tests

import "testing"

// TestEachHashUsesAFreshSalt : deux hachages du MÊME mot de passe donnent deux
// condensés différents.
//
// Sans sel aléatoire, deux comptes partageant un mot de passe auraient le même
// condensé : une simple requête `GROUP BY` sur la table révélerait qui utilise le
// même secret, et une table arc-en-ciel casserait tous ces comptes d'un coup.
//
// C'est la propriété la plus facile à perdre lors d'un refactoring — un sel dérivé
// de l'identifiant « pour être reproductible » suffit — et la plus silencieuse.
func TestEachHashUsesAFreshSalt(t *testing.T) {
	t.Parallel()

	const secret = "correct cheval batterie agrafe"
	hasher := newHasher()

	vus := map[string]struct{}{}
	for range 5 {
		encoded := hash(t, secret)
		if _, deja := vus[encoded]; deja {
			t.Fatal("deux hachages du même mot de passe ont produit le même condensé")
		}
		vus[encoded] = struct{}{}

		// Chaque condensé doit rester vérifiable : un sel aléatoire ne sert à rien
		// s'il n'est pas embarqué dans le résultat.
		ok, err := hasher.Verify(secret, encoded)
		if err != nil || !ok {
			t.Fatalf("condensé non vérifiable: ok=%v err=%v", ok, err)
		}
	}
}
