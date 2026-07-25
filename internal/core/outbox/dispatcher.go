package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Valeurs par défaut de la politique de dépilage.
//
// Choisies pour un service ordinaire, pas pour un cas extrême : cinquante
// messages par lot tiennent les verrous peu de temps, deux secondes de scrutation
// donnent une latence acceptable sans marteler la base, et huit tentatives à
// recul exponentiel couvrent environ quarante minutes de panne d'un consommateur.
const (
	defaultBatchSize   = 50
	defaultMaxAttempts = 8
	defaultBaseBackoff = time.Second
	defaultInterval    = 2 * time.Second
)

// ErrHandlerRequired refuse un dépileur sans publieur.
//
// Un dépileur sans publieur réserverait des messages pour ne rien en faire : ils
// paraîtraient traités alors que rien ne serait parti.
var ErrHandlerRequired = errors.New("le dépileur de l'outbox exige un publieur")

// ErrLoggerRequired refuse un dépileur sans journal.
//
// L'orchestration ne journalise pas — elle rend compte — mais le compte rendu doit
// aboutir quelque part. Sans destination, un message définitivement abandonné le
// serait en silence, et l'événement perdu ne se découvrirait que chez le client.
var ErrLoggerRequired = errors.New("le dépileur de l'outbox exige un journal")

// DispatcherDeps porte ce dont le dépileur a besoin en plus des ports du module.
type DispatcherDeps struct {
	// Handle publie un message vers le monde : courtier, webhook, courriel.
	Handle ports.Handler
	Logger *slog.Logger
	Now    func() time.Time
}

// NewDispatcher construit le dépileur à partir des ports du module.
//
// La politique se règle par `options` dans config/modules.yaml :
//
//	outbox:
//	  enabled: true
//	  driver: postgres
//	  options:
//	    batch_size: 50
//	    max_attempts: 8
//	    base_backoff: 1s
//	    interval: 2s
func NewDispatcher(mod Module, cfg config.Module, deps DispatcherDeps) (*application.Dispatcher, error) {
	if deps.Handle == nil {
		return nil, ErrHandlerRequired
	}
	if deps.Logger == nil {
		return nil, ErrLoggerRequired
	}

	policy, err := policyFrom(cfg)
	if err != nil {
		return nil, err
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}
	dispatcher, err := application.NewDispatcher(application.Ports{
		Claim:      mod.Claim,
		MarkDone:   mod.MarkDone,
		MarkFailed: mod.MarkFailed,
		Handle:     deps.Handle,
		Report:     LogReport(deps.Logger),
		Now:        now,
	}, policy)
	if err != nil {
		return nil, fmt.Errorf("modules.%s: %w", Name, err)
	}
	return dispatcher, nil
}

// policyFrom lit la politique de dépilage dans les options du module.
func policyFrom(cfg config.Module) (application.Policy, error) {
	batchSize, err := cfg.IntOption("batch_size", defaultBatchSize)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	maxAttempts, err := cfg.IntOption("max_attempts", defaultMaxAttempts)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	baseBackoff, err := cfg.DurationOption("base_backoff", defaultBaseBackoff)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	interval, err := cfg.DurationOption("interval", defaultInterval)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}

	return application.Policy{
		BatchSize: batchSize,
		Interval:  interval,
		Retry: domain.RetryPolicy{
			MaxAttempts: maxAttempts,
			BaseBackoff: baseBackoff,
		},
	}, nil
}

// LogReport construit un compte rendu qui journalise.
//
// Vit ici, dans le composition root, et non dans `application/` : l'orchestration
// n'a pas le droit de journaliser, mais quelqu'un doit le faire. Exportée pour
// qu'un appelant la remplace — par un compteur de métriques, par exemple.
//
// # Le niveau porte une décision, pas une habitude
//
//   - `published` en DEBUG : c'est le cas nominal, des milliers de fois par jour.
//   - `retry_scheduled` en WARN : une panne passagère est attendue, pas anormale.
//   - `exhausted` en ERROR : un événement est définitivement perdu pour son
//     consommateur. C'est le seul cas qui exige une intervention humaine.
//   - `resolve_failed` en ERROR : le message part une seconde fois. Bénin si les
//     consommateurs sont idempotents — et révélateur s'ils ne le sont pas.
func LogReport(logger *slog.Logger) ports.Report {
	return func(ctx context.Context, outcome domain.Outcome) {
		attrs := []slog.Attr{
			slog.String("event", string(outcome.Event)),
			slog.String("message_id", outcome.ID.String()),
			slog.String("message_type", outcome.Type),
			slog.Int("attempts", outcome.Attempts),
		}
		if outcome.Duration > 0 {
			attrs = append(attrs, slog.Duration("duration", outcome.Duration))
		}
		if outcome.Err != nil {
			attrs = append(attrs, slog.String("error", outcome.Err.Error()))
		}
		logger.LogAttrs(ctx, levelOf(outcome.Event), "dépilage de l'outbox", attrs...)
	}
}

// levelOf associe un niveau de journal à un événement.
func levelOf(event domain.Event) slog.Level {
	switch event {
	case domain.EventExhausted, domain.EventResolveFailed, domain.EventHandlerPanicked:
		return slog.LevelError
	case domain.EventRetryScheduled, domain.EventClaimFailed:
		return slog.LevelWarn
	case domain.EventPublished:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
