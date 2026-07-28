package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
)

// ─────────────────────────────────────────────────────────────────────────────
// Ce fichier est la moitié qui manquait — #13
// ─────────────────────────────────────────────────────────────────────────────
//
// `telemetry.Setup` et `httpserver.NewMetricsServer` existaient, étaient testés,
// et n'étaient appelés par AUCUN binaire. Conséquence mesurée avant correction :
//
//	occurrences de trace_id dans les journaux : 0
//	GET /metrics                              : aucune réponse
//
// `otelhttp` enveloppait pourtant déjà le routeur, et le gestionnaire slog
// injectait déjà `trace_id` — mais sans fournisseur de traces installé, le span
// ouvert par `otelhttp` n'est pas enregistré, son contexte est invalide, et
// l'injection ne se déclenche jamais. Trois pièces sur quatre, donc rien.
//
// C'est le mode de défaillance que `documentation/produit/personas.md` désigne
// comme le pire pour l'exploitant : *une configuration d'observabilité sans
// câblage est pire qu'aucune, elle fait croire que la question est traitée.*

// grâceTelemetrie borne l'arrêt des exportateurs.
//
// Un délai fini plutôt qu'aucun : un collecteur injoignable ferait autrement
// attendre l'arrêt indéfiniment, c'est-à-dire transformerait un incident
// d'observabilité en incident de disponibilité.
const graceTelemetrie = 5 * time.Second

// demarrerObservabilite installe les fournisseurs et rend la fonction d'arrêt.
//
// Désactivée — le défaut livré —, `Setup` rend un arrêt inerte : ce câblage
// n'ajoute donc AUCUN prérequis d'infrastructure. `go run ./cmd/server` démarre
// toujours sans collecteur, sans base et sans Docker (ADR 012).
func demarrerObservabilite(ctx context.Context, cfg config.Config) (telemetry.Shutdown, error) {
	arret, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("télémétrie: %w", err)
	}
	return arret, nil
}

// arreterObservabilite vide les exportateurs avant de rendre la main.
//
// Le contexte est délibérément DÉTACHÉ de l'annulation : à l'arrêt, `ctx` est
// déjà annulé, et le passer tel quel ferait jeter les spans en tampon — ceux de
// la dernière minute, c'est-à-dire précisément ceux qu'on cherchera après un
// arrêt anormal.
func arreterObservabilite(ctx context.Context, arret telemetry.Shutdown, logger *slog.Logger) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), graceTelemetrie)
	defer cancel()

	if err := arret(stopCtx); err != nil {
		logger.ErrorContext(stopCtx, "arrêt de la télémétrie", slog.String("erreur", err.Error()))
	}
}

// servirMetriques expose /metrics, et décide quoi faire quand le port est pris.
//
// # L'échec est fatal HORS développement, et c'est un arbitrage mesuré
//
// En production, un service qui tourne sans métriques est exactement ce que
// personne ne remarque avant l'incident où l'on en a besoin. Le refus doit donc
// être total.
//
// En développement, non — et c'est une correction, pas une tolérance. La version
// précédente coupait partout. Or `config/env/development.yaml` active la
// télémétrie : le port 9090 est donc ouvert par DÉFAUT en local, et 9090 est
// occupé sur beaucoup de postes. « `go run` démarre sur une machine vierge »
// aurait cessé d'être vrai à cause d'un port de diagnostic. Mesuré en le
// branchant, pas anticipé.
//
// La forme suit celle déjà retenue ailleurs dans le socle — `SecurityHeaders()`
// contre `SecurityHeadersWithoutHSTS()` : le comportement strict est le défaut,
// et la renonciation est NOMMÉE et bornée. Ici elle est bornée à
// `development`/`test`, et elle est bruyante.
func servirMetriques(ctx context.Context, cfg config.Config, logger *slog.Logger, couper context.CancelFunc) {
	if !cfg.Telemetry.Enabled {
		return
	}
	serveur := httpserver.NewMetricsServer(cfg.Telemetry.MetricsPort, logger)

	go func() {
		err := serveur.Run(ctx)
		if err == nil {
			return
		}
		champs := []any{
			slog.Int("port", cfg.Telemetry.MetricsPort),
			slog.String("erreur", err.Error()),
		}
		if cfg.App.Env.IsLocal() {
			logger.WarnContext(ctx, "métriques indisponibles — le service continue (développement)", champs...)
			return
		}
		logger.ErrorContext(ctx, "serveur de métriques — arrêt du service", champs...)
		couper()
	}()
}
