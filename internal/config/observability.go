package config

// Observability carries the settings of the four signals, without naming any
// vendor in the code.
//
// # The principle that makes the choice of tool reversible
//
// The signals go out through STANDARD formats, never through proprietary SDKs:
//
//	logs     → JSON on stdout      read by Filebeat/Fluent Bit (ELK), Loki,
//	                               Datadog, CloudWatch — zero lines of code
//	traces   → OTLP                Tempo (Grafana), Jaeger, Datadog, Honeycomb,
//	                               Elastic APM
//	metrics  → Prometheus / OTLP   Mimir, Prometheus, Datadog, Elastic
//
// Moving from ELK to Grafana, or from Jaeger to Datadog, is therefore done in
// the infrastructure — OpenTelemetry collector, log agent — and never in the
// application.
//
// Error tracking is the ONLY signal that requires a dedicated client: it is
// isolated behind a port so that no package of the core imports its SDK.
type Observability struct {
	Logs    LogSink        `yaml:"logs"`
	Traces  TraceExporter  `yaml:"traces"`
	Metrics MetricExporter `yaml:"metrics"`
	Errors  ErrorReporter  `yaml:"errors"`
}

// LogSink chooses the destination of the logs.
type LogSink struct {
	// stdout | file
	//
	// stdout in every containerised case: it is the agent that collects, not
	// the application that pushes. An application that writes a file inside a
	// container creates a volume to manage and a risk of filling the disk.
	Sink     string `yaml:"sink"`
	FilePath string `yaml:"file_path"`
}

// TraceExporter chooses the trace exporter.
type TraceExporter struct {
	// otlp | none
	//
	// A single exporter: the OpenTelemetry collector duplicates towards several
	// destinations. Multiplying the exporters inside the application means
	// multiplying the points of failure.
	Exporter string `yaml:"exporter"`
	// grpc | http
	Protocol string `yaml:"protocol"`
	// SampleRatio reduces the volume under load. 1.0 in development.
	SampleRatio float64 `yaml:"sample_ratio"`
}

// MetricExporter chooses the metric exporter.
type MetricExporter struct {
	// prometheus (pull) | otlp (push) | none
	Exporter string `yaml:"exporter"`
}

// ErrorReporter chooses the error reporter.
type ErrorReporter struct {
	// none | sentry
	Reporter string `yaml:"reporter"`
	Sentry   Sentry `yaml:"sentry"`
}

// Sentry carries the settings of the Sentry reporter.
type Sentry struct {
	DSN              string  `yaml:"dsn"`
	TracesSampleRate float64 `yaml:"traces_sample_rate"`
	// SendDefaultPII must stay false: personal data does not leave the system
	// (rules/securite.md §5). Validation refuses it outside local.
	SendDefaultPII bool `yaml:"send_default_pii"`
}

// Admitted log sinks.
//
// Homonymy assumed with driverFile: a log sink is not a module driver, the two
// do not share a constant.
const (
	sinkStdout = "stdout"
	sinkFile   = "file"
)

// applyDefaults fills in the absent values with the safest choices: stdout,
// OTLP, Prometheus, no error reporter.
func (o *Observability) applyDefaults() {
	if o.Logs.Sink == "" {
		o.Logs.Sink = sinkStdout
	}
	if o.Traces.Exporter == "" {
		o.Traces.Exporter = "otlp"
	}
	if o.Traces.Protocol == "" {
		o.Traces.Protocol = "grpc"
	}
	if o.Traces.SampleRatio == 0 {
		o.Traces.SampleRatio = 1.0
	}
	if o.Metrics.Exporter == "" {
		o.Metrics.Exporter = "prometheus"
	}
	if o.Errors.Reporter == "" {
		o.Errors.Reporter = "none"
	}
}

// validate checks the coherence of the tooling choices — everywhere, local
// included.
//
// The `local` boolean that once drove this function hid two functions inside
// one: the rules of form, always true, and the rules of hardening, true outside
// development. They now live separately, and `hardened` is called by
// validateHardening along with all the others.
func (o Observability) validate() []error {
	var problems []error

	problems = appendUnlessOneOf(problems, "observability.logs.sink", o.Logs.Sink, sinkStdout, sinkFile)
	problems = appendUnlessOneOf(problems, "observability.traces.exporter", o.Traces.Exporter, "otlp", "none")
	problems = appendUnlessOneOf(problems, "observability.traces.protocol", o.Traces.Protocol, "grpc", "http")
	problems = appendUnlessOneOf(problems, "observability.metrics.exporter", o.Metrics.Exporter,
		"prometheus", "otlp", "none")
	problems = appendUnlessOneOf(problems, "observability.errors.reporter", o.Errors.Reporter, "none", "sentry")

	if o.Traces.SampleRatio < 0 || o.Traces.SampleRatio > 1 {
		problems = append(problems, errorf("observability.traces.sample_ratio must be between 0 and 1"))
	}
	if o.Errors.Reporter == "sentry" && o.Errors.Sentry.DSN == "" {
		problems = append(problems, errorf("observability.errors.sentry.dsn is required when the reporter is sentry"))
	}
	return problems
}

// hardened carries the observability requirements that only hold outside local.
func (o Observability) hardened() []error {
	var problems []error
	if o.Errors.Sentry.SendDefaultPII {
		problems = append(problems, errorf(
			"observability.errors.sentry.send_default_pii must be false: "+
				"personal data does not leave the system"))
	}
	if o.Logs.Sink == sinkFile {
		problems = append(problems, errorf(
			"observability.logs.sink=file is refused outside development: "+
				"in a container it is the agent that collects stdout"))
	}
	return problems
}
