package tests

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestSameSubjectRegistersOnlyOnceUnderConcurrency ferme la fenêtre entre la
// vérification et l'écriture.
//
// # La fenêtre que ce test vise
//
// Le cas d'usage ne vérifie PAS l'unicité avant d'écrire, délibérément : entre une
// vérification et une écriture il existe un intervalle que deux demandes
// simultanées franchissent toutes les deux. C'est le magasin, qui détient le
// verrou, qui tranche — exactement comme une contrainte d'unicité SQL le ferait.
//
// Un test séquentiel passerait même avec la faute. Seize inscriptions
// concurrentes la font apparaître.
func TestSameSubjectRegistersOnlyOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	const attempts = 16
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		ids       = make(map[domain.IdentityID]bool)
	)

	wg.Add(attempts)
	for range attempts {
		go func() {
			defer wg.Done()
			identity, err := mod.Register(ctx, subject, secret)

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				succeeded++
				ids[identity.ID] = true
				return
			}
			if !errors.Is(err, domain.ErrSubjectTaken) {
				t.Errorf("refus attendu ErrSubjectTaken, obtenu %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded != 1 {
		t.Fatalf("%d inscriptions ont réussi sur le même sujet, attendu exactement 1", succeeded)
	}
	if len(ids) != 1 {
		t.Fatalf("%d identifiants distincts créés, attendu 1", len(ids))
	}
}

// TestEachAccountGetsItsOwnIdentifier exige deux comptes, deux identifiants.
//
// Un identifiant réutilisé ferait porter à quelqu'un les permissions d'un autre.
// Le test vérifie aussi que l'identité naît ACTIVE — contrairement à
// `user_registration`, dont le compte naît `pending`. La nuance est réelle :
// `auth` ne crée une identité que sur une demande déjà autorisée par son
// appelant, alors qu'une inscription publique doit être confirmée.
func TestEachAccountGetsItsOwnIdentifier(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	seen := make(map[domain.IdentityID]bool)
	for _, subj := range []string{"alice@example.com", "bob@example.com", "carol@example.com"} {
		identity, err := mod.Register(ctx, subj, secret)
		if err != nil {
			t.Fatalf("inscription de %q: %v", subj, err)
		}
		if seen[identity.ID] {
			t.Fatalf("identifiant réutilisé pour %q: %q", subj, identity.ID)
		}
		if !identity.Active {
			t.Fatalf("une identité d'authentification naît active ; %q ne l'est pas", subj)
		}
		if len(identity.Roles) != 0 {
			t.Fatalf("une identité naît SANS rôle ; %q en porte %v", subj, identity.Roles)
		}
		seen[identity.ID] = true
	}
}
