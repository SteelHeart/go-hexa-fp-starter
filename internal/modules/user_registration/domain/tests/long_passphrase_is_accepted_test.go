package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestLongPassphraseIsAccepted: an ordinary passphrase passes, without requiring
// an upper case letter, a digit or a special character.
//
// This is the counterpart of the high minimum, and it must be tested: a
// composition rule added "to look serious" would turn this test red, and that is
// exactly the signal one wants — the decision was taken, it is not undone by
// inadvertence.
func TestLongPassphraseIsAccepted(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"correct horse battery staple",
		"i really like apple pies a lot",
		"twelveCHARS1",
	} {
		if _, _, ok := domain.NewRawPassword(raw).Get(); !ok {
			t.Errorf("NewRawPassword(%q) should have been accepted", raw)
		}
	}
}
