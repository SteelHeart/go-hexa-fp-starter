//go:build integration

package integration

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
)

// TestOutboxAcceptsAMessageWithoutHeaders est le témoin d'un défaut RÉEL,
// trouvé par ce niveau de test à sa toute première exécution.
//
// Le pilote passait `msg.Headers` explicitement à un `INSERT`. Une carte nil
// arrive en NULL, la colonne est `NOT NULL`, et le `DEFAULT '{}'::jsonb` de la
// migration ne s'applique pas — un DEFAULT ne joue que si la colonne est OMISE.
//
// Portée exacte du défaut : `outboxpub`, l'adaptateur réel du module de
// référence, ne pose AUCUN en-tête. Donc **`POST /v1/users` aurait échoué en
// production dès que `modules.outbox.driver` vaut `postgres`.**
//
// Pourquoi 285 tests unitaires ne l'ont pas vu : ils tournent tous sur le pilote
// `memory`, qui accepte nil sans broncher. Les deux pilotes ne respectaient pas
// le même contrat, et rien ne pouvait le dire — c'est très exactement ce que le
// niveau `integration` existe pour attraper (#37).
//
// ⚠️ Ne pas « simplifier » ce test au motif qu'il ressemble à un Enqueue banal.
// C'est l'absence d'en-têtes qui est éprouvée, et elle seule.
func TestOutboxAcceptsAMessageWithoutHeaders(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	eventType := unique(t, "integration.sans-entetes")
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t),
			"DELETE FROM platform.outbox_messages WHERE event_type = $1", eventType)
	})

	id, err := store.Enqueue(ctx, domain.NewMessage{
		Type:        eventType,
		AggregateID: "agg-sans-entetes",
		Payload:     []byte(`{"k":"v"}`),
		// Headers et TraceParent volontairement absents : c'est la forme que
		// produit `outboxpub`, donc la forme du chemin réel.
	})
	if err != nil {
		t.Fatalf("un message sans en-têtes doit être accepté, comme sur le pilote memory: %v", err)
	}
	if id == "" {
		t.Fatal("identifiant vide")
	}

	// L'en-tête stocké doit être une carte VIDE, jamais NULL : un consommateur
	// qui lit `headers` ne doit pas avoir à distinguer deux formes de « rien ».
	var headers map[string]string
	if err := p.QueryRow(ctx,
		"SELECT headers FROM platform.outbox_messages WHERE id = $1", string(id),
	).Scan(&headers); err != nil {
		t.Fatalf("relecture des en-têtes: %v", err)
	}
	if headers == nil {
		t.Error("en-têtes relus à NULL : la normalisation n'a pas eu lieu")
	}
	if len(headers) != 0 {
		t.Errorf("en-têtes relus = %v, attendu une carte vide", headers)
	}
}
