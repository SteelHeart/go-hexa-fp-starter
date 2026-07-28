package auth

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog déclare les pilotes de ce module — ADR 014.
//
// Aucun fichier du framework ne nomme `auth` : le catalogue est construit ici, à
// côté de la fabrique `New`, et il PARTAGE ses constantes avec le `switch` de
// celle-ci. La divergence entre ce qui est déclarable et ce qui est
// constructible devient donc impossible.
//
// # Pourquoi un seul pilote pour l'instant
//
// L'ADR 017 déclare la cible — `postgres`, puis `oidc` — et n'en livre qu'un.
// Ce dépôt a déjà payé la faute inverse : huit paquets de pilotes ont vécu des
// mois avec ZÉRO test, à aucun niveau (#37), et deux défauts de production y
// dormaient. Un pilote écrit sans rien pour l'éprouver est du code qui a l'air
// de marcher.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : c'est ce qui rend vraie la promesse
			// « `hexa new` puis `go run`, ça démarre » — y compris avec
			// l'authentification active.
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// Perdu au redémarrage, et local au processus : voir les
				// NON-garanties du paquet drivers/memory avant de l'envisager
				// ailleurs qu'en développement.
				driverMemory: {Options: []string{OptionSessionTTL}},
			},
		},
	}
}
