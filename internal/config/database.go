package config

// DB porte la connexion à la base de données.
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
