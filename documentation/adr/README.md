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
| [008](008-chi-huma-plutot-qu-un-framework.md) | chi + huma plutôt qu'un framework Go | Accepté |
| [009](009-strategie-d-acces-aux-donnees.md) | Stratégie d'accès aux données : pile en couches, pas d'ORM unique | Accepté |
| 010 | Messagerie enfichable et communication inter-modules | **À écrire** |
| [011](011-isolation-des-donnees-par-module.md) | Isolation des données : un schéma et un rôle SQL par module | Accepté |
| [012](012-anatomie-d-un-module-et-pilotes.md) | Anatomie d'un module, pilotes, zéro prérequis d'infrastructure | Accepté |
| [013](013-un-garde-doit-savoir-echouer.md) | Un garde est livré avec le cas qui le fait échouer | Accepté |
| [014](014-catalogue-de-modules-passe-au-chargeur.md) | Le catalogue des modules est une valeur passée au chargeur, pas une table du framework | Accepté |
| [015](015-la-frontiere-publique-est-derivee-d-un-usage-mesure.md) | La frontière publique est dérivée d'un usage mesuré, pas décidée d'avance | Accepté |
| [016](016-le-generateur-est-une-bibliotheque-pas-un-composition-root.md) | Le générateur est une bibliothèque, pas un composition root | Accepté |

> ⚠️ **010 n'existe pas encore, et du code le référence déjà** — `internal/infrastructure/messaging`
> et `internal/infrastructure/modulebus`. Le numéro est réservé pour que rien d'autre ne le prenne.
> Suivi par l'issue #14.

## Note de vocabulaire — lire avant les ADR 001 à 009

L'[ADR 012](012-anatomie-d-un-module-et-pilotes.md) a fixé le vocabulaire du dépôt, **après** que
les neuf premiers ADR aient été écrits. Ceux-ci emploient donc les mots de l'époque :

| Dans les ADR 001 à 009 | Aujourd'hui |
|---|---|
| *feature* | **module métier** (`internal/modules/{nom}/`) |
| *socle technique* | selon le cas : **module noyau** (`internal/core/`) ou infrastructure |

Ces textes ne sont **pas corrigés**, et c'est délibéré : un ADR est immuable, il se remplace, il ne
se réécrit pas. Réviser leur vocabulaire reviendrait à effacer la trace du moment où le vocabulaire
n'était pas encore fixé — c'est-à-dire à faire croire qu'il l'avait toujours été.

## Règles

- **Un ADR est immuable.** Il ne se modifie pas : il se **remplace** par un nouvel ADR qui le
  déclare *Remplacé par NNN*. L'historique des décisions vaut autant que la décision courante.
- Statuts : `Proposé` · `Accepté` · `Remplacé par NNN` · `Abandonné`.
- Numérotation continue sur trois chiffres, jamais réutilisée.
- Un ADR est **obligatoire** pour toute PR touchant `rules/`, `arch-go.yml`, `.golangci.yml`,
  `internal/pkg/` ou `migrations/` — la CI le vérifie (job `inertia`).

Gabarit : [`000-template.md`](000-template.md).
