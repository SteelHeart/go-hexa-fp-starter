// Package tests contient les tests en BOÎTE NOIRE du module auth : ils n'utilisent
// que l'API publique, exactement comme une surface HTTP ou CLI le ferait.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
//
// # Aucune bibliothèque de simulacre
//
// Les ports sont des types fonction (ADR 003) : un double de test est une closure
// de trois lignes. C'est ce qui rend une bibliothèque de simulacre inutile, donc
// interdite (rules/dependances.md).
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

const (
	// secret est un secret valide — au moins douze caractères.
	secret = "un-secret-assez-long"

	// subject est le sujet utilisé par défaut.
	subject = "alice@example.com"

	// hashPrefix marque le condensé factice produit par le double de hachage.
	//
	// Le préfixe n'est pas décoratif : c'est lui qui rend OBSERVABLE la
	// différence entre le clair et le condensé, donc testable le fait que le
	// clair n'atteint jamais le magasin.
	hashPrefix = "condensé:"
)

// clock est une horloge que le test avance à la main.
//
// Le module reçoit son instant par un port — il ne lit jamais `time.Now()`. C'est
// ce qui permet de vérifier l'expiration d'une session en une ligne, plutôt qu'en
// attendant douze heures.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

// newClock fige un instant de référence.
func newClock() *clock {
	return &clock{now: time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)}
}

// Now rend l'instant courant du test.
func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance fait avancer l'horloge.
func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// hashSecret est un hachage FACTICE, réversible et instantané.
//
// Argon2id est délibérément lent : l'utiliser ici ferait payer des dizaines de
// millisecondes à chaque inscription, pour ne rien vérifier de plus. Ce que ces
// tests éprouvent, c'est le CHEMIN du secret — pas la solidité du condensé, qui
// est éprouvée dans internal/infrastructure/security.
func hashSecret(plain string) (string, error) { return hashPrefix + plain, nil }

// verifySecret compare un secret au condensé factice.
func verifySecret(plain, encoded string) (bool, error) { return encoded == hashPrefix+plain, nil }

// deps assemble les dépendances du module autour d'une horloge.
func deps(c *clock) auth.Deps {
	return auth.Deps{HashSecret: hashSecret, VerifySecret: verifySecret, Now: c.Now}
}

// newModule construit le module sur son pilote par défaut.
func newModule(t *testing.T, options map[string]any) (auth.Module, *clock) {
	t.Helper()

	c := newClock()
	mod, err := auth.New(
		config.Module{Enabled: true, Driver: "memory", Options: options},
		deps(c),
	)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod, c
}

// register inscrit une identité et rend son identifiant.
func register(t *testing.T, mod auth.Module, subj string) domain.IdentityID {
	t.Helper()

	identity, err := mod.Register(context.Background(), subj, secret)
	if err != nil {
		t.Fatalf("inscription de %q: %v", subj, err)
	}
	return identity.ID
}

// permission construit une permission ou fait échouer le test.
func permission(t *testing.T, raw string) domain.Permission {
	t.Helper()

	p, err := domain.NewPermission(raw)
	if err != nil {
		t.Fatalf("permission %q: %v", raw, err)
	}
	return p
}

// grant définit un rôle et l'affecte à une identité.
func grant(t *testing.T, mod auth.Module, id domain.IdentityID, role string, permissions ...string) {
	t.Helper()

	ctx := context.Background()
	if err := mod.DefineRole(ctx, role, permissions); err != nil {
		t.Fatalf("définition du rôle %q: %v", role, err)
	}
	if err := mod.AssignRoles(ctx, id, []string{role}); err != nil {
		t.Fatalf("affectation du rôle %q: %v", role, err)
	}
}
