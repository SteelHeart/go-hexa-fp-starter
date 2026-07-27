package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestInvalidInputTouchesNoPort : une commande invalide n'atteint AUCUN effet.
//
// La validation est la première étape du pipeline, et `result.Chain` court-circuite
// au premier échec. Conséquence concrète : une saisie fautive ne consomme ni
// connexion à la base, ni cycle de hachage. Un formulaire mal rempli en boucle ne
// coûte donc presque rien au serveur.
//
// C'est aussi ce qui garantit qu'aucune donnée invalide ne franchit la frontière :
// les étapes suivantes ne reçoivent que des types du domaine.
func TestInvalidInputTouchesNoPort(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cmd  domain.RegistrationCommand
		code domain.ErrorCode
	}{
		"adresse invalide": {
			cmd:  domain.RegistrationCommand{Email: "pas une adresse", Password: motDePasseFort},
			code: domain.CodeInvalidEmail,
		},
		"mot de passe faible": {
			cmd:  domain.RegistrationCommand{Email: adresseValide, Password: "court"},
			code: domain.CodeWeakPassword,
		},
		"commande vide": {
			cmd:  domain.RegistrationCommand{},
			code: domain.CodeInvalidEmail,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			observe := &journal{}
			registre := application.NewRegisterUser(depsNominales(observe))

			_, err, ok := registre(context.Background(), tc.cmd).Get()
			if ok {
				t.Fatal("une commande invalide doit être refusée")
			}
			if err.Code != tc.code {
				t.Errorf("code = %q, attendu %q", err.Code, tc.code)
			}
			if len(observe.appels) != 0 {
				t.Errorf("ports appelés = %v, attendu aucun", observe.appels)
			}
		})
	}
}
