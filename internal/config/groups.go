package config

// This file gathers the configuration groups that carry NO behaviour: they are
// records of values, read as they are by the composition root.
//
// The day one of them gains a method — a derived value, a validation of its
// own — it leaves this file for its own. It is that move which signals that the
// group has become something other than a bag of fields.

// DynConf configures the MECHANISM of dynamic configuration; the values
// themselves live in the database.
type DynConf struct {
	TTL       Duration `yaml:"ttl"`
	EnvPrefix string   `yaml:"env_prefix"`
}

// Worker carries the dispatching of the outbox and the scheduled tasks.
type Worker struct {
	PollInterval   Duration `yaml:"poll_interval"`
	BatchSize      int      `yaml:"batch_size"`
	MaxAttempts    int      `yaml:"max_attempts"`
	BaseBackoff    Duration `yaml:"base_backoff"`
	IdempotencyTTL Duration `yaml:"idempotency_ttl"`
}

// Storage carries object storage.
type Storage struct {
	Kind    string `yaml:"kind"`
	DiskDir string `yaml:"disk_dir"`
	BaseURL string `yaml:"base_url"`
}

// Mail carries the sending of email.
type Mail struct {
	Addr     string `yaml:"addr"`
	From     string `yaml:"from"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Telemetry carries the OpenTelemetry export and the log.
type Telemetry struct {
	Enabled      bool   `yaml:"enabled"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	ServiceName  string `yaml:"service_name"`
	MetricsPort  int    `yaml:"metrics_port"`
	LogLevel     string `yaml:"log_level"`
	LogFormat    string `yaml:"log_format"`
}

// I18n carries internationalisation.
//
// The domain NEVER produces translated text: it returns an ErrorCode, and each
// surface translates it from its catalogue (rules/internationalisation.md).
type I18n struct {
	DefaultLocale    string   `yaml:"default_locale"`
	SupportedLocales []string `yaml:"supported_locales"`
	// FailOnMissingKey makes the CI fail on a key absent from a catalogue.
	// A missing translation displays its key, which is visible — but it is
	// better to see it in CI than in production.
	FailOnMissingKey bool `yaml:"fail_on_missing_key"`
}
