package idempotency

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
// Écritures rejouables sans effet de bord.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : c'est ce qui rend vrai « `go run` démarre »
			// sans base, sans cache, sans conteneur (ADR 012).
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// `ttl` est lue AVANT le `switch` de `New`, donc par les trois
				// pilotes. `namespace` ne l'est que par redis, où elle préfixe les
				// clés : l'écrire ailleurs n'aurait aucun effet, et c'est
				// précisément le genre de réglage sans effet qu'on ne découvre
				// jamais (#93).
				//
				// Par instance : AUCUNE exclusivité derrière plusieurs répliques.
				driverMemory: {Options: []string{OptionTTL}},
				// Exclusivité entre répliques.
				driverPostgres: {SQL: true, Options: []string{OptionTTL}},
				// Exclusivité entre répliques, expiration passive.
				driverRedis: {Cache: true, Options: []string{OptionTTL, OptionNamespace}},
			},
		},
	}
}
