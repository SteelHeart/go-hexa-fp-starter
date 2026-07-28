package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestRevokingASessionIsImmediateAndIdempotent garde les deux propriétés d'une
// déconnexion.
//
// Immédiate : c'est ce qu'on attend d'une déconnexion, et c'est ce qu'un jeton
// signé autoportant ne sait pas faire sans liste de révocation — donc sans
// retomber sur le magasin qu'il prétendait éviter.
//
// Idempotente : un client qui se déconnecte deux fois n'a rien fait de mal. Faire
// échouer le second appel produirait une erreur que personne ne saurait traiter,
// et que tout le monde finirait par ignorer — y compris quand elle signale autre
// chose.
func TestRevokingASessionIsImmediateAndIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("le jeton vient d'être émis : %v", err)
	}

	if err := mod.Revoke(ctx, session.Token); err != nil {
		t.Fatalf("révocation: %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("jeton révoqué : attendu ErrTokenUnknown, obtenu %v", err)
	}

	if err := mod.Revoke(ctx, session.Token); err != nil {
		t.Fatalf("révoquer deux fois ne doit pas être une erreur : %v", err)
	}
}

// TestRevokingOneSessionSparesTheOthers empêche qu'une déconnexion en devienne
// une globale.
//
// Deux connexions du même compte — un téléphone et un poste — produisent deux
// jetons. Se déconnecter de l'un ne doit pas déconnecter l'autre. La faute
// inverse, indexer la session sur l'identité plutôt que sur le jeton, se remarque
// seulement le jour où quelqu'un se connecte depuis deux appareils.
func TestRevokingOneSessionSparesTheOthers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	first, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("première authentification: %v", err)
	}
	second, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("seconde authentification: %v", err)
	}
	if first.Token.Equals(second.Token) {
		t.Fatal("deux authentifications doivent produire deux jetons distincts")
	}

	if err := mod.Revoke(ctx, first.Token); err != nil {
		t.Fatalf("révocation: %v", err)
	}
	if _, err := mod.Verify(ctx, second.Token); err != nil {
		t.Fatalf("la seconde session ne devait pas être touchée : %v", err)
	}
}

// TestVerifyRefusesAnEmptyToken : la valeur zéro n'ouvre rien.
//
// Un jeton non construit est une chaîne vide. Sans ce refus, il deviendrait une
// clé légitime dans le magasin — et la première session enregistrée sous cette
// clé ouvrirait à quiconque n'envoie aucun jeton.
func TestVerifyRefusesAnEmptyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if _, err := mod.Verify(ctx, domain.Token{}); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("jeton vide : attendu ErrTokenUnknown, obtenu %v", err)
	}
	if err := mod.Revoke(ctx, domain.Token{}); !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("révocation d'un jeton vide : attendu ErrIncomplete, obtenu %v", err)
	}
}
