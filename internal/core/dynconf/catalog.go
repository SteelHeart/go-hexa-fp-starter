package dynconf

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
// Drapeaux et réglages modifiables à chaud.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : c'est ce qui rend vrai « `go run` démarre »
			// sans base, sans cache, sans conteneur (ADR 012).
			Default: driverFile,
			Drivers: map[string]config.Resources{
				// ⚠️ Les deux pilotes n'admettent PAS les mêmes options, et c'est
				// exactement ce que la déclaration par pilote permet de dire.
				//
				// `flags` et `settings` portent les valeurs VERSIONNÉES : elles
				// n'ont de sens que pour le pilote fichier. Les écrire sous le
				// pilote postgres serait une erreur de conception silencieuse — les
				// valeurs y vivent en base, pas dans le dépôt.
				//
				// Lecture seule, rechargée depuis le disque.
				driverFile: {Options: []string{OptionFlags, OptionSettings}},
				// Partagé entre répliques.
				driverPostgres: {SQL: true, Options: []string{OptionTTL}},
			},
		},
	}
}
