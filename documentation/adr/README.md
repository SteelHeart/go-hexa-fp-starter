# Décisions d'architecture (ADR)

> **Font foi.** En cas de contradiction entre un ADR et un autre document, l'ADR gagne — et l'autre
> document est corrigé dans la même PR.

Une décision non écrite n'existe pas : elle sera re-débattue au premier changement de contexte, et
tranchée différemment. Le coût d'écrire un ADR est de vingt minutes ; celui de re-trancher est une
refonte.

## Index

| N° | Décision | Statut |
|---|---|---|
| [001](001-hexagonal-modulaire-et-fonctionnel.md) | Hexagonal modulaire + programmation fonctionnelle | Accepté |
| [002](002-result-et-limites-du-typage-go.md) | `Result[T, E]` et limites réelles du typage Go | Accepté |
| [003](003-ports-comme-types-fonction.md) | Un port est un type fonction, jamais une interface | Accepté |
| [004](004-composition-manuelle-sans-conteneur-di.md) | Composition manuelle, sans conteneur d'injection | Accepté |
| [005](005-n-frontends-adaptateurs-primaires.md) | N frontends simultanés comme adaptateurs primaires | Accepté |
| [006](006-outbox-transactionnel.md) | Outbox transactionnel comme seule sortie vers le monde | Accepté |
| [007](007-tronc-unique-et-environnements.md) | Tronc unique ; un environnement n'est pas une branche | Accepté |
| [009](009-strategie-d-acces-aux-donnees.md) | Strategie d'acces aux donnees : pile en couches, pas d'ORM unique | Accepte |
| [012](012-anatomie-d-un-module-et-pilotes.md) | Anatomie d'un module, pilotes, zero prerequis d'infrastructure | Accepte |
| [008](008-chi-huma-plutot-qu-un-framework.md) | chi + huma plutôt qu'un framework Go | Accepté |

## Règles

- **Un ADR est immuable.** Il ne se modifie pas : il se **remplace** par un nouvel ADR qui le
  déclare *Remplacé par NNN*. L'historique des décisions vaut autant que la décision courante.
- Statuts : `Proposé` · `Accepté` · `Remplacé par NNN` · `Abandonné`.
- Numérotation continue sur trois chiffres, jamais réutilisée.
- Un ADR est **obligatoire** pour toute PR touchant `rules/`, `.arch-go.yml`, `.golangci.yml`,
  `internal/pkg/` ou `migrations/` — la CI le vérifie (job `inertia`).

Gabarit : [`000-template.md`](000-template.md).
