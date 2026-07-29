package config

// DB carries the connection to the database.
//
// MigrationDSN is distinct from DSN: the application role does not own the
// schema, which prevents a successful SQL injection from modifying it or from
// disabling an RLS policy (rules/donnees-et-migrations.md §6).
type DB struct {
	DSN             string   `yaml:"dsn"`
	MigrationDSN    string   `yaml:"migration_dsn"`
	MaxConns        int32    `yaml:"max_conns"`
	MinConns        int32    `yaml:"min_conns"`
	MaxConnLifetime Duration `yaml:"max_conn_lifetime"`
	ConnectTimeout  Duration `yaml:"connect_timeout"`
}

// Cache carries the connection to the cache.
type Cache struct {
	Addr       string   `yaml:"addr"`
	Password   string   `yaml:"password"`
	DB         int      `yaml:"db"`
	DefaultTTL Duration `yaml:"default_ttl"`
}
