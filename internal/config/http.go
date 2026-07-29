package config

import "fmt"

// HTTP carries the parameters of the HTTP server.
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

// Addr returns the listening address.
func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

// Limits carries rate limiting.
//
// ⚠️ In memory, hence PER INSTANCE: behind N replicas the effective limit is
// multiplied by N (see SECURITY.md).
type Limits struct {
	RPS       float64 `yaml:"rps"`
	Burst     int     `yaml:"burst"`
	AuthRPS   float64 `yaml:"auth_rps"`
	AuthBurst int     `yaml:"auth_burst"`
}
