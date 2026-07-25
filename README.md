# go-hexa — socle Go hexagonal, modulaire, fonctionnel, multi-frontend

Un **socle Go réutilisable** en cours de construction : architecture hexagonale modulaire,
programmation fonctionnelle, exposé à **N frontends simultanés** (web, mobile, CLI, événements).

Ce n'est pas une application, et ce n'est pas non plus un modèle de projet à copier une fois. La
cible est un **framework** : un noyau de modules réutilisables, un générateur (`hexa new`), et un
règlement outillé qui empêche la forme de se dégrader. Le chemin y menant est décrit dans
[`documentation/technique/parite-frameworks.md`](documentation/technique/parite-frameworks.md).

> **État d'avancement.** Le socle compile et ses modules noyau sont testés, mais **aucun binaire
> n'existe encore** : ni serveur HTTP, ni worker, ni CLI. Le relevé factuel, daté et sans
> complaisance, est dans [`CLAUDE.md`](CLAUDE.md) § « État réel du dépôt ». Il fait foi sur les
> faits — pas ce README.

## Deux propriétés, et tout le reste en découle

**1. Le cœur ne connaît rien.** Ni HTTP, ni SQL, ni cache, ni horloge, ni logger. Il reçoit des
fonctions, il retourne des valeurs. Il se teste en microsecondes, sans conteneur.

**2. Le nombre de frontends est un non-sujet.** Une surface est un adaptateur primaire branché sur
les mêmes cas d'usage. En ajouter une ne modifie aucun fichier du cœur.

## Le vocabulaire, en quatre mots

Le sens de ces mots est fixé par l'[ADR 012](documentation/adr/012-anatomie-d-un-module-et-pilotes.md)
et par [`rules/references.md`](rules/references.md). Le mot **service** est proscrit : il désigne
trois choses différentes selon l'interlocuteur.

| Mot | Sens | Où |
|---|---|---|
| **module noyau** | capacité technique réutilisable, fournie par le socle | `internal/core/{nom}/` |
| **module métier** | bounded context d'une application | `internal/modules/{nom}/` |
| **pilote** | une implémentation interchangeable d'un module | `.../drivers/{nom}/` |
| **surface** | un frontend servi — web, mobile, CLI, événements | `.../adapters/primary/{nom}/` |

Les deux sortes de modules ont **la même anatomie**. Un module métier ne connaît aucun autre module
métier ; il consomme les ports du noyau.

## Zéro prérequis d'infrastructure

Chaque module a un pilote **sans aucune dépendance externe**, choisi par défaut :

| Module noyau | Défaut | Aussi disponible |
|---|---|---|
| `outbox` | `memory` | `postgres` |
| `idempotency` | `memory` | `postgres` · `redis` |
| `dynconf` | `file` | `postgres` |
| `audit` | `log` | `postgres` |
| `storage` | `disk` | — |
| `scheduler` | `cron-inproc` | `advisory-lock` |

Avec cette configuration, rien n'est requis : ni base, ni Redis, ni Docker. Un test verrouille cette
promesse — tous les modules actifs, tous sur leur pilote par défaut, ne doivent exiger aucun service.

**Chaque pilote documente ses NON-garanties** en tête de paquet. Un pilote en mémoire ne survit pas à
un redémarrage et ne partage rien entre répliques : c'est écrit, en majuscules, là où on le branche.

La promesse porte sur l'**infrastructure**, pas sur les secrets. Une variable reste obligatoire,
`SECURITY_ENCRYPTION_KEY`, et elle n'aura jamais de valeur par défaut — une clé de chiffrement par
défaut chiffrerait les données de tout le monde avec une valeur publiquement connue.

## Démarrage

```bash
git config core.hooksPath .githooks   # garde-fou anti-push direct sur le tronc
task init                             # .env + outillage
task check                            # fmt · vet · lint · arch · test · vuln
```

**Docker n'est pas requis** pour développer : `go test ./...` sans tag n'exige aucun service. Les
niveaux qui en ont besoin (`-tags=integration`, `-tags=e2e`) sont fournis par la CI.

> ⚠️ `task run` et `task up` sont écrits mais **ne peuvent pas encore fonctionner** : `cmd/` est
> vide. Le premier binaire est suivi par les issues #3 à #8, #10.

## Repartir de ce socle

```bash
task rename -- github.com/{org}/{projet}
```

Le chemin de module est la **seule** valeur nominative du dépôt. Aucun pseudo, aucune équipe, aucun
`CODEOWNERS` : les contraintes portent sur des **règles** vérifiées par la CI, pas sur des personnes.
Le socle fonctionne à un contributeur comme à vingt.

Ensuite : supprimer `internal/modules/user_registration/` — rien d'autre n'en dépend — et créer son
propre module métier sur le même patron.

## Structure

```
rules/                       règlement d'ingénierie — FAIT FOI
documentation/adr/           décisions d'architecture — FONT FOI
CLAUDE.md                    amorçage et état réel : à lire en premier

config/*.yaml                configuration par groupes, secrets par ${VAR} uniquement
cmd/{server,worker,cli}      composition root — le seul code qui connaît tout   ⟨absent⟩
internal/pkg/                primitives sans dépendance : result · fp · pagination · middleware
internal/infrastructure/      socle technique sans métier : db · cache · http · telemetry · security
internal/contracts/           langage publié : ce que les modules s'échangent, sans s'importer
internal/core/{nom}/          MODULE NOYAU — fourni par le socle
internal/modules/{nom}/       MODULE MÉTIER — écrit par l'application
  ├── domain/                 PUR — objets valeur, règles, erreurs, événements
  ├── ports/                  types fonction UNIQUEMENT
  ├── application/            pipeline de cas d'usage + décorateurs, sans I/O
  ├── drivers/{nom}/          une implémentation interchangeable du module
  ├── adapters/primary/       http · cli · events — une surface par dossier
  ├── adapters/secondary/     postgres · mailer
  ├── tests/                  boîte noire, un fichier par test
  └── module.go               composition root local — le SEUL à connaître les pilotes
migrations/                  SQL versionné, rétro-compatible N-1                 ⟨absent⟩
api/openapi.yaml             généré depuis le code — jamais édité à la main      ⟨absent⟩
tests/{e2e,perf}             tags `e2e` — hors du `go test ./...` par défaut
```

## Ce qui rend le cadre tenable

Une règle non outillée n'existe pas. Chaque contrainte a son garde — et la colonne « éprouvé » dit
si le garde a **déjà tourné** sur ce dépôt, parce qu'un garde jamais exécuté ne garde rien.

| Contrainte | Garde | Éprouvé |
|---|---|---|
| Le cœur n'importe ni transport, ni persistance, ni logger | `arch-go` · `depguard` | non |
| Un module métier n'importe pas un autre module métier | `arch-go` | non |
| Un module noyau ne connaît aucun module métier | `arch-go` | non |
| Un port est un type fonction, pas une interface | `arch-go` | non |
| Fonctions courtes, peu de paramètres, complexité bornée | `funlen` · `cyclop` · `arch-go` | non |
| Aucune erreur ignorée, aucun `switch` non exhaustif | `errcheck` · `exhaustive` | non |
| Aucun état global, aucune `func init()` | `gochecknoglobals` · `gochecknoinits` | non |
| Aucun ORM | `depguard` | non |
| Le code compile, `go vet` passe, les tests passent | `go build` · `go vet` · `go test` | **oui** |
| La configuration livrée charge et n'exige aucun service | tests de `internal/config/tests/` | **oui** |
| Aucun secret dans l'historique | `gitleaks` | CI seulement |
| Aucune vulnérabilité connue | `govulncheck` · CodeQL | CI seulement |
| Couverture ≥ 70 % global, ≥ 90 % sur le cœur | CI, cliquets | CI seulement |
| Toucher au règlement exige un ADR | CI, job `inertia` | CI seulement |
| Aucune dette dissimulée en `TODO` | CI, job `inertia` | CI seulement |
| Aucun commit direct sur le tronc | crochet `pre-push` | **oui** |

`golangci-lint` et `arch-go` sont installés et configurés strictement, mais **n'ont jamais été
exécutés** sur l'état courant. Des violations sont donc à attendre. C'est la PREMIÈRE action de la liste
de `CLAUDE.md`, et le dire vaut mieux que de laisser croire à un vert qui n'existe pas.

## Stack

| Couche | Choix | Pourquoi |
|---|---|---|
| Routage | `chi` | 100 % `http.Handler` — réversible en une journée ([ADR 008](documentation/adr/008-chi-huma-plutot-qu-un-framework.md)) |
| Contrat | `huma` v2, *code-first* | `api/openapi.yaml` généré ; les SDK clients en découlent |
| Persistance | `pgx` v5, SQL explicite | Aucun ORM : ils fuient dans le domaine ([ADR 009](documentation/adr/009-strategie-d-acces-aux-donnees.md)) |
| Moteur de base | **aucun imposé** | `postgres` est un pilote parmi d'autres (issue #36) |
| Asynchrone | outbox transactionnel | Ni perte, ni fantôme ([ADR 006](documentation/adr/006-outbox-transactionnel.md)) |
| Câblage | composition manuelle | Vérifiée par le compilateur ([ADR 004](documentation/adr/004-composition-manuelle-sans-conteneur-di.md)) |
| Observabilité | OpenTelemetry + `slog` | Traces, métriques et logs reliés par `trace_id` |

## La doc ne mente jamais sur l'état réel

C'est une règle d'or, pas une intention. Trois niveaux, jamais confondus :

- **prouvé localement** — la commande a tourné sur la machine de référence et son code de retour a
  été vérifié ;
- **écrit, non prouvé** — le code existe et compile ; rien ne l'a exécuté ;
- **jamais déployé** — `deploy-uat.yml` et `deploy-production.yml` n'ont jamais tourné.

Un document qui coche « ✅ testé » sans test est pire qu'aucun document. Le relevé complet, avec sa
date, est dans [`CLAUDE.md`](CLAUDE.md).

Deux conséquences à ne pas oublier : `user_registration` est un exemple de référence **incomplet** —
son cœur est couvert par 31 tests, mais il n'a encore aucun adaptateur, donc aucune surface ne
l'appelle — et le dépôt **n'a aucune authentification ni autorisation**. Il ne faut donc jamais
parler de « zéro faille » à son sujet.

## Documentation

| Je cherche | C'est ici |
|---|---|
| Par où commencer, et l'état réel | [`CLAUDE.md`](CLAUDE.md) |
| Le règlement | [`rules/README.md`](rules/README.md) |
| Ce qui est interdit | [`rules/interdictions.md`](rules/interdictions.md) |
| La barre pour livrer | [`rules/definition-of-done.md`](rules/definition-of-done.md) |
| Pourquoi telle décision | [`documentation/adr/`](documentation/adr/README.md) |
| L'anatomie d'un module | [`ADR 012`](documentation/adr/012-anatomie-d-un-module-et-pilotes.md) |
| Les modules noyau prévus | [`documentation/technique/modules-noyau.md`](documentation/technique/modules-noyau.md) |
| Le catalogue des pilotes | [`documentation/technique/pilotes.md`](documentation/technique/pilotes.md) |
| Ce qu'un framework mûr offre | [`documentation/technique/parite-frameworks.md`](documentation/technique/parite-frameworks.md) |
| Nommer branche, commit, fichier | [`documentation/process/NOMENCLATURE.md`](documentation/process/NOMENCLATURE.md) |
| Contribuer | [`rules/workflow-git.md`](rules/workflow-git.md) |
| Signaler une faille | [`SECURITY.md`](SECURITY.md) |
