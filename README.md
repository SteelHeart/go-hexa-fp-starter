# Socle Go — hexagonal modulaire, fonctionnel, multi-frontend

Base de projet Go réutilisable : **architecture hexagonale**, découpe en **features étanches**,
cœur en **style fonctionnel**, exposé à **N frontends simultanés** (web, mobile, CLI, événements).

Ce n'est pas une application. Sa valeur n'est pas le code de démonstration, c'est la **forme** qu'il
impose — et les gardes automatiques qui empêchent cette forme de se dégrader.

## Deux propriétés, et tout le reste en découle

**1. Le cœur ne connaît rien.** Ni HTTP, ni SQL, ni cache, ni horloge, ni logger. Il reçoit des
fonctions, il retourne des valeurs. Il se teste en microsecondes, sans conteneur.

**2. Le nombre de frontends est un non-sujet.** Une surface est un adaptateur primaire branché sur
les mêmes cas d'usage. En ajouter une ne modifie aucun fichier du cœur — et la CI le vérifie.

## Démarrage

```bash
git config core.hooksPath .githooks   # garde-fou anti-push direct sur le tronc
task init                             # .env + outillage
task check                            # fmt · vet · lint · arch · test · vuln
```

**Docker n'est pas requis** pour développer : `go test ./...` sans tag n'exige aucun service.
Les niveaux qui en ont besoin (`-tags=integration`, `-tags=e2e`) sont fournis par la CI.

Avec Docker :

```bash
task up      # Postgres · Redis · Jaeger · Mailpit, puis migrations
task run     # API HTTP        → http://localhost:8080/docs
task run:worker
```

## Repartir de ce socle

```bash
task rename -- github.com/{org}/{projet}
```

Le chemin de module est la **seule** valeur nominative du dépôt. Aucun pseudo, aucune équipe, aucun
`CODEOWNERS` : les contraintes portent sur des **règles**, vérifiées par la CI, pas sur des
personnes. Le socle fonctionne à un contributeur comme à vingt.

Ensuite : supprimer `internal/modules/user_registration/` (rien d'autre n'en dépend) et créer sa
propre feature sur le même patron.

## Structure

```
rules/                     règlement d'ingénierie — FAIT FOI
documentation/adr/         décisions d'architecture — FONT FOI
CLAUDE.md                  amorçage : à lire en premier

cmd/{server,worker,cli}    composition root — le seul code qui connaît tout
config/                    environnement, immuable, validé au démarrage
internal/pkg/              primitives sans dépendance : result · fp · middleware
internal/infrastructure/   socle technique sans métier : db · cache · http · telemetry
internal/modules/{f}/
  ├── domain/              PUR — value objects, règles, erreurs, événements
  ├── ports/               types fonction UNIQUEMENT
  ├── application/         pipeline de cas d'usage + décorateurs
  ├── adapters/primary/    http · cli · events — une surface par dossier
  ├── adapters/secondary/  postgres · outbox · mailer
  └── module.go            composition root local
migrations/                SQL versionné, rétro-compatible N-1
api/openapi.yaml           généré depuis le code — jamais édité à la main
tests/{e2e,perf}           tags `e2e` — hors du `go test ./...` par défaut
```

## Ce qui rend le cadre tenable

Une règle non outillée n'existe pas. Chaque contrainte a son garde :

| Contrainte | Garde |
|---|---|
| Le cœur n'importe ni transport, ni persistance, ni logger | `arch-go` · `depguard` |
| Une feature n'importe pas une autre feature | `arch-go` |
| Un port est un type fonction, pas une interface | `arch-go` |
| Fonctions courtes, peu de paramètres, complexité bornée | `funlen` · `cyclop` · `arch-go` |
| Aucune erreur ignorée, aucun `switch` non exhaustif | `errcheck` · `exhaustive` |
| Aucun état global, aucune `func init()` | `gochecknoglobals` · `gochecknoinits` |
| Aucun ORM | `depguard` |
| Aucun secret dans l'historique | `gitleaks` |
| Aucune vulnérabilité connue | `govulncheck` · CodeQL |
| Couverture ≥ 70 % global, ≥ 90 % sur le cœur | CI, cliquets |
| Toucher au règlement exige un ADR | CI, job `inertia` |
| Aucune dette dissimulée en `TODO` | CI, job `inertia` |
| Aucun commit direct sur le tronc | crochet `pre-push` + ruleset serveur |

## Stack

| Couche | Choix | Pourquoi |
|---|---|---|
| Routage | `chi` | 100 % `http.Handler` — réversible en une journée ([ADR 008](documentation/adr/008-chi-huma-plutot-qu-un-framework.md)) |
| Contrat | `huma` v2, *code-first* | `api/openapi.yaml` généré ; les SDK clients en découlent |
| Persistance | `pgx` v5, SQL explicite | Aucun ORM : ils fuient dans le domaine |
| Asynchrone | outbox transactionnel | Ni perte, ni fantôme ([ADR 006](documentation/adr/006-outbox-transactionnel.md)) |
| Câblage | composition manuelle | Vérifiée par le compilateur ([ADR 004](documentation/adr/004-composition-manuelle-sans-conteneur-di.md)) |
| Observabilité | OpenTelemetry + `slog` | Traces, métriques et logs reliés par `trace_id` |

## État réel — vérifié le 2026-07-25

Le dépôt distingue explicitement trois niveaux, et ne les confond jamais :

- **Prouvé localement** : compilation, `go vet`, tests unitaires `-race -shuffle=on`, `arch-go`,
  `golangci-lint`.
- **Écrit, non prouvé sur la machine de référence** (Docker absent) : migrations, tests
  d'intégration et de bout en bout, images conteneur. Exécutés par la CI.
- **Écrit, jamais déployé** : `deploy-uat.yml` et `deploy-production.yml` n'ont jamais tourné. Ils
  exigent des secrets et un hôte qui n'existent pas encore.

`user_registration` est un exemple de référence complet, pas un besoin métier.

## Documentation

| Je cherche | C'est ici |
|---|---|
| Par où commencer | [`CLAUDE.md`](CLAUDE.md) |
| Le règlement | [`rules/README.md`](rules/README.md) |
| Ce qui est interdit | [`rules/interdictions.md`](rules/interdictions.md) |
| La barre pour livrer | [`rules/definition-of-done.md`](rules/definition-of-done.md) |
| Pourquoi telle décision | [`documentation/adr/`](documentation/adr/README.md) |
| Nommer branche, commit, fichier | [`documentation/process/NOMENCLATURE.md`](documentation/process/NOMENCLATURE.md) |
| Contribuer | [`rules/workflow-git.md`](rules/workflow-git.md) |
| Signaler une faille | [`SECURITY.md`](SECURITY.md) |
