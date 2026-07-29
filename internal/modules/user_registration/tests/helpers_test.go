// Package tests exercises the registration module as a BLACK BOX.
//
// It only imports what a surface would import: the module and its domain. No
// access to the driver, no access to the internal state — a test that inspects
// the inside locks the implementation down and forbids changing it.
package tests

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// fixedInstant is the clock of every test: deterministic, so that no test
// depends on the moment at which it runs.
var fixedInstant = time.Date(2026, time.July, 25, 10, 30, 0, 0, time.UTC)

// spyPublisher retains the published events, without ever failing.
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

// fakeHash produces a recognisable digest without costing an Argon2.
//
// A real hash would make every test a hundred times slower while proving nothing
// more: what is verified here is the WIRING, not the cryptography — it has its
// own tests in internal/infrastructure/security.
func fakeHash(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("hashed:" + password.Expose()),
	)
}

// sequentialIDs hands out predictable identifiers.
//
// # Why the counter is atomic
//
// The generator is built ONCE per module and then called from every goroutine
// the module serves. `TestConcurrentRegistrationsNeverDuplicateAnAddress` fires
// sixteen at once, so a plain `counter++` is a data race — read, increment and
// write are three operations, and nothing orders them.
//
// It was one, silently, until CI caught it: the race detector is sound but
// INCOMPLETE — it only reports what it happens to observe. The defect survived
// because the schedule never exposed it, not because it was not there. It
// surfaced on a translation lot that changed nothing but the length of a few
// strings.
//
// A racy helper makes the very guarantee the test exists for unprovable: one
// cannot verify that a store settles concurrency using a generator that does
// not survive it.
func sequentialIDs() ports.GenerateID {
	var counter atomic.Int64
	return func() domain.UserID {
		return domain.UserID("user-" + string(rune('0'+counter.Add(1))))
	}
}

// newModule mounts the module on its default driver.
func newModule(t *testing.T, publisher *spyPublisher) userregistration.Module {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err != nil {
		t.Fatalf("mounting the module: %v", err)
	}
	return mod
}

// register is the shorthand for a successful registration.
func register(t *testing.T, mod userregistration.Module, email, password string) domain.User {
	t.Helper()

	user, failure, ok := mod.Register(context.Background(), domain.RegistrationCommand{
		Email:    email,
		Password: password,
	}).Get()
	if !ok {
		t.Fatalf("registration of %q refused: %v", email, failure)
	}
	return user
}

// validPassword satisfies the bounds of the domain.
const validPassword = "correct horse battery staple"
