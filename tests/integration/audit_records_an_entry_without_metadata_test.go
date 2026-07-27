//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	pgaudit "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/postgres"
)

// TestAuditRecordsAnEntryWithoutMetadata éprouve le pilote d'audit contre la
// vraie table, sans métadonnées.
//
// La question posée est la même que celle qui a fait tomber le pilote d'outbox :
// que devient une carte nil face à une colonne `jsonb NOT NULL` ? Ici la réponse
// diffère — `json.Marshal(nil)` produit le littéral JSON `null`, qui est une
// valeur jsonb VALIDE, donc la contrainte passe.
//
// Elle passe, mais elle stocke `null` là où la colonne annonce `DEFAULT '{}'`.
// Un consommateur devrait donc distinguer deux formes de « rien ». Ce test fixe
// la forme attendue au lieu de la laisser dépendre d'un détail d'encodage.
//
// L'horodatage vient d'un port injecté : c'est ce qui garde le module testable
// sans attendre, et ce qui rend cette vérification possible du tout.
func TestAuditRecordsAnEntryWithoutMetadata(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	instant := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	record := pgaudit.New(p, func() time.Time { return instant })

	actor := unique(t, "integration-audit")
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.audit_log WHERE actor = $1", actor)
	})

	if err := record(ctx, domain.Entry{
		Actor:      actor,
		Action:     "integration.executed",
		EntityType: "run",
		EntityID:   "1",
		// Metadata volontairement absente.
	}); err != nil {
		t.Fatalf("une entrée sans métadonnées doit être acceptée: %v", err)
	}

	var (
		metadata   []byte
		occurredAt time.Time
	)
	if err := p.QueryRow(ctx,
		"SELECT metadata, occurred_at FROM platform.audit_log WHERE actor = $1", actor,
	).Scan(&metadata, &occurredAt); err != nil {
		t.Fatalf("relecture de l'entrée: %v", err)
	}

	if got := string(metadata); got != "{}" {
		t.Errorf("métadonnées stockées = %q, attendu \"{}\" — un consommateur devrait "+
			"sinon distinguer deux formes de « aucune métadonnée »", got)
	}
	if !occurredAt.UTC().Equal(instant) {
		t.Errorf("horodatage = %v, attendu %v : l'instant doit venir du port injecté, "+
			"jamais de la base", occurredAt.UTC(), instant)
	}
}
