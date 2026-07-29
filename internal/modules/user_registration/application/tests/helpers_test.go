// Package tests holds the BLACK BOX tests of the registration use cases: they
// only use the public API, exactly like a primary adapter would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
//
// # No mocking library, and that is not a deprivation
//
// Every port is a FUNCTION type: a test double is a three-line closure. No code
// generation, no magic call assertions, no `.On("Method").Return(...)` that
// breaks at the first signature change. The compiler checks the doubles the way
// it checks the rest (rules/dependances.md).
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Scenario constants, so that the tests read like sentences.
const (
	validAddress   = "alice@example.com"
	strongPassword = "correct horse battery staple"
	digest         = "$argon2id$v=19$m=65536,t=3,p=2$c2Vs$Y29uZGVuc2U"
	identifier     = domain.UserID("user-42")
)

// fixedInstant is the clock of the tests: no test reads the real time.
func fixedInstant() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// callLog records what the pipeline actually called, and in which order.
//
// The order matters as much as the content: hashing before checking the
// availability of the address would be functionally correct and costly in
// practice.
type callLog struct {
	calls     []string
	saved     domain.User
	event     any
	eventType string
	aggregate string
	hashed    string
}

func (j *callLog) note(name string) { j.calls = append(j.calls, name) }

func (j *callLog) called(name string) bool {
	for _, call := range j.calls {
		if call == name {
			return true
		}
	}
	return false
}

// command forges a valid registration intent.
func command() domain.RegistrationCommand {
	return domain.RegistrationCommand{Email: validAddress, Password: strongPassword}
}

// nominalDeps builds a set of ports where everything succeeds, and which records
// its calls.
//
// Every test starts from there and replaces ONLY the port it cares about: what
// changes in the test is exactly what the test verifies.
func nominalDeps(j *callLog) application.Deps {
	return application.Deps{
		EmailIsTaken: func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			j.note("EmailIsTaken")
			return result.Ok[bool, domain.Error](false)
		},
		HashPassword: func(p domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
			j.note("HashPassword")
			j.hashed = p.Expose()
			return result.Ok[domain.PasswordHash, domain.Error](domain.NewPasswordHash(digest))
		},
		SaveUser: func(_ context.Context, u domain.User) result.Result[domain.User, domain.Error] {
			j.note("SaveUser")
			j.saved = u
			return result.Ok[domain.User, domain.Error](u)
		},
		PublishEvent: func(
			_ context.Context, eventType, aggregateID string, payload any,
		) result.Result[domain.Ack, domain.Error] {
			j.note("PublishEvent")
			j.eventType, j.aggregate, j.event = eventType, aggregateID, payload
			return result.Ok[domain.Ack, domain.Error](domain.Ack{})
		},
		GenerateID: func() domain.UserID { j.note("GenerateID"); return identifier },
		Now:        func() time.Time { j.note("Now"); return fixedInstant() },
	}
}

// failing fabricates a port in error, for any success type.
func failing[T any](code domain.ErrorCode, message string) result.Result[T, domain.Error] {
	return result.Err[T, domain.Error](domain.NewError(code, message))
}

// register runs the use case and returns its Result.
func register(deps application.Deps) result.Result[domain.User, domain.Error] {
	return application.NewRegisterUser(deps)(context.Background(), command())
}

// userOf extracts the user of a Result expected to be a success.
func userOf(t *testing.T, r result.Result[domain.User, domain.Error]) domain.User {
	t.Helper()
	user, err, ok := r.Get()
	if !ok {
		t.Fatalf("registration refused although it should have succeeded: %v", err)
	}
	return user
}

// codeOf extracts the error code of a Result expected to be a failure.
func codeOf(t *testing.T, r result.Result[domain.User, domain.Error]) domain.ErrorCode {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("a failure was expected, got a success")
	}
	return err.Code
}
