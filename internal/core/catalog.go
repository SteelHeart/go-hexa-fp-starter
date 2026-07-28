// Package core assemble le catalogue des modules NOYAU — ADR 014.
//
// Il ne contient que ça, et volontairement : c'est le seul endroit du socle qui
// nomme ses sept modules, et il ne fait rien d'autre que les énumérer.
//
// # Pourquoi ce paquet existe
//
// `internal/config` ne peut nommer aucun module — règle 7 d'`arch-go`, qui lui
// interdit toute dépendance interne. Le catalogue doit donc être construit
// ailleurs et lui être passé.
//
// Le composition root pourrait le faire, mais `cmd/server` et `cmd/worker`
// écriraient alors la même liste deux fois — et le jour où elles divergeraient,
// un module serait configurable dans un binaire et pas dans l'autre, sans que
// rien ne le dise. C'est la divergence que ce dépôt a déjà payée trois fois.
//
// # Ce que ce catalogue déclare, et ce qu'il ne déclare pas
//
// Il déclare ce que les binaires du socle EMBARQUENT, pas ce qu'ils montent.
// La nuance compte : `cmd/server` ne câble aujourd'hui qu'`outbox`, tandis que
// `config/modules.yaml` déclare les six. Restreindre le catalogue aux seuls
// modules câblés ferait donc REFUSER la configuration livrée.
//
// L'ADR 014 formule la règle plus étroitement — « un module qui n'est pas monté
// n'est pas dans le catalogue ». L'écart est assumé, et écrit ici plutôt que taire :
// la propriété qui compte est qu'aucun nom ARBITRAIRE ne passe, et elle tient.
package core

import (
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// Catalog rend le catalogue fusionné des modules noyau.
//
// Chaque module déclare le sien chez lui, à côté de la fabrique qui construit
// ses pilotes. Ce fichier ne fait que les réunir : il ne contient aucun nom de
// pilote, donc il ne peut pas diverger d'une fabrique.
//
// Une collision de noms est impossible ici — sept constantes distinctes — mais
// `MergeCatalogs` la refuserait, et c'est cette garantie-là qui compte quand
// une application ajoutera les siens.
func Catalog() (config.ModuleCatalog, error) {
	merged, err := config.MergeCatalogs(
		outbox.Catalog(),
		idempotency.Catalog(),
		dynconf.Catalog(),
		audit.Catalog(),
		storage.Catalog(),
		scheduler.Catalog(),
		auth.Catalog(),
	)
	if err != nil {
		return nil, fmt.Errorf("catalogue du noyau: %w", err)
	}
	return merged, nil
}
