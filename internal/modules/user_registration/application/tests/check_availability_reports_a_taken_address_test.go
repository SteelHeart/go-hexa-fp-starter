package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCheckAvailabilityReportsATakenAddress : une adresse prise n'est pas
// disponible, et une panne n'est pas une disponibilité.
//
// Le second point est le vrai piège. Si une base injoignable rendait « disponible »,
// une panne transformerait le formulaire en machine à créer des doublons — que la
// contrainte d'unicité refuserait ensuite, avec un message incompréhensible pour
// l'utilisateur qui vient de voir une coche verte.
//
// L'erreur remonte donc telle quelle : « je ne sais pas » n'est pas « oui ».
func TestCheckAvailabilityReportsATakenAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	prise := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			return result.Ok[bool, domain.Error](true)
		},
	)
	disponible, _, ok := prise(ctx, adresseValide).Get()
	if !ok {
		t.Fatal("une adresse prise doit rendre un succès portant false, pas une erreur")
	}
	if disponible {
		t.Error("une adresse déjà enregistrée ne doit PAS être annoncée disponible")
	}

	enPanne := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			return echoue[bool](domain.CodeUnavailable, "base injoignable")
		},
	)
	_, err, ok := enPanne(ctx, adresseValide).Get()
	if ok {
		t.Fatal("une panne ne doit pas se transformer en disponibilité")
	}
	if err.Code != domain.CodeUnavailable {
		t.Errorf("code = %q, attendu %q", err.Code, domain.CodeUnavailable)
	}
}
