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

// Câblage de l'observabilité du dépileur — #13.
//
// Le pendant de `cmd/server/observabilite.go`, et il est nécessaire pour une
// raison propre au dépileur : c'est le seul processus dont PERSONNE ne constate
// l'inactivité. Un serveur muet se voit à la première requête ; un dépileur qui
// ne dépile plus ne se voit qu'au moment où l'on cherche un événement qui n'est
// jamais parti.
//
// Deux composition roots, deux fichiers : c'est la contrepartie assumée de
// l'ADR 004. Le seul endroit qui connaît tout est celui-ci, et le partager
// reviendrait à fabriquer un mini-conteneur d'injection.

const graceTelemetrie = 5 * time.Second

// demarrerObservabilite installe les fournisseurs et rend la fonction d'arrêt.
//
// Désactivée — le défaut livré —, `Setup` rend un arrêt inerte : le dépileur
// démarre donc toujours sans collecteur.
func demarrerObservabilite(ctx context.Context, cfg config.Config) (telemetry.Shutdown, error) {
	arret, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("télémétrie: %w", err)
	}
	return arret, nil
}

// arreterObservabilite vide les exportateurs avant de rendre la main.
//
// Contexte DÉTACHÉ de l'annulation : à l'arrêt, `ctx` est déjà annulé, et le
// passer tel quel jetterait les spans en tampon — ceux de la dernière minute.
func arreterObservabilite(ctx context.Context, arret telemetry.Shutdown, logger *slog.Logger) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), graceTelemetrie)
	defer cancel()

	if err := arret(stopCtx); err != nil {
		logger.ErrorContext(stopCtx, "arrêt de la télémétrie", slog.String("erreur", err.Error()))
	}
}

// servirMetriques expose /metrics, fatal hors développement.
//
// ⚠️ `telemetry.metrics_port` est UNE valeur, partagée par les deux binaires.
// Sur un même hôte, avec la télémétrie activée — ce que fait
// `config/env/development.yaml` —, le second à démarrer ne peut pas se lier. En
// développement il continue donc, en le disant ; ailleurs il refuse.
//
// C'est le même arbitrage que dans `cmd/server`, et il est justifié en détail
// là-bas. Le dépileur y ajoute une raison propre : c'est le seul processus dont
// personne ne constate l'inactivité, donc celui dont les métriques comptent le
// plus — et celui pour lequel un refus de démarrer en local coûterait le plus
// cher en friction.
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
			logger.WarnContext(ctx, "métriques indisponibles — le dépileur continue (développement)", champs...)
			return
		}
		logger.ErrorContext(ctx, "serveur de métriques — arrêt du dépileur", champs...)
		couper()
	}()
}
