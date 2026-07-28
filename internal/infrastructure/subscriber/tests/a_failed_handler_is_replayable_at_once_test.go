package tests

import (
	"context"
	"errors"
	"testing"

	idempotencydomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// errFournisseur simule une panne du destinataire de l'effet.
var errFournisseur = errors.New("fournisseur injoignable")

// TestAFailedHandlerIsReplayableAtOnce garde la libération de la clé.
//
// # Ce que l'oubli produit
//
// La clé reste réservée. L'enveloppe devient donc INTRAITABLE jusqu'à
// l'expiration — vingt-quatre heures avec la configuration livrée — et chaque
// rejeu se heurte à « déjà en cours » sans que rien n'explique pourquoi.
//
// Le remède serait alors pire que le mal : quelqu'un raccourcirait le TTL pour
// « débloquer », et rouvrirait la fenêtre de rejeu que ce module existe pour
// fermer.
//
// Le test constate la propriété qui compte : après un échec, la remise suivante
// EXÉCUTE — elle n'est ni refusée, ni prise pour un rejeu.
func TestAFailedHandlerIsReplayableAtOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	guard := newGuard(t)
	env := envelope("evt-echec")

	rate := echouant(errFournisseur)
	if err := subscriber.Once(guard, rate.handler())(ctx, env); !errors.Is(err, errFournisseur) {
		t.Fatalf("l'échec du gestionnaire doit remonter INTACT, obtenu %v", err)
	}
	if rate.total() != 1 {
		t.Fatalf("le gestionnaire devait être appelé une fois, obtenu %d", rate.total())
	}

	// Même clé, même enveloppe, sur le MÊME garde : la seconde remise doit
	// exécuter, pas être prise pour un rejeu.
	reussi := &compteur{}
	if err := subscriber.Once(guard, reussi.handler())(ctx, env); err != nil {
		t.Fatalf("la remise après échec doit aboutir : %v", err)
	}
	if reussi.total() != 1 {
		t.Fatal("la remise après échec n'a pas exécuté : la clé est restée verrouillée")
	}
}

// TestAnEnvelopeWithoutIDIsRefusedNotSilentlyLetThrough refuse bruyamment.
//
// L'identifiant EST la clé d'idempotence. Sans lui, aucun rejeu ne peut être
// reconnu — et un décorateur qui laisserait passer offrirait une garantie
// SILENCIEUSEMENT absente, ce qui est pire que pas de garde du tout : on cesse
// de se méfier.
func TestAnEnvelopeWithoutIDIsRefusedNotSilentlyLetThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effet := &compteur{}
	handler := subscriber.Once(newGuard(t), effet.handler())

	env := envelope("")
	if err := handler(ctx, env); !errors.Is(err, subscriber.ErrMissingID) {
		t.Fatalf("attendu ErrMissingID, obtenu %v", err)
	}
	if effet.total() != 0 {
		t.Fatal("une enveloppe sans identifiant ne doit produire aucun effet")
	}
}

// TestTheHandlerErrorSurvivesTheDecorator garde la CAUSE de l'échec.
//
// L'erreur du gestionnaire doit remonter reconnaissable par `errors.Is`. La
// remplacer par une erreur du décorateur ferait perdre la CAUSE — et le
// dépileur, en amont, décide de rejouer ou d'abandonner sur cette cause.
//
// Le test vérifie aussi qu'une erreur de libération ne l'écrase pas : les deux
// sont jointes, parce qu'une clé restée verrouillée est un incident distinct qui
// se diagnostique mal quand son message a disparu.
func TestTheHandlerErrorSurvivesTheDecorator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errLiberation := errors.New("magasin injoignable")

	guard := newGuard(t)
	guard.Release = func(context.Context, idempotencydomain.Key) error { return errLiberation }

	rate := echouant(errFournisseur)
	err := subscriber.Once(guard, rate.handler())(ctx, envelope("evt-double-echec"))

	if !errors.Is(err, errFournisseur) {
		t.Errorf("la cause du gestionnaire doit survivre : %v", err)
	}
	if !errors.Is(err, errLiberation) {
		t.Errorf("l'échec de libération doit être joint, pas écrasé : %v", err)
	}
}
