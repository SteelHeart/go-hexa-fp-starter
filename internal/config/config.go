// Package config lit la configuration de dÃ©marrage depuis les fichiers de conf/.
//
// Quatre principes, et ils expliquent tout le paquet :
//
//  1. Fichiers, pas variables d'environnement. La configuration est versionnÃ©e,
//     groupÃ©e par domaine, relisible en revue. Les variables d'environnement ne
//     servent QU'aux secrets, rÃ©fÃ©rencÃ©s par ${VAR} dans les fichiers.
//  2. Immuable â€” lue UNE fois au dÃ©marrage, passÃ©e par valeur. Aucun accÃ¨s Ã 
//     os.Getenv ailleurs dans le dÃ©pÃ´t.
//  3. Fail-fast â€” une configuration invalide refuse le dÃ©marrage. Un service qui
//     dÃ©marre Ã  moitiÃ© configurÃ© Ã©choue plus tard, ailleurs, et pour une raison
//     qui n'aura plus rien Ã  voir.
//  4. Ce qui change sans redÃ©ploiement n'est PAS ici : les seuils mÃ©tier et les
//     drapeaux vivent en base (internal/infrastructure/dynconf).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Environment nomme l'environnement d'exÃ©cution.
type Environment string

// Les environnements reconnus. Toute autre valeur refuse le dÃ©marrage.
const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvUAT         Environment = "uat"
	EnvProduction  Environment = "production"
)

// IsProduction indique si l'environnement exige les protections maximales.
func (e Environment) IsProduction() bool { return e == EnvProduction }

// IsDevelopment indique si l'on est en dÃ©veloppement local.
func (e Environment) IsDevelopment() bool { return e == EnvDevelopment }

// IsLocal regroupe dÃ©veloppement et test : les durcissements rÃ©seau ne s'y
// appliquent pas, mais AUCUN affaiblissement de sÃ©curitÃ© n'y est permis.
func (e Environment) IsLocal() bool { return e == EnvDevelopment || e == EnvTest }

// encryptionKeyBytes est la taille exigÃ©e par AES-256-GCM.
const encryptionKeyBytes = 32

// Config porte l'intÃ©gralitÃ© de la configuration de dÃ©marrage.
// Un groupe = un fichier dans conf/.
type Config struct {
	App       App       `yaml:"app"`
	HTTP      HTTP      `yaml:"http"`
	Limits    Limits    `yaml:"limits"`
	Database  DB        `yaml:"database"`
	Cache     Cache     `yaml:"cache"`
	DynConf   DynConf   `yaml:"dynconf"`
	Worker    Worker    `yaml:"worker"`
	Storage   Storage   `yaml:"storage"`
	Messaging Messaging `yaml:"messaging"`
	Modules   Modules   `yaml:"modules"`
	Security  Security  `yaml:"security"`
	Mail      Mail      `yaml:"mail"`
	Telemetry Telemetry `yaml:"telemetry"`
	I18n          I18n          `yaml:"i18n"`
	Observability Observability `yaml:"observability"`
}

// App porte l'identitÃ© du service.
type App struct {
	Env     Environment `yaml:"env"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
}

// HTTP porte les paramÃ¨tres du serveur HTTP.
type HTTP struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	MaxBodyBytes    int64         `yaml:"max_body_bytes"`
	AllowedOrigins  []string      `yaml:"allowed_origins"`
}

// Addr retourne l'adresse d'Ã©coute.
func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

// Limits porte la limitation de dÃ©bit.
//
// âš ï¸ En mÃ©moire, donc PAR INSTANCE : derriÃ¨re N rÃ©pliques la limite effective
// est multipliÃ©e par N (voir SECURITY.md).
type Limits struct {
	RPS       float64 `yaml:"rps"`
	Burst     int     `yaml:"burst"`
	AuthRPS   float64 `yaml:"auth_rps"`
	AuthBurst int     `yaml:"auth_burst"`
}

// DB porte la connexion Postgres.
//
// MigrationDSN est distinct de DSN : le rÃ´le applicatif ne possÃ¨de pas le
// schÃ©ma, ce qui empÃªche une injection SQL rÃ©ussie de le modifier ou de
// dÃ©sactiver une politique RLS (rules/donnees-et-migrations.md Â§6).
type DB struct {
	DSN             string        `yaml:"dsn"`
	MigrationDSN    string        `yaml:"migration_dsn"`
	MaxConns        int32         `yaml:"max_conns"`
	MinConns        int32         `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
}

// Cache porte la connexion au cache.
type Cache struct {
	Addr       string        `yaml:"addr"`
	Password   string        `yaml:"password"`
	DB         int           `yaml:"db"`
	DefaultTTL time.Duration `yaml:"default_ttl"`
}

// DynConf paramÃ¨tre le MÃ‰CANISME de configuration dynamique ; les valeurs, elles,
// vivent en base.
type DynConf struct {
	TTL       time.Duration `yaml:"ttl"`
	EnvPrefix string        `yaml:"env_prefix"`
}

// Worker porte le dÃ©pilage de l'outbox et les tÃ¢ches planifiÃ©es.
type Worker struct {
	PollInterval   time.Duration `yaml:"poll_interval"`
	BatchSize      int           `yaml:"batch_size"`
	MaxAttempts    int           `yaml:"max_attempts"`
	BaseBackoff    time.Duration `yaml:"base_backoff"`
	IdempotencyTTL time.Duration `yaml:"idempotency_ttl"`
}

// Storage porte le stockage d'objets.
type Storage struct {
	Kind    string `yaml:"kind"`
	DiskDir string `yaml:"disk_dir"`
	BaseURL string `yaml:"base_url"`
}

// Messaging porte le relais d'Ã©vÃ©nements.
//
// Le relais est INTERCHANGEABLE : l'outbox garantit la durabilitÃ© en amont, donc
// changer de broker ne touche aucune ligne du cÅ“ur (ADR 010).
type Messaging struct {
	Driver         string        `yaml:"driver"`
	TopicPrefix    string        `yaml:"topic_prefix"`
	ConsumerGroup  string        `yaml:"consumer_group"`
	PublishTimeout time.Duration `yaml:"publish_timeout"`
	Kafka          Kafka         `yaml:"kafka"`
	RabbitMQ       RabbitMQ      `yaml:"rabbitmq"`
}

// Kafka porte les paramÃ¨tres du relais Kafka.
type Kafka struct {
	Brokers                []string `yaml:"brokers"`
	AllowAutoTopicCreation bool     `yaml:"allow_auto_topic_creation"`
}

// RabbitMQ porte les paramÃ¨tres du relais AMQP.
type RabbitMQ struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
}

// Topic dÃ©rive le nom de destination d'un type d'Ã©vÃ©nement.
//
// Le point devient un tiret : AMQP donne au point un sens de routage
// hiÃ©rarchique, et Kafka le rÃ©serve Ã  ses conventions de mÃ©triques.
func (m Messaging) Topic(eventType string) string {
	return m.TopicPrefix + "." + strings.ReplaceAll(eventType, ".", "-")
}

// Modules porte les modes de communication inter-modules.
//
// Un module n'accÃ¨de JAMAIS aux tables d'un autre : il passe par le langage
// publiÃ© puis par l'un de ces modes (ADR 011).
type Modules struct {
	DefaultTransport string            `yaml:"default_transport"`
	CallTimeout      time.Duration     `yaml:"call_timeout"`
	Transports       map[string]string `yaml:"transports"`
	BaseURLs         map[string]string `yaml:"base_urls"`
}

// TransportFor rÃ©sout le mode applicable Ã  un module.
func (m Modules) TransportFor(module string) string {
	if raw, found := m.Transports[module]; found && strings.TrimSpace(raw) != "" {
		return strings.TrimSpace(raw)
	}
	if m.DefaultTransport == "" {
		return "inproc"
	}
	return m.DefaultTransport
}

// Security porte les paramÃ¨tres cryptographiques.
type Security struct {
	// EncryptionKey doit dÃ©coder sur exactement 32 octets (AES-256-GCM).
	// Toujours une rÃ©fÃ©rence ${VAR} dans les fichiers, jamais une valeur.
	EncryptionKey string `yaml:"encryption_key"`
	Argon2        Argon2 `yaml:"argon2"`
}

// Argon2 porte le coÃ»t du hachage de mot de passe.
type Argon2 struct {
	MemoryKiB  uint32 `yaml:"memory_kib"`
	Iterations uint32 `yaml:"iterations"`
	Threads    uint8  `yaml:"threads"`
}

// DecodedEncryptionKey dÃ©code et valide la clÃ© de chiffrement.
func (s Security) DecodedEncryptionKey() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(s.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("security.encryption_key n'est pas du base64 valide: %w", err)
	}
	if len(key) != encryptionKeyBytes {
		return nil, fmt.Errorf(
			"security.encryption_key doit faire %d octets, reÃ§u %d",
			encryptionKeyBytes, len(key),
		)
	}
	return key, nil
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
	// FailOnMissingKey fait Ã©chouer la CI sur une clÃ© absente d'un catalogue.
	// Une traduction manquante affiche sa clÃ©, ce qui est visible â€” mais mieux
	// vaut le voir en CI qu'en production.
	FailOnMissingKey bool `yaml:"fail_on_missing_key"`
}

// applyDefaults comble les valeurs que les fichiers pourraient ne pas porter.
//
// Ce sont des dÃ©fauts STRUCTURELS, pas des valeurs mÃ©tier : ils garantissent
// qu'un fichier de conf incomplet ne produit pas un pool Ã  zÃ©ro connexion.
func (c *Config) applyDefaults() {
	if c.App.Env == "" {
		c.App.Env = EnvDevelopment
	}
	if c.Database.MigrationDSN == "" {
		// En local, les deux rÃ´les peuvent coÃ¯ncider ; validate() l'interdit
		// ailleurs.
		c.Database.MigrationDSN = c.Database.DSN
	}
	if c.Messaging.Driver == "" {
		c.Messaging.Driver = "inproc"
	}
	if c.Modules.DefaultTransport == "" {
		c.Modules.DefaultTransport = "inproc"
	}
	if c.Modules.Transports == nil {
		c.Modules.Transports = map[string]string{}
	}
	if c.Modules.BaseURLs == nil {
		c.Modules.BaseURLs = map[string]string{}
	}
	c.Observability.applyDefaults()
	if c.I18n.DefaultLocale == "" {
		c.I18n.DefaultLocale = "fr"
	}
	if len(c.I18n.SupportedLocales) == 0 {
		c.I18n.SupportedLocales = []string{c.I18n.DefaultLocale}
	}
}

// validate rassemble TOUTES les invaliditÃ©s plutÃ´t que de s'arrÃªter Ã  la
// premiÃ¨re : corriger la configuration en six redÃ©marrages est inacceptable.
func (c Config) validate() error {
	problems := make([]error, 0, 4)
	problems = append(problems, c.validateCore()...)
	problems = append(problems, c.validateHardening()...)
	problems = append(problems, c.Observability.validate(c.App.Env.IsLocal())...)

	if len(problems) > 0 {
		return fmt.Errorf("configuration invalide: %w", errors.Join(problems...))
	}
	return nil
}

func (c Config) validateCore() []error {
	var problems []error

	switch c.App.Env {
	case EnvDevelopment, EnvTest, EnvUAT, EnvProduction:
	default:
		problems = append(problems, fmt.Errorf(
			"app.env=%q inconnu (attendu: development, test, uat, production)", c.App.Env))
	}

	if _, err := c.Security.DecodedEncryptionKey(); err != nil {
		problems = append(problems, err)
	}
	if c.Database.DSN == "" {
		problems = append(problems, errors.New("database.dsn est obligatoire"))
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		problems = append(problems, fmt.Errorf("http.port=%d hors plage", c.HTTP.Port))
	}
	if c.HTTP.ReadTimeout <= 0 {
		problems = append(problems, errors.New(
			"http.read_timeout doit Ãªtre > 0 : une connexion sans dÃ©lai immobilise une goroutine"))
	}
	if c.Database.MinConns > c.Database.MaxConns {
		problems = append(problems, fmt.Errorf(
			"database.min_conns=%d > database.max_conns=%d", c.Database.MinConns, c.Database.MaxConns))
	}
	if c.Worker.MaxAttempts < 1 {
		problems = append(problems, errors.New("worker.max_attempts doit Ãªtre >= 1"))
	}

	switch c.Messaging.Driver {
	case "inproc", "kafka", "rabbitmq", "noop":
	default:
		problems = append(problems, fmt.Errorf(
			"messaging.driver=%q inconnu (attendu: inproc, kafka, rabbitmq, noop)", c.Messaging.Driver))
	}

	for module, mode := range c.Modules.Transports {
		switch mode {
		case "inproc", "event", "disabled":
		case "http":
			if c.Modules.BaseURLs[module] == "" {
				problems = append(problems, fmt.Errorf(
					"modules.base_urls.%s est requis quand le transport est http", module))
			}
		default:
			problems = append(problems, fmt.Errorf(
				"modules.transports.%s=%q inconnu", module, mode))
		}
	}

	if !contains(c.I18n.SupportedLocales, c.I18n.DefaultLocale) {
		problems = append(problems, fmt.Errorf(
			"i18n.default_locale=%q absent de i18n.supported_locales", c.I18n.DefaultLocale))
	}
	return problems
}

// validateHardening porte les exigences qui ne s'appliquent qu'hors local.
//
// Deny par dÃ©faut : ce qui n'est pas explicitement sÃ»r est refusÃ©.
func (c Config) validateHardening() []error {
	if c.App.Env.IsLocal() {
		return nil
	}
	var problems []error

	if c.Database.MigrationDSN == c.Database.DSN {
		problems = append(problems, errors.New(
			"database.migration_dsn doit diffÃ©rer de database.dsn hors dÃ©veloppement "+
				"(le rÃ´le applicatif ne possÃ¨de pas le schÃ©ma)"))
	}
	if len(c.HTTP.AllowedOrigins) == 0 {
		problems = append(problems, errors.New("http.allowed_origins ne peut pas Ãªtre vide hors dÃ©veloppement"))
	}
	for _, origin := range c.HTTP.AllowedOrigins {
		switch {
		case origin == "*":
			problems = append(problems, errors.New("http.allowed_origins ne peut pas contenir '*' hors dÃ©veloppement"))
		case strings.HasPrefix(origin, "http://"):
			problems = append(problems, fmt.Errorf("origine non chiffrÃ©e interdite hors dÃ©veloppement: %s", origin))
		case origin == "":
			problems = append(problems, errors.New("http.allowed_origins contient une entrÃ©e vide (rÃ©fÃ©rence ${VAR} non rÃ©solue ?)"))
		}
	}
	if c.Messaging.Driver == "kafka" && c.Messaging.Kafka.AllowAutoTopicCreation {
		problems = append(problems, errors.New(
			"messaging.kafka.allow_auto_topic_creation doit Ãªtre false hors dÃ©veloppement : "+
				"crÃ©er un topic Ã  la volÃ©e masque une erreur de configuration"))
	}
	if !c.Telemetry.Enabled {
		problems = append(problems, errors.New(
			"telemetry.enabled doit Ãªtre true hors dÃ©veloppement : un service non observable n'est pas exploitable"))
	}
	return problems
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
