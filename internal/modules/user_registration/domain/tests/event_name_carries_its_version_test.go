package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEventNameCarriesItsVersion: the name of an event carries its version.
//
// A consumer is deployed INDEPENDENTLY of the producer. Without a version in the
// name, changing the shape of the payload silently breaks every consumer still
// in flight — and the producer, for its part, sees no error at all.
//
// With the version, v2 appears alongside v1: consumers migrate at their own
// pace, and v1 is withdrawn once nobody listens to it any more.
func TestEventNameCarriesItsVersion(t *testing.T) {
	t.Parallel()

	if !strings.HasSuffix(domain.EventUserRegistered, ".v1") {
		t.Errorf("name = %q: an event without a version is a silent break waiting to happen",
			domain.EventUserRegistered)
	}
}
