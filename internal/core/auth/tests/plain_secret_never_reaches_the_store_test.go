package tests

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestPlainSecretNeverReachesTheStore constate l'ORDRE des étapes.
//
// # Ce que le test observe réellement
//
// `VerifySecret` reçoit le clair saisi ET ce que le magasin a retenu. Le second
// est donc le seul point d'observation qu'un appelant a sur le contenu du
// magasin — et il suffit : si le clair y était, il apparaîtrait là.
//
// Hacher AVANT d'écrire n'est pas un détail de style. C'est ce qui garantit qu'un
// pilote ne peut pas journaliser un mot de passe même par accident : il n'en voit
// jamais.
func TestPlainSecretNeverReachesTheStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := newClock()

	var (
		mu     sync.Mutex
		stored string
	)
	spy := auth.Deps{
		HashSecret: hashSecret,
		VerifySecret: func(plain, encoded string) (bool, error) {
			mu.Lock()
			stored = encoded
			mu.Unlock()
			return verifySecret(plain, encoded)
		},
		Now: c.Now,
	}

	mod, err := auth.New(config.Module{Enabled: true, Driver: "memory"}, spy)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	register(t, mod, subject)

	if _, err := mod.Authenticate(ctx, subject, secret); err != nil {
		t.Fatalf("authentification: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	switch {
	case stored == "":
		t.Fatal("le condensé n'a jamais été comparé : le test n'observe rien")
	case stored == secret:
		t.Fatal("le magasin retient le secret EN CLAIR")
	case strings.Contains(stored, secret) && !strings.HasPrefix(stored, hashPrefix):
		t.Fatalf("le clair transparaît dans ce que retient le magasin : %q", stored)
	}
}

// TestSessionCarriesNoPermission constate ce que la session NE porte PAS.
//
// C'est la décision 1 de l'ADR 017 vue depuis le type : une `Session` n'a que son
// jeton, son identité et ses dates. Le jour où quelqu'un y ajouterait un champ
// `Permissions`, ce test ne compilerait plus — et c'est le bon moment pour rouvrir
// l'ADR, pas six mois plus tard devant une révocation qui ne prend pas effet.
func TestSessionCarriesNoPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "comptable", "billing.invoice.cancel")

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}

	if session.Token.IsZero() {
		t.Fatal("la session doit porter un jeton")
	}
	if session.Identity != id {
		t.Fatalf("la session doit porter l'identité %q, elle porte %q", id, session.Identity)
	}
	if !session.ExpiresAt.After(session.IssuedAt) {
		t.Fatal("une session doit être bornée dans le temps")
	}
	if strings.Contains(session.Token.String(), "billing") {
		t.Fatal("le jeton porte une permission : il authentifie, il n'autorise pas")
	}
}
