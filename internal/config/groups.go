package config

// Ce fichier rassemble les groupes de configuration qui ne portent AUCUN
// comportement : ce sont des enregistrements de valeurs, lus tels quels par le
// composition root.
//
// Le jour où l'un d'eux gagne une méthode — une valeur dérivée, une validation
// qui lui est propre — il quitte ce fichier pour le sien. C'est ce déménagement
// qui signale que le groupe est devenu autre chose qu'un sac de champs.

// DynConf paramètre le MÉCANISME de configuration dynamique ; les valeurs, elles,
// vivent en base.
type DynConf struct {
	TTL       Duration `yaml:"ttl"`
	EnvPrefix string   `yaml:"env_prefix"`
}

// Worker porte le dépilage de l'outbox et les tâches planifiées.
type Worker struct {
	PollInterval   Duration `yaml:"poll_interval"`
	BatchSize      int      `yaml:"batch_size"`
	MaxAttempts    int      `yaml:"max_attempts"`
	BaseBackoff    Duration `yaml:"base_backoff"`
	IdempotencyTTL Duration `yaml:"idempotency_ttl"`
}

// Storage porte le stockage d'objets.
type Storage struct {
	Kind    string `yaml:"kind"`
	DiskDir string `yaml:"disk_dir"`
	BaseURL string `yaml:"base_url"`
}

// Mail porte l'envoi de courriel.
type Mail struct {
	Addr     string `yaml:"addr"`
	From     string `yaml:"from"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Telemetry porte l'export OpenTelemetry et le journal.
type Telemetry struct {
	Enabled      bool   `yaml:"enabled"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	ServiceName  string `yaml:"service_name"`
	MetricsPort  int    `yaml:"metrics_port"`
	LogLevel     string `yaml:"log_level"`
	LogFormat    string `yaml:"log_format"`
}

// I18n porte l'internationalisation.
//
// Le domaine ne produit JAMAIS de texte traduit : il retourne un ErrorCode, et
// chaque surface le traduit depuis son catalogue (rules/internationalisation.md).
type I18n struct {
	DefaultLocale    string   `yaml:"default_locale"`
	SupportedLocales []string `yaml:"supported_locales"`
	// FailOnMissingKey fait échouer la CI sur une clé absente d'un catalogue.
	// Une traduction manquante affiche sa clé, ce qui est visible — mais mieux
	// vaut le voir en CI qu'en production.
	FailOnMissingKey bool `yaml:"fail_on_missing_key"`
}
