package userregistration

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog déclare les pilotes de ce module — ADR 014.
//
// # C'est CE fichier qui répond à la friction
//
// Avant l'ADR 014, un module métier ne pouvait pas figurer dans
// `config/modules.yaml` : la configuration refusait son nom avec
// `modules.user_registration : module inconnu`, parce que la seule table de
// modules vivait dans `internal/config/modules.go` — un fichier du framework.
//
// Conséquence mesurée : `cmd/server` lisait `cfg.Modules[Name].Driver`, un
// champ qui restait TOUJOURS vide. La tranche de référence contournait donc
// silencieusement le mécanisme qu'elle est censée démontrer — et c'est sa forme
// qui sera copiée pour écrire `billing`.
//
// Ce fichier est la démonstration : un module métier déclare ses pilotes chez
// lui, le composition root les fusionne, et AUCUN fichier du framework ne nomme
// `user_registration`. Écrire `billing` demande le même fichier, au même
// endroit, sans toucher au socle.
//
// Le module reste supprimable d'un seul `rm -rf` : ce fichier part avec lui, et
// la seule ligne à retirer est celle du composition root qui le monte.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// `memory` est le défaut, et c'est une décision : il n'exige aucune
			// infrastructure, donc `go run` démarre. Voir les NON-garanties du
			// paquet drivers/memory avant de l'envisager ailleurs qu'en
			// développement.
			Default: DriverMemory,
			Drivers: map[string]config.Resources{
				// Perdu au redémarrage, et local au processus.
				DriverMemory: {},
			},
		},
	}
}
