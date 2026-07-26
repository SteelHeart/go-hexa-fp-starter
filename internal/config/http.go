package config

import "fmt"

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
