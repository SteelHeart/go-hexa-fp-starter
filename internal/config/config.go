// Package config lit la configuration de démarrage depuis les fichiers de conf/.
//
// Quatre principes, et ils expliquent tout le paquet :
//
//  1. Fichiers, pas variables d'environnement. La configuration est versionnée,
//     groupée par domaine, relisible en revue. Les variables d'environnement ne
//     servent QU'aux secrets, référencés par ${VAR} dans les fichiers.
//  2. Immuable — lue UNE fois au démarrage, passée par valeur. Aucun accès Ã
//     os.Getenv ailleurs dans le dépôt.
//  3. Fail-fast — une configuration invalide refuse le démarrage. Un service qui
//     démarre à moitié configuré échoue plus tard, ailleurs, et pour une raison
//     qui n'aura plus rien à voir.
//  4. Ce qui change sans redéploiement n'est PAS ici : les seuils métier et les
//     drapeaux vivent en base (internal/infrastructure/dynconf).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Environment nomme l'environnement d'exécution.
type Environment string

// Les environnements reconnus. Toute autre valeur refuse le démarrage.
const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvUAT         Environment = "uat"
	EnvProduction  Environment = "production"
)

// IsProduction indique si l'environnement exige les protections maximales.
func (e Environment) IsProduction() bool { return e == EnvProduction }

// IsDevelopment indique si l'on est en développement local.
func (e Environment) IsDevelopment() bool { return e == EnvDevelopment }

// IsLocal regroupe développement et test : les durcissements réseau ne s'y
// appliquent pas, mais AUCUN affaiblissement de sécurité n'y est permis.
func (e Environment) IsLocal() bool { return e == EnvDevelopment || e == EnvTest }

// encryptionKeyBytes est la taille exigée par AES-256-GCM.
const encryptionKeyBytes = 32

// Config porte l'intégralité de la configuration de démarrage.
// Un groupe = un fichier dans conf/.
type Config struct {
	App           App           `yaml:"app"`
	HTTP          HTTP          `yaml:"http"`
	Limits        Limits        `yaml:"limits"`
	Database      DB            `yaml:"database"`
	Cache         Cache         `yaml:"cache"`
	DynConf       DynConf       `yaml:"dynconf"`
	Worker        Worker        `yaml:"worker"`
	Storage       Storage       `yaml:"storage"`
	Messaging     Messaging     `yaml:"messaging"`
	Modules       Modules       `yaml:"modules"`
	Interop       Interop       `yaml:"interop"`
	Security      Security      `yaml:"security"`
	Mail          Mail          `yaml:"mail"`
	Telemetry     Telemetry     `yaml:"telemetry"`
	I18n          I18n          `yaml:"i18n"`
	Observability Observability `yaml:"observability"`
}

// App porte l'identité du service.
type App struct {
	Env     Environment `yaml:"env"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
}

// HTTP porte les paramètres du serveur HTTP.
type HTTP struct {
	Host            string   `yaml:"host"`
	Port            int      `yaml:"port"`
	ReadTimeout     Duration `yaml:"read_timeout"`
	WriteTimeout    Duration `yaml:"write_timeout"`
	IdleTimeout     Duration `yaml:"idle_timeout"`
	ShutdownTimeout Duration `yaml:"shutdown_timeout"`
	MaxBodyBytes    int64    `yaml:"max_body_bytes"`
	AllowedOrigins  []string `yaml:"allowed_origins"`
}

// Addr retourne l'adresse d'écoute.
func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

// Limits porte la limitation de débit.
//
// ⚠️ En mémoire, donc PAR INSTANCE : derrière N répliques la limite effective
// est multipliée par N (voir SECURITY.md).
type Limits struct {
	RPS       float64 `yaml:"rps"`
	Burst     int     `yaml:"burst"`
	AuthRPS   float64 `yaml:"auth_rps"`
	AuthBurst int     `yaml:"auth_burst"`
}

// DB porte la connexion Postgres.
//
// MigrationDSN est distinct de DSN : le rôle applicatif ne possède pas le
// schéma, ce qui empêche une injection SQL réussie de le modifier ou de
// désactiver une politique RLS (rules/donnees-et-migrations.md §6).
type DB struct {
	DSN             string   `yaml:"dsn"`
	MigrationDSN    string   `yaml:"migration_dsn"`
	MaxConns        int32    `yaml:"max_conns"`
	MinConns        int32    `yaml:"min_conns"`
	MaxConnLifetime Duration `yaml:"max_conn_lifetime"`
	ConnectTimeout  Duration `yaml:"connect_timeout"`
}

// Cache porte la connexion au cache.
type Cache struct {
	Addr       string   `yaml:"addr"`
	Password   string   `yaml:"password"`
	DB         int      `yaml:"db"`
	DefaultTTL Duration `yaml:"default_ttl"`
}

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

// Messaging porte le relais d'événements.
//
// Le relais est INTERCHANGEABLE : l'outbox garantit la durabilité en amont, donc
// changer de broker ne touche aucune ligne du cœur (ADR 010).
type Messaging struct {
	Driver         string   `yaml:"driver"`
	TopicPrefix    string   `yaml:"topic_prefix"`
	ConsumerGroup  string   `yaml:"consumer_group"`
	PublishTimeout Duration `yaml:"publish_timeout"`
	Kafka          Kafka    `yaml:"kafka"`
	RabbitMQ       RabbitMQ `yaml:"rabbitmq"`
}

// Kafka porte les paramètres du relais Kafka.
type Kafka struct {
	Brokers                []string `yaml:"brokers"`
	AllowAutoTopicCreation bool     `yaml:"allow_auto_topic_creation"`
}

// RabbitMQ porte les paramètres du relais AMQP.
type RabbitMQ struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
}

// Topic dérive le nom de destination d'un type d'événement.
//
// Le point devient un tiret : AMQP donne au point un sens de routage
// hiérarchique, et Kafka le réserve à ses conventions de métriques.
func (m Messaging) Topic(eventType string) string {
	return m.TopicPrefix + "." + strings.ReplaceAll(eventType, ".", "-")
}

// Security porte les paramètres cryptographiques.
type Security struct {
	// EncryptionKey doit décoder sur exactement 32 octets (AES-256-GCM).
	// Toujours une référence ${VAR} dans les fichiers, jamais une valeur.
	EncryptionKey string `yaml:"encryption_key"`
	Argon2        Argon2 `yaml:"argon2"`
}

// Argon2 porte le coût du hachage de mot de passe.
type Argon2 struct {
	MemoryKiB  uint32 `yaml:"memory_kib"`
	Iterations uint32 `yaml:"iterations"`
	Threads    uint8  `yaml:"threads"`
}

// DecodedEncryptionKey décode et valide la clé de chiffrement.
func (s Security) DecodedEncryptionKey() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(s.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("security.encryption_key n'est pas du base64 valide: %w", err)
	}
	if len(key) != encryptionKeyBytes {
		return nil, fmt.Errorf(
			"security.encryption_key doit faire %d octets, reçu %d",
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
	// FailOnMissingKey fait échouer la CI sur une clé absente d'un catalogue.
	// Une traduction manquante affiche sa clé, ce qui est visible — mais mieux
	// vaut le voir en CI qu'en production.
	FailOnMissingKey bool `yaml:"fail_on_missing_key"`
}

// applyDefaults comble les valeurs que les fichiers pourraient ne pas porter.
//
// Ce sont des défauts STRUCTURELS, pas des valeurs métier : ils garantissent
// qu'un fichier de conf incomplet ne produit pas un pool à zéro connexion.
func (c *Config) applyDefaults() {
	if c.App.Env == "" {
		c.App.Env = EnvDevelopment
	}
	if c.Database.MigrationDSN == "" {
		// En local, les deux rôles peuvent coïncider ; validate() l'interdit
		// ailleurs.
		c.Database.MigrationDSN = c.Database.DSN
	}
	if c.Messaging.Driver == "" {
		c.Messaging.Driver = "inproc"
	}
	if c.Interop.DefaultTransport == "" {
		c.Interop.DefaultTransport = "inproc"
	}
	if c.Interop.Transports == nil {
		c.Interop.Transports = map[string]string{}
	}
	if c.Interop.BaseURLs == nil {
		c.Interop.BaseURLs = map[string]string{}
	}
	c.Observability.applyDefaults()
	if c.I18n.DefaultLocale == "" {
		c.I18n.DefaultLocale = "fr"
	}
	if len(c.I18n.SupportedLocales) == 0 {
		c.I18n.SupportedLocales = []string{c.I18n.DefaultLocale}
	}
}

// validate rassemble TOUTES les invalidités plutôt que de s'arrêter à la
// première : corriger la configuration en six redémarrages est inacceptable.
func (c Config) validate() error {
	problems := make([]error, 0, 4)
	problems = append(problems, c.validateCore()...)
	problems = append(problems, c.validateHardening()...)
	problems = append(problems, c.Observability.validate(c.App.Env.IsLocal())...)
	problems = append(problems, c.Modules.validate()...)
	problems = append(problems, c.Interop.validate()...)

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
			"http.read_timeout doit être > 0 : une connexion sans délai immobilise une goroutine"))
	}
	if c.Database.MinConns > c.Database.MaxConns {
		problems = append(problems, fmt.Errorf(
			"database.min_conns=%d > database.max_conns=%d", c.Database.MinConns, c.Database.MaxConns))
	}
	if c.Worker.MaxAttempts < 1 {
		problems = append(problems, errors.New("worker.max_attempts doit être >= 1"))
	}

	switch c.Messaging.Driver {
	case "inproc", "kafka", "rabbitmq", "noop":
	default:
		problems = append(problems, fmt.Errorf(
			"messaging.driver=%q inconnu (attendu: inproc, kafka, rabbitmq, noop)", c.Messaging.Driver))
	}

	if !contains(c.I18n.SupportedLocales, c.I18n.DefaultLocale) {
		problems = append(problems, fmt.Errorf(
			"i18n.default_locale=%q absent de i18n.supported_locales", c.I18n.DefaultLocale))
	}
	return problems
}

// validateHardening porte les exigences qui ne s'appliquent qu'hors local.
//
// Deny par défaut : ce qui n'est pas explicitement sûr est refusé.
func (c Config) validateHardening() []error {
	if c.App.Env.IsLocal() {
		return nil
	}
	var problems []error

	if c.Database.MigrationDSN == c.Database.DSN {
		problems = append(problems, errors.New(
			"database.migration_dsn doit différer de database.dsn hors développement "+
				"(le rôle applicatif ne possède pas le schéma)"))
	}
	if len(c.HTTP.AllowedOrigins) == 0 {
		problems = append(problems, errors.New("http.allowed_origins ne peut pas être vide hors développement"))
	}
	for _, origin := range c.HTTP.AllowedOrigins {
		switch {
		case origin == "*":
			problems = append(problems, errors.New("http.allowed_origins ne peut pas contenir '*' hors développement"))
		case strings.HasPrefix(origin, "http://"):
			problems = append(problems, fmt.Errorf("origine non chiffrée interdite hors développement: %s", origin))
		case origin == "":
			problems = append(problems, errors.New("http.allowed_origins contient une entrée vide (référence ${VAR} non résolue ?)"))
		}
	}
	if c.Messaging.Driver == "kafka" && c.Messaging.Kafka.AllowAutoTopicCreation {
		problems = append(problems, errors.New(
			"messaging.kafka.allow_auto_topic_creation doit être false hors développement : "+
				"créer un topic à la volée masque une erreur de configuration"))
	}
	if !c.Telemetry.Enabled {
		problems = append(problems, errors.New(
			"telemetry.enabled doit être true hors développement : un service non observable n'est pas exploitable"))
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
