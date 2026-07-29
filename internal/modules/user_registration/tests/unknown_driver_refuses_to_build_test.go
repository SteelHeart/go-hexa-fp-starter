package tests

import (
	"strings"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// TestUnknownDriverRefusesToBuild: an unknown driver refuses to be assembled.
//
// Deny by default, and what is at stake is concrete: falling back on the default
// driver would be MUCH WORSE than the error. A typo in the production
// configuration — `postgersql` instead of `postgres` — would start the service
// IN MEMORY, without reporting anything. The service would answer normally, and
// the loss of every registration would only be seen at the first restart.
func TestUnknownDriverRefusesToBuild(t *testing.T) {
	t.Parallel()

	publisher := &spyPublisher{}
	mod, err := userregistration.New("postgersql", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err == nil {
		t.Fatal("an unknown driver must refuse to be assembled")
	}

	// The returned module must be inert: if New fails but still returns a
	// callable module, a caller who ignores the error would believe it works.
	if mod.Register != nil {
		t.Error("a refused assembly must return no callable use case")
	}
	if !strings.Contains(err.Error(), "postgersql") {
		t.Errorf("the error must name the faulty driver, got: %v", err)
	}
}
