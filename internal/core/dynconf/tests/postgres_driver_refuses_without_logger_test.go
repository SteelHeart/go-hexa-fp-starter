package tests

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestPostgresDriverRefusesWithoutLogger : ce pilote ne PEUT pas retourner ses
// pannes — le contrat de ports.IsEnabled l'interdit. Sans journal, une base
// injoignable éteindrait les fonctionnalités masquées sans laisser de trace. Le
// journal n'est donc pas un confort, c'est la contrepartie du contrat.
//
// Le pool passé est une coquille vide et jamais utilisée : la fabrique ne teste
// que sa présence, et vérifier un refus n'exige pas une base.
func TestPostgresDriverRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(
		config.Module{Enabled: true, Driver: "postgres"},
		dynconf.Deps{Pool: &pgxpool.Pool{}},
	)
	if !errors.Is(err, dynconf.ErrLoggerRequired) {
		t.Errorf("erreur = %v, attendu ErrLoggerRequired", err)
	}
}
