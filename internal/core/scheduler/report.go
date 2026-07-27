package scheduler

import (
	"context"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// LogReport construit un compte rendu qui journalise.
//
// Vit ici, dans le composition root, et non dans `application/` : l'orchestration
// n'a pas le droit de journaliser (`rules/README.md` § « le cœur est pur »), mais
// quelqu'un doit bien le faire. Exportée pour qu'un appelant la remplace — par un
// compteur de métriques, par exemple — sans toucher au module.
//
// # Le niveau de journal porte une décision, pas une habitude
//
//   - `skipped` en DEBUG : c'est le cas nominal sur N-1 répliques, à chaque tick.
//     En info, les journaux ne contiendraient plus que ça et l'incident réel y
//     serait invisible.
//   - `election_failed` en WARN : la tâche n'a pas tourné, mais le repli est sûr.
//   - `failed` en ERROR : le travail a échoué, quelqu'un doit regarder.
//   - `release_failed` en ERROR : la tâche ne tournera PLUS jusqu'à péremption du
//     verrou. C'est plus grave qu'un échec ponctuel, malgré les apparences.
func LogReport(logger *slog.Logger) ports.Report {
	return func(ctx context.Context, outcome domain.Outcome) {
		attrs := []slog.Attr{
			slog.String("task", outcome.Task.String()),
			slog.String("event", string(outcome.Event)),
		}
		if outcome.Duration > 0 {
			attrs = append(attrs, slog.Duration("duration", outcome.Duration))
		}
		if outcome.Err != nil {
			attrs = append(attrs, slog.String("error", outcome.Err.Error()))
		}
		logger.LogAttrs(ctx, levelOf(outcome.Event), "tâche planifiée", attrs...)
	}
}

// levelOf associe un niveau de journal à un événement.
func levelOf(event domain.Event) slog.Level {
	switch event {
	case domain.EventFailed, domain.EventReleaseFailed:
		return slog.LevelError
	case domain.EventElectionFailed:
		return slog.LevelWarn
	case domain.EventSkipped, domain.EventSucceeded:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
