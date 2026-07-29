package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestInvalidInputTouchesNoPort: an invalid command reaches NO effect at all.
//
// Validation is the first step of the pipeline, and `result.Chain` short
// circuits on the first failure. Concrete consequence: a faulty input consumes
// neither a database connection nor a hashing cycle. A form filled in badly over
// and over therefore costs the server almost nothing.
//
// It is also what guarantees that no invalid data crosses the boundary: the
// following steps only receive domain types.
func TestInvalidInputTouchesNoPort(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cmd  domain.RegistrationCommand
		code domain.ErrorCode
	}{
		"invalid address": {
			cmd:  domain.RegistrationCommand{Email: "not an address", Password: strongPassword},
			code: domain.CodeInvalidEmail,
		},
		"weak password": {
			cmd:  domain.RegistrationCommand{Email: validAddress, Password: "short"},
			code: domain.CodeWeakPassword,
		},
		"empty command": {
			cmd:  domain.RegistrationCommand{},
			code: domain.CodeInvalidEmail,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			observed := &callLog{}
			registerUser := application.NewRegisterUser(nominalDeps(observed))

			_, err, ok := registerUser(context.Background(), tc.cmd).Get()
			if ok {
				t.Fatal("an invalid command must be refused")
			}
			if err.Code != tc.code {
				t.Errorf("code = %q, want %q", err.Code, tc.code)
			}
			if len(observed.calls) != 0 {
				t.Errorf("ports called = %v, want none", observed.calls)
			}
		})
	}
}
