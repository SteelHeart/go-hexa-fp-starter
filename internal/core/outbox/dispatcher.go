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

// Default values of the dispatching policy.
//
// Chosen for an ordinary service, not for an extreme case: fifty messages per
// batch hold the locks for a short time, two seconds of polling give an
// acceptable latency without hammering the database, and eight attempts with
// exponential backoff cover roughly forty minutes of a consumer outage.
// Option keys read by the dispatcher.
//
// Declared here and REFERENCED by the catalogue, in the same package: a key
// that the catalogue would admit without anyone reading it, or the reverse,
// would become a divergence between two lists. Sharing the constant makes it
// impossible (ADR 014, #93).
const (
	OptionBatchSize   = "batch_size"
	OptionMaxAttempts = "max_attempts"
	OptionBaseBackoff = "base_backoff"
	OptionInterval    = "interval"
)

const (
	defaultBatchSize   = 50
	defaultMaxAttempts = 8
	defaultBaseBackoff = time.Second
	defaultInterval    = 2 * time.Second
)

// ErrHandlerRequired refuses a dispatcher without a publisher.
//
// A dispatcher without a publisher would claim messages to do nothing with
// them: they would look processed while nothing had been sent.
var ErrHandlerRequired = errors.New("the outbox dispatcher requires a publisher")

// ErrLoggerRequired refuses a dispatcher without a log.
//
// Orchestration does not log — it reports — but the report must end up
// somewhere. Without a destination, a definitively abandoned message would be
// abandoned in silence, and the lost event would only be discovered at the
// customer's end.
var ErrLoggerRequired = errors.New("the outbox dispatcher requires a logger")

// DispatcherDeps carries what the dispatcher needs beyond the module's ports.
type DispatcherDeps struct {
	// Handle publishes a message to the world: broker, webhook, email.
	Handle ports.Handler
	Logger *slog.Logger
	Now    func() time.Time
}

// NewDispatcher builds the dispatcher from the module's ports.
//
// The policy is tuned through `options` in config/modules.yaml:
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

// policyFrom reads the dispatching policy from the module's options.
func policyFrom(cfg config.Module) (application.Policy, error) {
	batchSize, err := cfg.IntOption(OptionBatchSize, defaultBatchSize)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	maxAttempts, err := cfg.IntOption(OptionMaxAttempts, defaultMaxAttempts)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	baseBackoff, err := cfg.DurationOption(OptionBaseBackoff, defaultBaseBackoff)
	if err != nil {
		return application.Policy{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	interval, err := cfg.DurationOption(OptionInterval, defaultInterval)
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

// LogReport builds a report that logs.
//
// Lives here, in the composition root, and not in `application/`: orchestration
// is not allowed to log, but someone has to. Exported so that a caller can
// replace it — with a metrics counter, for example.
//
// # The level carries a decision, not a habit
//
//   - `published` at DEBUG: this is the nominal case, thousands of times a day.
//   - `retry_scheduled` at WARN: a transient outage is expected, not abnormal.
//   - `exhausted` at ERROR: an event is definitively lost for its consumer.
//     This is the only case that requires human intervention.
//   - `resolve_failed` at ERROR: the message goes out a second time. Benign if
//     the consumers are idempotent — and revealing if they are not.
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
		logger.LogAttrs(ctx, levelOf(outcome.Event), "outbox dispatching", attrs...)
	}
}

// levelOf maps a log level to an event.
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
