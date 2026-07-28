package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestExpiredSessionStopsVerifying vérifie l'expiration sans attendre.
//
// # Pourquoi le cas d'usage revérifie ce que le magasin sait déjà
//
// Le pilote en mémoire ne purge PAS : une session expirée y reste jusqu'au
// redémarrage. Si `Verify` se contentait de la trouver, elle continuerait de
// valoir indéfiniment. Compter sur le pilote pour ne rendre que des sessions
// valides ferait dépendre la sécurité d'un détail d'implémentation — et le
// premier pilote qui purgerait autrement rouvrirait la faille sans que rien ne le
// signale.
//
// La borne est STRICTE : une session expire À sa date, pas après.
func TestExpiredSessionStopsVerifying(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, c := newModule(t, map[string]any{"session_ttl": "1h"})
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}

	c.Advance(59 * time.Minute)
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("la session n'a pas encore expiré : %v", err)
	}

	c.Advance(time.Minute)
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("session expirée : attendu ErrTokenUnknown, obtenu %v", err)
	}
}

// TestSessionTTLOptionIsHonoured constate que l'option est LUE.
//
// Une option de configuration silencieusement ignorée est le défaut qui a coûté
// l'issue #93 : le serveur démarrait, montait le pilote, et ne disait rien. Ici,
// une durée d'une seconde doit produire une session qui ne survit pas à une
// seconde — ce que la durée par défaut de douze heures ne ferait pas.
func TestSessionTTLOptionIsHonoured(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, c := newModule(t, map[string]any{"session_ttl": "1s"})
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}

	c.Advance(time.Second)
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("l'option session_ttl est ignorée : %v", err)
	}
}
