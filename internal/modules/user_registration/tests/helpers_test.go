// Package tests éprouve le module d'inscription en BOÎTE NOIRE.
//
// Il n'importe que ce qu'une surface importerait : le module et son domaine.
// Aucun accès au pilote, aucun accès à l'état interne — un test qui inspecte
// l'intérieur verrouille l'implémentation et interdit de la changer.
package tests

import (
	"context"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// fixedInstant est l'horloge de tous les tests : déterministe, donc aucun test
// ne dépend du moment où il s'exécute.
var fixedInstant = time.Date(2026, time.July, 25, 10, 30, 0, 0, time.UTC)

// spyPublisher retient les événements publiés, sans jamais échouer.
type spyPublisher struct {
	types        []string
	aggregateIDs []string
}

func (s *spyPublisher) port() ports.PublishEvent {
	return func(_ context.Context, eventType, aggregateID string, _ any) result.Result[domain.Ack, domain.Error] {
		s.types = append(s.types, eventType)
		s.aggregateIDs = append(s.aggregateIDs, aggregateID)
		return result.Ok[domain.Ack, domain.Error](domain.Ack{})
	}
}

// fakeHash produit un condensé reconnaissable sans coûter d'Argon2.
//
// Un vrai hachage rendrait chaque test cent fois plus lent pour ne rien prouver
// de plus : ce qu'on vérifie ici est le CÂBLAGE, pas la cryptographie — elle a
// ses propres tests dans internal/infrastructure/security.
func fakeHash(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("hashed:" + password.Expose()),
	)
}

// sequentialIDs distribue des identifiants prévisibles.
func sequentialIDs() ports.GenerateID {
	counter := 0
	return func() domain.UserID {
		counter++
		return domain.UserID("user-" + string(rune('0'+counter)))
	}
}

// newModule monte le module sur son pilote par défaut.
func newModule(t *testing.T, publisher *spyPublisher) userregistration.Module {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err != nil {
		t.Fatalf("montage du module: %v", err)
	}
	return mod
}

// register est le raccourci d'une inscription réussie.
func register(t *testing.T, mod userregistration.Module, email, password string) domain.User {
	t.Helper()

	user, failure, ok := mod.Register(context.Background(), domain.RegistrationCommand{
		Email:    email,
		Password: password,
	}).Get()
	if !ok {
		t.Fatalf("inscription de %q refusée: %v", email, failure)
	}
	return user
}

// validPassword satisfait les bornes du domaine.
const validPassword = "correct cheval batterie agrafe"
