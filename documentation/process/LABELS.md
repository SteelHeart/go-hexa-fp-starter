# Labels GitHub

Deux familles obligatoires (`type:`, `area:`) plus des marqueurs d'état. Une issue porte
**exactement un** `type:` et **au moins un** `area:`.

Ces labels sont créés par `task gh:labels` — le fichier [`labels.yml`](../../.github/labels.yml)
fait foi.

## `type:` — la nature du travail

| Label | Usage |
|---|---|
| `type:feature` | Nouvelle fonctionnalité |
| `type:fix` | Correction de bug |
| `type:sec` | Correction de sécurité — accompagne toujours un `sec:` |
| `type:docs` | Documentation, ADR, règlement |
| `type:refactor` | Restructuration sans changement de comportement |
| `type:test` | Tests uniquement |
| `type:chore` | Outillage, dépendances, CI |

## `area:` — où ça se passe

**Couches**

| Label | Portée |
|---|---|
| `area:core` | `domain/`, `ports/`, `application/` — le cœur pur |
| `area:pkg` | `internal/pkg/` — primitives transverses (zone à haute inertie) |
| `area:infra` | `internal/infrastructure/`, `config/` |
| `area:data` | Migrations, SQL, outbox |

**Surfaces**

| Label | Portée |
|---|---|
| `area:http` | Adaptateur HTTP — web et mobile |
| `area:cli` | Adaptateur ligne de commande |
| `area:events` | Consommateurs asynchrones, worker |

**Transverse**

| Label | Portée |
|---|---|
| `area:ci` | Workflows, gardes, outillage |
| `area:docs` | `rules/`, `documentation/` |

## `sec:` — sévérité

À poser sur toute issue issue du [registre de sécurité](../securite/registre-securite.md).

| Label | Définition |
|---|---|
| `sec:critique` | Contournement d'authentification ou d'autorisation, fuite de données, falsification de preuve |
| `sec:eleve` | Déni de service, vol de session, dégradation d'une garde |
| `sec:moyen` | Défaut de durcissement, contrôle manquant, exposition d'information |

**Aucune issue `sec:critique` ne part en production ouverte.** Une entrée du registre ne se ferme
qu'avec son **test de non-régression**.

## Marqueurs d'état

| Label | Signification |
|---|---|
| `blocked` | Bloqué — attend une décision ou un paramètre manquant |
| `needs-decision` | Arbitrage requis **avant** implémentation |
| `friction` | Trou du socle — voir [`JOURNAL_FRICTION.md`](JOURNAL_FRICTION.md) |
| `inertia:justified` | PR touchant une zone à haute inertie **sans** ADR, avec justification dans le corps. Utilisé par la garde CI `inertia` |
| `dependencies` | Posé par Dependabot |

> `blocked` sur une décision manquante n'est pas un échec : c'est le système qui fonctionne. Coder
> autour d'un paramètre non tranché produit une implémentation fausse qui inspire confiance —
> c'est le plus coûteux des défauts, parce qu'il ne se signale jamais.

## Board

Le board GitHub Projects **fait foi** pour le suivi de livraison. Aucun fichier `.md` de suivi n'est
versionné.

Champs : `Status` (Backlog · À faire · En cours · En revue · Livré · Bloqué), `Priorité` (P0–P3),
`Taille` (S/M/L/XL), `Surface` (core · http · cli · events · infra · data).
