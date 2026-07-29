package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// These tests target NON-EXPORTED identifiers — `newLoggerTo`, `parseLevel`,
// `traceHandler` — hence an `internal_test.go` (rules/tests.md §2). The log
// writes into a buffer, never on the standard output: a test that polluted the
// output would make the others unreadable.

func loggerConfig(env config.Environment, format, level string) config.Config {
	var cfg config.Config
	cfg.App.Env = env
	cfg.App.Name = "hexa-tests"
	cfg.App.Version = "1.2.3"
	cfg.Telemetry.LogFormat = format
	cfg.Telemetry.LogLevel = level
	return cfg
}

// TestNonDevelopmentAlwaysLogsJSON: outside development, the format is imposed.
//
// # The defect this test catches
//
// The condition is `format == "json" || !env.IsDevelopment()`. The second term
// is what counts: it IMPOSES JSON outside development, even if the
// configuration asks for text.
//
// Without it, a `log_format: text` copied over from a development file towards
// production would produce logs that the collector does not know how to
// deserialise. They would keep being emitted, the supervision would stay green
// — and on the day of the incident, nothing is queryable any more. It is only
// noticed at the moment it is needed.
func TestNonDevelopmentAlwaysLogsJSON(t *testing.T) {
	t.Parallel()

	for _, env := range []config.Environment{
		config.EnvTest, config.EnvUAT, config.EnvProduction, "unknown",
	} {
		var buf bytes.Buffer
		// The configuration explicitly asks for TEXT: it must be ignored.
		newLoggerTo(loggerConfig(env, "text", "info"), &buf).Info("message")

		var decoded map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
			t.Errorf("env=%q, format=text: the output is not JSON (%v): %s",
				env, err, buf.String())
		}
	}
}

// TestDevelopmentLogsTextUnlessJSONIsAsked: locally, the human first.
func TestDevelopmentLogsTextUnlessJSONIsAsked(t *testing.T) {
	t.Parallel()

	var text bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvDevelopment, "", "info"), &text).Info("message")
	if json.Valid(bytes.TrimSpace(text.Bytes())) {
		t.Errorf("development without a requested format produces JSON: %s", text.String())
	}

	var asJSON bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvDevelopment, "json", "info"), &asJSON).Info("message")
	if !json.Valid(bytes.TrimSpace(asJSON.Bytes())) {
		t.Errorf("development with format=json does not produce JSON: %s", asJSON.String())
	}
}

// TestEveryRecordCarriesTheServiceIdentity: service, env and version, always.
//
// An aggregated log mixes services and versions. Without these three
// attributes, an error line says neither WHO emitted it nor WHICH version —
// hence one can neither correlate it to a deployment, nor rule out the healthy
// replicas.
func TestEveryRecordCarriesTheServiceIdentity(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &buf).Info("message")

	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
		t.Fatalf("unreadable output: %v", err)
	}
	for field, want := range map[string]string{
		"service": "hexa-tests",
		"env":     string(config.EnvProduction),
		"version": "1.2.3",
	} {
		if got, _ := decoded[field].(string); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
}

// TestAnUnknownLevelFallsBackToInfo: an unreadable level does not cut the log.
//
// # Why the fallback goes towards Info and not towards Error
//
// It is the only fallback of the repository that is NOT a refusal, and it is
// justified: falling back to Error would make a service mute on a typo
// (`log_level: infoo`), and a mute service cannot be diagnosed. Falling back to
// Debug would flood production.
//
// Info is the only choice that keeps the log usable in both directions.
func TestAnUnknownLevelFallsBackToInfo(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"WARNING": slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"infoo":   slog.LevelInfo,
		"trace":   slog.LevelInfo,
	} {
		if got := parseLevel(raw); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

// TestTheSourceIsAttachedOnlyAtDebug: the code location follows the level.
//
// # What this test locks down, in both directions
//
// `AddSource: level == slog.LevelDebug` links two settings that have no obvious
// reason to be linked. Both directions count:
//
//   - Always ACTIVE, it would make every production line pay for the capture of
//     a call trace — the cost of `runtime.Callers`, on the hottest path of the
//     service — and would publish the source tree in the logs.
//   - Never active, debug mode would lose what makes it useful: knowing WHICH
//     line emitted the message. Debug is enabled to understand, not to get more
//     text.
//
// This coupling is written nowhere else but here.
func TestTheSourceIsAttachedOnlyAtDebug(t *testing.T) {
	t.Parallel()

	var atDebug bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "debug"), &atDebug).Debug("message")
	if !strings.Contains(atDebug.String(), "source") {
		t.Errorf("debug level WITHOUT a source location: %s — "+
			"debug mode loses what makes it useful", atDebug.String())
	}

	var atInfo bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &atInfo).Info("message")
	if strings.Contains(atInfo.String(), "source") {
		t.Errorf("info level WITH a source location: %s — "+
			"every production line would pay for a stack capture", atInfo.String())
	}
}

// TestATraceIsAttachedToEveryRecordThatHasOne: the trace_id is injected by the
// handler, not by the callers.
//
// # Why this is worth a test
//
// "A log without a trace_id is a log that cannot be cross-checked during an
// incident." Relying on the callers to add it guarantees that it will be
// missing exactly where it serves — in the error path nobody has reread.
//
// The handler therefore reads it from the context. This test checks BOTH
// directions: present when there is a span, AND ABSENT when there is none. The
// second counts just as much: writing
// `trace_id: "00000000000000000000000000000000"` on every line outside a
// request pollutes the indexes and makes unrelated lines match.
func TestATraceIsAttachedToEveryRecordThatHasOne(t *testing.T) {
	t.Parallel()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("invalid test trace identifier: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("invalid test span identifier: %v", err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
	))

	var withTrace bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &withTrace).
		InfoContext(ctx, "inside a request")

	emitted := withTrace.String()
	if !strings.Contains(emitted, "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("trace_id missing: %s", emitted)
	}
	if !strings.Contains(emitted, "00f067aa0ba902b7") {
		t.Errorf("span_id missing: %s", emitted)
	}

	var withoutTrace bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &withoutTrace).
		InfoContext(context.Background(), "outside a request")

	if strings.Contains(withoutTrace.String(), "trace_id") {
		t.Errorf("trace_id written without a span: %s — "+
			"a null identifier would make unrelated lines match", withoutTrace.String())
	}
}

// TestTheTraceHandlerSurvivesAttrsAndGroups: decoration does not lose the span.
//
// # The defect this test catches
//
// `WithAttrs` and `WithGroup` must REWRAP. Returning `h.inner.WithAttrs(...)`
// directly — the natural oversight — would remove the `traceHandler` from the
// chain, and the trace_id would disappear from every decorated log.
//
// Now `NewLogger` itself calls `.With(service, env, version)`: the defect would
// therefore touch the WHOLE of the repository's logs, immediately, without
// anything signalling it — the line would simply be a bit shorter.
func TestTheTraceHandlerSurvivesAttrsAndGroups(t *testing.T) {
	t.Parallel()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("invalid test trace identifier: %v", err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		},
	))

	var buf bytes.Buffer
	// .With() then .WithGroup(): both decoration paths, chained.
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &buf).
		With(slog.String("module", "user_registration")).
		WithGroup("request").
		InfoContext(ctx, "decorated twice")

	if !strings.Contains(buf.String(), "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("trace_id lost after decoration: %s — "+
			"WithAttrs or WithGroup does not rewrap the handler", buf.String())
	}
}

// TestDisabledTelemetryYieldsAnInertShutdown: disabled, the shutdown succeeds.
//
// # The defect this test catches
//
// This is the DEFAULT configuration of the starter: `telemetry.enabled: false`,
// no collector. The shutdown must return nil, otherwise every graceful shutdown
// of a binary would return an error and the orchestrator would count the
// deployment as a failure.
//
// The contract is also that the caller does NOT have to know whether telemetry
// is active: the shutdown function is never nil.
func TestDisabledTelemetryYieldsAnInertShutdown(t *testing.T) {
	t.Parallel()

	var cfg config.Config
	cfg.Telemetry.Enabled = false

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("disabled Setup returned an error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("nil shutdown function — the caller would have to test before calling")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("the inert shutdown returned %v — every deployment would be counted as a failure", err)
	}
}

// TestASuccessfulShutdownReportsNoError: a SUCCESSFUL shutdown returns nil.
//
// # The defect this test brought to light
//
// The shutdown wrapped unconditionally: `fmt.Errorf("telemetry shutdown: %w",
// errJoin(...))`. With a `%w` on a nil, `fmt.Errorf` does NOT return nil — it
// returns an error carrying literally "%!w(<nil>)".
//
// A perfectly successful shutdown therefore returned an error. The defect was
// still LATENT — `Setup` is called nowhere, telemetry is not wired in (#13) —
// but it would have been triggered on the day of the wiring, on the most
// frequent path there is: the normal shutdown, at every deployment.
//
// A second fault lived next to it: the helper was called `errJoin` and joined
// nothing, it returned the first error and threw away the second. Both are
// replaced by `errors.Join`.
//
// # Why this test runs WITHOUT a collector
//
// gRPC connects LAZILY: `otlptracegrpc.New` succeeds without any collector
// listening, and the shutdown returns in under a millisecond. The enabled path
// is therefore exercisable without infrastructure — which does not for all that
// make the EXPORT proven: nobody has checked that a span really reaches a
// collector. That remains a matter for #13.
//
// No `t.Parallel()`: `Setup` installs the GLOBAL OpenTelemetry providers. A
// parallel test reading them would see a state modified under it.
func TestASuccessfulShutdownReportsNoError(t *testing.T) {
	var cfg config.Config
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.ServiceName = "hexa-tests"
	// Plausible address but with nothing behind it: that is the point of the
	// test.
	cfg.Telemetry.OTLPEndpoint = "127.0.0.1:4317"

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("enabled Setup returned an error: %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("a successful shutdown returned %v — want nil.\n"+
			"This is the defect of `fmt.Errorf(\"...: %%w\", nil)`: it does not return nil, "+
			"and every graceful shutdown would be counted as a failure", err)
	}
}
