package storage

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog déclare les pilotes de ce module — ADR 014.
//
// # Pourquoi ici, et pas dans internal/config
//
// Cette table vivait dans `internal/config/modules.go`, à deux paquets de la
// fabrique qui construit réellement les pilotes. Le commentaire qui
// l'accompagnait avouait déjà craindre la divergence : « une faute de frappe
// dans l'une des deux rendrait un module inactivable, avec un message qui
// accuse la configuration de l'utilisateur ».
//
// Elle est désormais dans le MÊME paquet que le `switch` de `New`, souvent sur
// le même écran. La divergence ne devient pas impossible — rien ne la vérifie
// mécaniquement, et l'ADR 014 le note comme sa faiblesse [humain] — mais elle
// devient improbable.
//
// Dépôt d’objets, clés validées contre la traversée de répertoire.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : c'est ce qui rend vrai « `go run` démarre »
			// sans base, sans cache, sans conteneur (ADR 012).
			Default: driverDisk,
			Drivers: map[string]config.Resources{
				// Local au processus : rien n’est partagé entre répliques.
				driverDisk: {},
			},
		},
	}
}
