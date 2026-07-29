package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestPasswordNeverAppearsInALog is the most important security test of the
// domain.
//
// A clear-text password in a log is a definitive leak: logs are kept for a long
// time, duplicated towards aggregators, and read by humans. Rotating it is not
// enough — every user has to be warned.
//
// The protection is structural: `String()` returns a marker, so `%v` and `%s`
// mask the value. This test also covers the case that really causes leaks — a
// `%+v` on a STRUCTURE that contains the password, written without a second
// thought into a debug log.
func TestPasswordNeverAppearsInALog(t *testing.T) {
	t.Parallel()

	const secret = "correct horse battery staple"
	value, _, ok := domain.NewRawPassword(secret).Get()
	if !ok {
		t.Fatal("the test password should have been accepted")
	}

	// Going through fmt is DELIBERATE, and gocritic is wrong to suggest
	// `value.String()`: it is precisely the `fmt` path that is being tested. A
	// developer who leaks a password never does it by calling String() — they
	// do it by passing the value to a `%v` in a log. Calling String() here would
	// test the method instead of the leak path, and the test would stay green
	// the day someone removes the Stringer implementation.
	formats := map[string]string{
		"%v":  fmt.Sprintf("%v", value), //nolint:gocritic // the fmt path IS the subject of the test, not a detour
		"%s":  fmt.Sprintf("%s", value), //nolint:gocritic,staticcheck // same: String() would bypass the tested leak
		"%+v": fmt.Sprintf("%+v", struct{ Password domain.RawPassword }{value}),
		"%v struct": fmt.Sprintf("%v", struct {
			Email    string
			Password domain.RawPassword
		}{"alice@example.com", value}),
	}

	for format, rendered := range formats {
		if strings.Contains(rendered, secret) {
			t.Errorf("%s leaked the password: %q", format, rendered)
		}
	}

	// Expose remains the only path to the value, and it is named so as to be
	// seen in review: there must be a single call in the whole repository, the
	// one made by the hashing port.
	if value.Expose() != secret {
		t.Error("Expose must return the real value: that is its only reason to exist")
	}
}
