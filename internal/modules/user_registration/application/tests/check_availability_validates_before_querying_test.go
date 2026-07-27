package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCheckAvailabilityValidatesBeforeQuerying : l'adresse est validée AVANT
// d'interroger le stockage.
//
// Ce cas d'usage est le point d'entrée d'un champ de formulaire : il est appelé à
// chaque frappe, sans authentification, depuis n'importe quel client. Sans
// validation préalable, chaque caractère saisi deviendrait une requête — un moyen
// gratuit de faire travailler la base.
//
// Il rend `true` pour DISPONIBLE : la double négation « pas pris » est une source
// d'erreur classique, et l'inverser rendrait toutes les adresses libres.
func TestCheckAvailabilityValidatesBeforeQuerying(t *testing.T) {
	t.Parallel()

	interroge := false
	verifie := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			interroge = true
			return result.Ok[bool, domain.Error](false)
		},
	)
	ctx := context.Background()

	_, err, ok := verifie(ctx, "pas une adresse").Get()
	if ok {
		t.Fatal("une adresse invalide doit être refusée")
	}
	if err.Code != domain.CodeInvalidEmail {
		t.Errorf("code = %q, attendu %q", err.Code, domain.CodeInvalidEmail)
	}
	if interroge {
		t.Error("le stockage ne doit PAS être interrogé pour une adresse invalide")
	}

	disponible, _, ok := verifie(ctx, adresseValide).Get()
	if !ok {
		t.Fatal("une adresse valide et libre doit rendre un succès")
	}
	if !disponible {
		t.Error("une adresse NON prise doit être rendue DISPONIBLE")
	}
}
