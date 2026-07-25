package tests

import "testing"

// TestHashThenVerifyAcceptsTheRightPassword : le chemin nominal.
//
// Il paraît trivial et ne l'est pas : c'est lui qui échouerait si le format encodé
// changeait d'un côté sans l'autre, et ce défaut-là verrouillerait TOUS les comptes
// existants au prochain déploiement.
//
// Les cas ne sont pas décoratifs. Le format encodé est
// `$argon2id$v=…$m=…,t=…,p=…$sel$condensé` : il utilise `$` comme séparateur. Un
// mot de passe qui contient `$` est donc le cas où une implémentation naïve — qui
// découperait avant de décoder, ou qui concaténerait sans échapper — se briserait.
// De même, un mot de passe non-ASCII vérifie que le hachage porte sur des OCTETS et
// non sur des runes : compter en runes n'échouerait qu'en production, pour les seuls
// utilisateurs dont la langue s'écrit avec des accents.
func TestHashThenVerifyAcceptsTheRightPassword(t *testing.T) {
	t.Parallel()

	secrets := map[string]string{
		"phrase ASCII":         "correct cheval batterie agrafe",
		"accents et espaces":   "un été à Nîmes, où l'on flâne",
		"contient le sépareur": "mot$de$passe$argon2id$v=19",
		"symboles et chiffres": "9!£{}[]|\\~`^@#%&*()_+-=<>?/",
		"idéogrammes":          "正しい馬のバッテリーステープル",
	}

	hasher := newHasher()
	for name, secret := range secrets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded := hash(t, secret)

			ok, err := hasher.Verify(secret, encoded)
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if !ok {
				t.Error("le bon mot de passe doit être accepté")
			}
		})
	}
}
