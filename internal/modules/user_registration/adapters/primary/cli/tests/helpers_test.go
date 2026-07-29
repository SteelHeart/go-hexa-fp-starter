// Package tests exercises the COMMAND LINE SURFACE of the registration module.
//
// # What they verify
//
// The TRANSLATION, and nothing else: from arguments to a domain command, then
// from an outcome to output and an EXIT CODE. The domain and the pipeline have
// their own tests.
//
// The exit code matters as much as the text: a command line binary is called by
// scripts more often than by humans, and a script reads the code to decide
// whether to retry.
package tests

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	usercli "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/cli"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

const (
	// email and secret satisfy the bounds of the domain.
	email  = "alice@example.com"
	secret = "correct horse battery staple"
)

// streams retains what the command wrote, separately.
//
// The two streams are distinct DELIBERATELY: a binary that writes its errors on
// standard output breaks every script that redirects one without the other, and
// it is the kind of defect no test sees if it merges the two buffers.
type streams struct {
	out *bytes.Buffer
	err *bytes.Buffer
}

// newCommand mounts the surface on the module's default driver.
//
// Every test therefore has its own store, pristine, and can run in parallel
// without interfering.
func newCommand(t *testing.T, input string) (usercli.Command, *streams) {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: noopPublish,
		GenerateID:   func() domain.UserID { return "019f9b46-3aec-735a-977d-129192ef130f" },
		Now:          func() time.Time { return time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("mounting the module: %v", err)
	}

	captured := &streams{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	return usercli.Command{
		Module: mod,
		In:     strings.NewReader(input),
		Out:    captured.out,
		Err:    captured.err,
	}, captured
}

// fakeHash is a DUMMY hash, instantaneous.
//
// Argon2id is deliberately slow: paying for it here would make every test last
// tens of milliseconds without exercising anything more. What is tested is the
// translation, not the soundness of the digest.
func fakeHash(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("digest:" + password.String()))
}

// noopPublish accepts the event without doing anything with it.
func noopPublish(
	_ context.Context, _, _ string, _ any,
) result.Result[domain.Ack, domain.Error] {
	return result.Ok[domain.Ack, domain.Error](domain.Ack{})
}
