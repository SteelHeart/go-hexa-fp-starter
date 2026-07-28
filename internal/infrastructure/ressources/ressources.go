// Package ressources ouvre les connexions que les modules ACTIVÉS réclament, et
// aucune autre.
//
// # Le défaut que ce paquet corrige — #103
//
// `database.New` n'était appelée par AUCUN binaire. Ni `cmd/server`, ni
// `cmd/worker`. Conséquence mesurée : le dépileur refusait le pilote `memory`
// par conception — il dépilerait un magasin que le serveur ne partage pas — et
// ne pouvait pas utiliser `postgres` faute de pool. **Il ne démarrait dans
// aucune configuration.**
//
// Par extension, aucun pilote `postgres` du dépôt n'était atteignable depuis un
// binaire : `audit`, `dynconf`, `idempotency`, `outbox`, `scheduler` en ont tous
// un, tous inaccessibles. Ils ont des tests d'intégration (#37) et rien ne les
// montait.
//
// Personne ne l'avait vu parce que le seul chemin jamais exercé était celui du
// REFUS : le job CI vérifie que le dépileur refuse une outbox non partagée, ce
// qui est une bonne garde — mais le chemin nominal n'était exécuté nulle part.
//
// # Pourquoi ce paquet, et pas cinq lignes dans chaque composition root
//
// Parce qu'il y en a DEUX, et que ce dépôt a déjà payé trois fois la divergence
// entre `cmd/server` et `cmd/worker`. Une ouverture de pool qui diffère d'un
// binaire à l'autre produit un serveur qui écrit et un dépileur qui ne lit pas —
// exactement le défaut qu'on vient de corriger, sous une autre forme.
//
// # Ce que ce paquet NE fait pas
//
// Il n'ouvre rien « au cas où ». C'est la promesse de l'ADR 012 : avec la
// configuration livrée, tous les pilotes sont sans dépendance, donc **aucune
// connexion n'est ouverte** et `go run ./cmd/server` démarre sur une machine
// vierge. Corriger #103 ne devait pas coûter cette promesse.
package ressources

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/cache"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Connexions porte ce qui a été ouvert, et de quoi le refermer.
//
// Une structure plutôt que trois retours : `Open` rendrait
// `(*pgxpool.Pool, *goredis.Client, error)`, et la règle d'architecture le
// refuse. Elle a raison ici pour une raison propre au sujet — c'est la SIXIÈME
// occurrence de la même leçon dans ce dépôt, après `election`, `decodedHash`,
// `RetryPolicy`, `messaging.Broker` et `worker`.
//
// ⚠️ **Les deux champs peuvent être nil, légitimement.** Un nil ne signale pas
// un échec : il signale qu'aucun module activé ne réclamait cette ressource. Les
// modules qui n'en ont pas besoin reçoivent ce nil sans jamais le déréférencer,
// et ceux qui en ont besoin refusent le démarrage en le disant.
type Connexions struct {
	Pool  *pgxpool.Pool
	Cache *goredis.Client
}

// Open ouvre ce que la configuration réclame, et rien de plus.
//
// La décision vient de `config.Modules.RequiresSQL` et `RequiresCache`, qui
// n'interrogent que les modules ACTIVÉS et le pilote réellement retenu. Ces deux
// fonctions existaient et n'étaient appelées par aucun binaire : la promesse
// « démarre sans base » était *assertée* par des tests, jamais *exercée*.
//
// L'ouverture VÉRIFIE que la connexion répond — `database.New` fait un ping,
// `cache.New` aussi. C'est délibéré : un service qui démarre sans base signale
// son défaut à la première requête utilisateur, c'est-à-dire trop tard, et
// depuis un endroit qui n'accuse pas la bonne cause.
func Open(ctx context.Context, cfg config.Config, catalog config.ModuleCatalog) (Connexions, error) {
	var ouvertes Connexions

	if cfg.Modules.RequiresSQL(catalog) {
		pool, err := database.New(ctx, cfg.Database)
		if err != nil {
			return Connexions{}, fmt.Errorf("un module activé exige une base: %w", err)
		}
		ouvertes.Pool = pool
	}

	if cfg.Modules.RequiresCache(catalog) {
		client, err := cache.New(ctx, cfg.Cache)
		if err != nil {
			// Le pool déjà ouvert est refermé avant de remonter : sans cela, un
			// démarrage qui échoue à mi-chemin laisserait des connexions
			// pendantes, et un redémarrage en boucle les accumulerait jusqu'à
			// saturer le serveur de base.
			ouvertes.Close()
			return Connexions{}, fmt.Errorf("un module activé exige un cache: %w", err)
		}
		ouvertes.Cache = client
	}

	return ouvertes, nil
}

// Close referme ce qui a été ouvert. Sûre sur une valeur zéro.
//
// Ne rend PAS d'erreur, délibérément. Elle est appelée en `defer` au moment de
// l'arrêt, où il n'existe plus personne à qui rendre compte : un appelant qui
// recevrait une erreur ne pourrait qu'ignorer ou journaliser, et l'obligation de
// la traiter pousse à écrire `defer func() { _ = c.Close() }()` — donc à
// masquer, plutôt qu'à décider.
func (c Connexions) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
	if c.Cache != nil {
		_ = c.Cache.Close()
	}
}
