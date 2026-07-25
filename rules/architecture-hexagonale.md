# Architecture hexagonale

> Décision de référence : [ADR 001](../documentation/adr/001-hexagonal-modulaire-et-fonctionnel.md).
> Garde : [`.arch-go.yml`](../.arch-go.yml) + `depguard` dans [`.golangci.yml`](../.golangci.yml).

## 1. La seule règle qui compte

**Les dépendances pointent toujours vers l'intérieur.** Le cœur ne sait pas qu'il existe un monde
extérieur ; il déclare ce dont il a besoin, et quelqu'un d'autre le lui fournit.

```
            ┌──────────────────── cmd/ ────────────────────┐
            │   server · worker · cli                      │  ← composition root
            └──────────────────┬───────────────────────────┘
                               │ construit
   adaptateurs PRIMAIRES       ▼        adaptateurs SECONDAIRES
   (appellent le cœur)   ┌──────────┐   (implémentent ses besoins)
   http · cli · events ─►│  ports   │◄─ postgres · outbox · mailer
                         ├──────────┤
                         │application│  pipeline + décorateurs
                         ├──────────┤
                         │  domain  │  règles pures
                         └──────────┘
```

## 2. Matrice d'imports — qui a le droit d'importer quoi

| Depuis ↓ / Vers → | `domain` | `ports` | `application` | `adapters` | `infrastructure` | `pkg` | autre feature |
|---|---|---|---|---|---|---|---|
| `domain` | — | ❌ | ❌ | ❌ | ❌ | ✅ `result`, `fp` | ❌ |
| `ports` | ✅ | — | ❌ | ❌ | ❌ | ✅ `result`, `fp` | ❌ |
| `application` | ✅ | ✅ | — | ❌ | ❌ | ✅ | ❌ |
| `adapters/primary` | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `adapters/secondary` | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `module.go` (racine feature) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| `infrastructure` | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `cmd/*` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

Deux cases méritent une explication :

- **`adapters/primary` n'importe pas `application`.** Un adaptateur reçoit un *port* (type
  fonction) par paramètre. Il ne construit rien. C'est ce qui rend un handler HTTP testable avec
  une closure de trois lignes.
- **`module.go` a le droit de tout voir.** C'est le *composition root* local de la feature : il
  assemble les adaptateurs secondaires, construit les cas d'usage, applique les décorateurs, et
  expose les ports primaires. Il n'a **aucune logique**.

## 3. Anatomie d'une feature

```
internal/features/{feature}/
├── module.go                 # composition root local — assemble et expose les ports primaires
├── domain/                   # PUR : value objects, règles, erreurs, événements
├── ports/                    # SIGNATURES SEULEMENT — types fonction
├── application/              # pipeline de cas d'usage + décorateurs
└── adapters/
    ├── primary/              # ce qui APPELLE le cœur — un sous-dossier par transport
    │   ├── http/             #   web + mobile
    │   ├── cli/              #   terminal
    │   └── events/           #   consommateur asynchrone
    └── secondary/            # ce qui SERT le cœur
        ├── postgres/
        ├── outbox/
        └── mailer/
```

`domain/` ne contient **aucune** interface : les contrats sont dans `ports/`, et ce sont des types
fonction ([`ports-et-contrats.md`](ports-et-contrats.md)).

## 4. Étanchéité entre features

Une feature **n'importe jamais** une autre feature. Aucune exception, y compris « juste pour un
type partagé ».

Deux features communiquent par **événement** : la source écrit dans l'`outbox` **dans la même
transaction** que son changement d'état ; le worker dépile et publie ; la cible consomme via son
adaptateur `primary/events/` et met à jour **ses propres tables**.

Ce que ça coûte : de la cohérence à terme, de la duplication de données, un worker à opérer.
Ce que ça achète : la possibilité de supprimer, réécrire ou extraire une feature sans toucher aux
autres. Si ce coût n'est pas acceptable pour deux features données, **elles n'en font qu'une** —
fusionner est une décision légitime, contourner ne l'est pas.

## 5. Composition root

Trois niveaux, et un seul autorisé à connaître le monde entier :

1. `cmd/{server,worker,cli}/main.go` — lit la config, ouvre les connexions, construit les modules,
   branche les adaptateurs primaires, gère l'arrêt propre.
2. `internal/features/{feature}/module.go` — assemble **une** feature.
3. Tout le reste — reçoit ses dépendances, n'en construit aucune.

**Aucun conteneur d'injection.** La composition manuelle *est* de la programmation fonctionnelle
(application partielle explicite), elle est vérifiée par le compilateur, et elle se lit de haut en
bas ([ADR 004](../documentation/adr/004-composition-manuelle-sans-conteneur-di.md)).

## 6. Le socle `internal/infrastructure/`

Technique et **sans connaissance du métier** : pool Postgres, client Redis, serveur HTTP,
télémétrie, chiffrement, dépileur d'outbox générique. `arch-go` refuse tout import de
`internal/features/**` depuis ce dossier.

Corollaire : il ne contient **aucune** règle métier, et il est réutilisable tel quel dans un autre
projet.

## 7. `internal/pkg/` — primitives

`result`, `fp`, `middleware`. **Zéro dépendance externe** pour `result` et `fp` (`arch-go` le
vérifie). Ce qu'on met ici est un engagement à très long terme : un changement de signature s'y
propage à tout le dépôt. Dans le doute, ne pas y mettre.
