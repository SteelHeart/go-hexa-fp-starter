# ADR 012 — Anatomie d'un module, pilotes, et zéro prérequis d'infrastructure

- **Statut** : Accepté
- **Date** : 2026-07-25
- **Remplace** : le vocabulaire « feature » de [`rules/references.md`](../../rules/references.md)

## Contexte

Trois constats ont convergé.

**1. Le socle exigeait 6 tables pour démarrer.** L'outbox, l'idempotence, la configuration dynamique
et l'audit étaient câblés directement en Postgres. Or `spring init`, `symfony new` et `laravel new`
produisent tous une application qui **démarre sans base de données**. J'avais transformé des choix
d'implémentation en prérequis — l'inverse de ce que fait un framework.

**2. Chaque capacité du framework est structurellement un module.** L'authentification doit offrir
plusieurs modes selon la surface servie (OAuth2, OIDC via Google ou Microsoft, SSO, mot de passe) ;
la notification plusieurs canaux et plusieurs fournisseurs ; le paiement plusieurs prestataires.
Ce n'est pas trois problèmes différents : c'est **un port et plusieurs pilotes**, trois fois.

**3. Le vocabulaire était ambigu.** « Feature » désignait un *bounded context* métier, mais les
capacités du noyau ont exactement la même anatomie. Et « service » signifie déjà trois choses
distinctes : microservice, couche applicative, unité système.

## Décision

### Une seule anatomie

```
internal/modules/{nom}/
├── domain/                règles pures
├── ports/                 contrats — types fonction uniquement
├── application/           cas d'usage + décorateurs
├── drivers/               ← le point d'extension
│   ├── memory/            pilote SANS dépendance
│   ├── postgres/
│   │   └── migrations/    publiées SEULEMENT si ce pilote est choisi
│   └── {fournisseur}/
├── surfaces/              adaptateurs primaires optionnels : http · cli · events
└── module.go              composition root local
```

Elle est **identique** pour un module livré par le framework et pour un module écrit par une
application. Conséquence directe : `hexa make:module facturation` génère la même structure que celle
du noyau, et un module métier se promeut en module noyau sans réécriture.

### Vocabulaire imposé

| Terme | Définition |
|---|---|
| **Module** | L'unité structurelle. Anatomie ci-dessus. |
| **Module noyau** | Livré par le framework, activable par configuration |
| **Module métier** | Écrit par l'application |
| **Pilote** | Implémentation interchangeable d'un port de module, choisie par configuration |
| **Surface** | Adaptateur primaire : `http`, `cli`, `events`, `grpc` |
| ~~Service~~ | ⛔ **proscrit** — ambigu, et rend les recherches dans le code inexploitables |

### Zéro prérequis d'infrastructure

**Tout module noyau expose au moins un pilote sans dépendance externe, et c'est le défaut.**

| Module | Pilote par défaut | Autres pilotes |
|---|---|---|
| `outbox` | `memory` | `postgres` |
| `idempotency` | `memory` | `postgres` · `redis` |
| `dynconf` | `file` | `postgres` |
| `audit` | `log` | `postgres` |
| `storage` | `disk` | `s3` |
| `notification` | `log` | `smtp` · `mailjet` · `ses` · `twilio` |
| `payment` | `log` | `stripe` · autres prestataires |
| `auth` | — | `oauth2` · `oidc` · `saml` · `password` |

`hexa new mon-projet && go run ./cmd/server` doit démarrer : ni Postgres, ni Redis, ni Docker.

**Les migrations appartiennent au pilote, pas au socle.** Activer `outbox` en pilote `postgres`
publie *sa* migration, dans *son* schéma. Le pilote `memory` n'en publie aucune.

Un pilote `memory` n'est **pas un bouchon** : c'est le bon choix pour un test, une CLI ou un
développement local. Il documente explicitement ce qu'il ne garantit pas — ici, la durabilité.

## Conséquences

### Ce que ça achète

- **Démarrage en trente secondes**, sans dépendance. C'est la condition d'adoption d'un framework.
- Les tests d'intégration deviennent l'exception, pas la règle : un module se teste sur son pilote
  `memory`, sans conteneur — ce qui compte d'autant plus sans Docker en local.
- Le choix d'un fournisseur (Stripe, Mailjet, Google) n'entre nulle part dans le cœur.
- Un module s'extrait en service séparé plus tard sans réécriture (voir
  [ADR 011](011-isolation-des-donnees-par-module.md)).

### Ce que ça coûte

- **N implémentations par port au lieu d'une.** Chacune doit passer la même suite de conformité,
  sinon le pilote `memory` mentirait sur le comportement du pilote `postgres`.
- Un pilote sans durabilité **peut induire en erreur** : un développeur qui valide un comportement
  d'outbox sur `memory` n'a rien prouvé sur la reprise après incident. La documentation de chaque
  pilote doit énoncer ses garanties **et ses non-garanties**.
- Le renommage `internal/features/` → `internal/modules/` et la mise à jour du règlement.

### Ce que ça rend impossible

- Câbler Postgres, Redis ou un prestataire directement dans un cas d'usage.
- Livrer une migration hors d'un pilote.
- Employer le mot « service ».

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| Garder Postgres obligatoire | S'écarte de tous les frameworks de référence ; interdit le démarrage immédiat |
| Pilote `memory` réservé aux tests | Fait exister deux vérités : ce qu'on teste et ce qu'on exécute |
| Deux anatomies (module noyau ≠ feature) | Deux générateurs, deux documentations, et une migration de chemin à chaque promotion |
| « Module » / « service » | « Service » entre en collision avec la vision microservices de la version future |

## Garde

`.arch-go.yml` (chemins `internal.modules`), `depguard` (le cœur n'importe aucun pilote),
suite de **conformité de port** partagée entre tous les pilotes d'un même port, et validation de
`config/modules.yaml` au démarrage — un pilote inconnu refuse le démarrage.

Reste **non outillé** (`[humain]`) : l'interdiction du mot « service », et l'honnêteté de la
documentation d'un pilote sur ses non-garanties.
