package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestPublishFailureFailsTheWholeRegistration est l'autre moitié du patron outbox,
// et la plus contre-intuitive.
//
// L'utilisateur a été écrit avec succès. Pourtant, si l'enregistrement de
// l'événement échoue, TOUTE l'inscription échoue.
//
// La tentation serait de rendre un succès en journalisant l'incident : après tout,
// le compte existe. Mais l'écriture et l'événement sont dans la MÊME transaction :
// l'échec déclenche le rollback, donc le compte n'existe déjà plus. Rendre un
// succès mentirait à l'appelant sur un compte qui vient de disparaître.
//
// Un utilisateur créé sans son événement de bienvenue est un état incohérent
// silencieux — et le silence est le pire des défauts.
func TestPublishFailureFailsTheWholeRegistration(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	deps := depsNominales(observe)
	deps.PublishEvent = func(
		context.Context, string, string, any,
	) result.Result[domain.Ack, domain.Error] {
		observe.note("PublishEvent")
		return echoue[domain.Ack](domain.CodeUnavailable, "outbox injoignable")
	}

	if got := codeDe(t, inscrit(deps)); got != domain.CodeUnavailable {
		t.Errorf("code = %q, attendu %q", got, domain.CodeUnavailable)
	}
	if !observe.aAppele("SaveUser") {
		t.Error("l'écriture doit bien avoir été tentée avant la publication")
	}
}
