package config

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
