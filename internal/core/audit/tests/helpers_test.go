// Package tests contient les tests en BOÎTE NOIRE du module audit : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// recordedAt est l'instant injecté. Aucun test n'appelle time.Now : un journal
// d'audit doit être reproductible à la seconde près.
func recordedAt() time.Time {
	return time.Date(2026, time.July, 25, 12, 30, 0, 0, time.UTC)
}

// newLogModule construit le module sur son pilote par défaut et rend le tampon
// où le journal est écrit, pour pouvoir l'inspecter.
func newLogModule(t *testing.T) (audit.Module, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mod, err := audit.New(
		config.Module{Enabled: true, Driver: "log"},
		audit.Deps{Logger: logger, Now: recordedAt},
	)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod, &buf
}

// completeEntry est une entrée valide, à altérer champ par champ dans les tests.
func completeEntry() domain.Entry {
	return domain.Entry{
		Actor:      "user-42",
		Action:     "user.registered",
		EntityType: "user",
		EntityID:   "42",
		Metadata:   map[string]any{"surface": "http"},
	}
}

// decodeJournal lit la ligne écrite par le pilote log.
func decodeJournal(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("journal illisible (%q): %v", buf.String(), err)
	}
	return line
}
