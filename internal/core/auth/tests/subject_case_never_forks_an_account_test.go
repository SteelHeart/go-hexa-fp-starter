package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestSubjectCaseNeverForksAnAccount : `Alice@Example.COM ` et `alice@example.com`
// sont UNE identité, pas deux.
//
// # La faute, quand la normalisation manque
//
// Deux comptes coexistent. L'inscription réussit deux fois, donc rien ne signale
// quoi que ce soit. Puis quelqu'un se connecte avec la casse « de l'autre »
// compte, tombe sur un compte vide, et personne ne comprend — surtout pas au
// support, qui voit bien l'adresse dans la base.
//
// La normalisation est portée par le TYPE : le champ de `Subject` est privé, donc
// un sujet non normalisé ne peut pas exister hors du domaine.
func TestSubjectCaseNeverForksAnAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, "  Alice@Example.COM ")

	if _, err := mod.Register(ctx, "alice@example.com", secret); !errors.Is(err, domain.ErrSubjectTaken) {
		t.Fatalf("la même adresse à la casse près : attendu ErrSubjectTaken, obtenu %v", err)
	}

	for _, variant := range []string{"alice@example.com", "ALICE@EXAMPLE.COM", " Alice@Example.Com "} {
		if _, err := mod.Authenticate(ctx, variant, secret); err != nil {
			t.Errorf("authentification avec %q : %v", variant, err)
		}
	}
}

// TestMalformedSubjectIsRefusedBeforeTheStore garde le refus EN AMONT du pilote.
//
// Le refus est en amont pour deux raisons : il ne coûte aucune requête, et il ne
// laisse pas une chaîne vide atteindre un pilote, où elle deviendrait une clé
// légitime — donc un compte que n'importe qui pourrait revendiquer en n'envoyant
// rien.
func TestMalformedSubjectIsRefusedBeforeTheStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	for _, raw := range []string{"", "   ", "\t", "alice bob@example.com"} {
		if _, err := mod.Register(ctx, raw, secret); !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("sujet %q : attendu ErrIncomplete, obtenu %v", raw, err)
		}
	}
}

// TestShortSecretIsRefused : douze caractères, et aucune règle de composition.
//
// La longueur est la seule contrainte qui augmente réellement l'entropie. Les
// règles de composition — une majuscule, un chiffre, un caractère spécial —
// poussent surtout à écrire `Password1!`, qui satisfait les quatre et ne résiste
// à rien.
func TestShortSecretIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if _, err := mod.Register(ctx, subject, "onze-carac"); !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("secret trop court : attendu ErrIncomplete, obtenu %v", err)
	}

	// Douze caractères sans majuscule, sans chiffre, sans caractère spécial :
	// accepté. La règle est la longueur, et elle seule.
	if _, err := mod.Register(ctx, subject, "abcdefghijkl"); err != nil {
		t.Fatalf("aucune règle de composition ne doit s'appliquer : %v", err)
	}
}

// TestSubjectIsMaskedForLogs : un sujet est une donnée personnelle.
//
// Il ne se journalise jamais en clair (rules/securite.md §5) — et c'est le
// journal d'authentification qu'on exporte le plus volontiers vers un collecteur
// tiers.
func TestSubjectIsMaskedForLogs(t *testing.T) {
	t.Parallel()

	subj, err := domain.NewSubject("alice@example.com")
	if err != nil {
		t.Fatalf("sujet: %v", err)
	}

	masked := subj.Masked()
	if masked == subj.String() {
		t.Fatal("la forme masquée ne doit pas être le sujet en clair")
	}
	if masked != "a***@example.com" {
		t.Fatalf("forme masquée inattendue : %q", masked)
	}

	short, err := domain.NewSubject("ab")
	if err != nil {
		t.Fatalf("sujet court: %v", err)
	}
	if short.Masked() != "***" {
		t.Fatalf("un sujet trop court doit disparaître entièrement, obtenu %q", short.Masked())
	}
}
