package outbox

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
// optionsDuDepileur énumère les clés lues par `policyFrom`.
//
// Une fonction plutôt qu'une variable de paquet : `gochecknoglobals` refuse un
// état global, et il a raison — une tranche partagée est modifiable par
// n'importe quel appelant, y compris par accident.
func optionsDuDepileur() []string {
	return []string{OptionBatchSize, OptionMaxAttempts, OptionBaseBackoff, OptionInterval}
}

// Publication garantie d’événements.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : c'est ce qui rend vrai « `go run` démarre »
			// sans base, sans cache, sans conteneur (ADR 012).
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// Les deux pilotes lisent la MÊME politique de dépilage : elle
				// décrit le comportement du dépileur, pas le stockage.
				//
				// Les clés sont référencées, jamais réécrites. Une clé admise que
				// personne ne lit — ou lue sans être admise — serait une
				// divergence entre deux listes ; le partage de constante la rend
				// impossible (#93).
				//
				// Perdu au redémarrage : le processus EST le stockage.
				driverMemory: {Options: optionsDuDepileur()},
				// Durable et transactionnel — la seule forme qui tient la promesse de l’ADR 006.
				driverPostgres: {SQL: true, Options: optionsDuDepileur()},
			},
		},
	}
}
