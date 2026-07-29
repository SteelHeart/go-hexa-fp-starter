package config

// Environment names the runtime environment.
type Environment string

// The recognised environments. Any other value refuses to start.
const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvUAT         Environment = "uat"
	EnvProduction  Environment = "production"
)

// IsProduction tells whether the environment requires the maximum protections.
func (e Environment) IsProduction() bool { return e == EnvProduction }

// IsDevelopment tells whether we are in local development.
func (e Environment) IsDevelopment() bool { return e == EnvDevelopment }

// IsLocal groups development and test: the network hardenings do not apply
// there, but NO weakening of security is permitted there either.
func (e Environment) IsLocal() bool { return e == EnvDevelopment || e == EnvTest }
