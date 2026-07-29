package scheduler

import (
	"context"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// LogReport builds a report that logs.
//
// Lives here, in the composition root, and not in `application/`: the
// orchestration is not allowed to log (`rules/README.md` § "the core is pure"),
// but somebody has to. Exported so that a caller can replace it — by a metrics
// counter, for instance — without touching the module.
//
// # The log level carries a decision, not a habit
//
//   - `skipped` at DEBUG: it is the nominal case on N-1 replicas, on every tick.
//     At info, the logs would contain nothing but that and the real incident
//     would be invisible in them.
//   - `election_failed` at WARN: the task did not run, but the fallback is safe.
//   - `failed` at ERROR: the work failed, somebody has to look.
//   - `release_failed` at ERROR: the task will NOT run again until the lock
//     expires. That is more serious than a one-off failure, despite appearances.
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
		logger.LogAttrs(ctx, levelOf(outcome.Event), "scheduled task", attrs...)
	}
}

// levelOf maps a log level to an event.
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
