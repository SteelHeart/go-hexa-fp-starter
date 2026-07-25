# ADR 008 — chi + huma plutôt qu'un framework Go

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Le choix d'un socle HTTP engage pour des années. L'écosystème Go propose trois familles :

- **frameworks opinionated** — Encore, Kratos, Go-Zero, GoFrame, Goa ;
- **micro-frameworks** — `net/http` (stdlib), chi, echo, gin, fiber ;
- **générateurs** — huma, ogen, oapi-codegen, sqlc, connectrpc.

Le critère qui domine ici n'est pas la popularité ni la vitesse, mais la **confinabilité** : dans
une architecture hexagonale, le framework est un détail d'implémentation d'un adaptateur primaire.
La question n'est donc pas « lequel est le meilleur » mais « lequel reste enfermable dans
`adapters/primary/http/` ».

## Décision

**chi** pour le routage, **huma v2** pour la validation et la génération d'OpenAPI.

- chi est 100 % compatible `http.Handler` — l'interface la plus stable de l'écosystème Go, jamais
  cassée depuis 2012.
- huma est *code-first* : les structs Go annotées sont la source de vérité, `api/openapi.yaml` en
  est le produit. Écrire les deux garantirait leur divergence.
- Un changement cassant du contrat impose une **nouvelle version de route**, jamais une
  modification en place : plusieurs surfaces déployées indépendamment consomment la v1.

## Conséquences

### Ce que ça achète

- Le coût de sortie est d'une journée : `infrastructure/http_server/` plus un mapper par feature.
- Tout l'écosystème `http.Handler` reste disponible : `httptest`, `otelhttp`, middlewares standards.
- Un handler se teste sans démarrer de serveur.
- Le contrat d'API et les SDK clients sont générés, donc jamais divergents.

### Ce que ça coûte

- Pas de génération de squelette, pas de CLI : tout le câblage est à écrire (c'est aussi
  l'[ADR 004](004-composition-manuelle-sans-conteneur-di.md)).
- huma impose sa forme d'entrée/sortie (`struct` avec champ `Body`) dans l'adaptateur HTTP.
- Deux dépendances plutôt qu'une stdlib pure — `net/http` seul aurait suffi au routage, mais sans
  sous-routeurs ni chaînes de middlewares prêtes.

### Ce que ça rend impossible

- Utiliser un `Context` maison dans un adaptateur (gin, echo, fiber) — ce qui rendrait les
  handlers non testables sans le framework.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| **gin** / **echo** | `Context` maison : l'adaptateur devient dépendant du framework, et les middlewares standards demandent des ponts |
| **fiber** | Hors `net/http` (fasthttp) : exclut tout l'écosystème `http.Handler`, HTTP/2 et h2c limités |
| **Kratos** / **Go-Zero** / **GoFrame** | Imposent un layout et un style concurrents de celui du socle ; le code généré *devient* l'architecture |
| **Encore** | Excellente expérience de développement, mais toolchain propriétaire — incompatible avec « socle durable et réversible » |
| **Goa** | Design-first cohérent, mais courbe d'apprentissage et écosystème restreint |
| `net/http` seul | Suffisant pour le routage ; chi ajoute sous-routeurs et middlewares pour une dépendance sans transitive |
| **connectrpc** (gRPC + HTTP/JSON) | Pertinent dès qu'il y a du service-à-service ; reste ajoutable comme adaptateur primaire supplémentaire sans toucher au cœur |

## Garde

`depguard` interdit `huma` et `chi` dans le cœur ; `rules/dependances.md` interdit `fiber` par nom ;
`arch-go` empêche `adapters/primary/http` d'importer `application/`.
