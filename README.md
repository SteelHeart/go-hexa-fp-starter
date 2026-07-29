# go-hexa — socle Go hexagonal, modulaire, fonctionnel, multi-frontend

Un **socle Go réutilisable** en cours de construction : architecture hexagonale modulaire,
programmation fonctionnelle, exposé à **N frontends simultanés** (web, mobile, CLI, événements).

Ce n'est pas une application, et ce n'est pas non plus un modèle de projet à copier une fois. La
cible est un **framework** : un noyau de modules réutilisables, un générateur (`hexa new`), et un
règlement outillé qui empêche la forme de se dégrader. Le chemin y menant est décrit dans
[`documentation/technique/parite-frameworks.md`](documentation/technique/parite-frameworks.md).

> **⚠️ Lisible, pas réutilisable.** Ce dépôt est **public en lecture** et reste sous
> [`LICENSE`](LICENSE) **tous droits réservés** : aucun droit d'usage, de copie, de modification ni
> de redistribution n'est accordé. Le publier sert la transparence et l'analyse de sécurité, pas
> l'adoption. **Ne le forkez pas, ne l'intégrez pas** — ni dans un produit, ni dans un jeu de
> données d'entraînement. Une ouverture est envisagée, sans date et sans engagement.
>
> C'est une décision, pas un oubli. Elle est prise et tracée en
> [#113](https://github.com/SteelHeart/go-hexa-fp-starter/issues/113).

> **État d'avancement.** Le serveur HTTP, la ligne de commande et le dépileur d'événements existent
> et tournent — la chaîne complète `inscription → outbox → dépileur → relais → garde d'idempotence
> → notification` a été exercée sur les binaires réels. **`auth` et `notification` existent
> désormais** ; il manque le **multi-locataire**, le **paiement** et la **limitation de débit**,
> décrits sans aucun code. Le relevé factuel, daté et sans complaisance, est dans
> [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md) § « État réel du dépôt ». Il fait foi sur les faits — pas ce README.
>
> Les écarts connus entre ce qui est écrit et ce qui est sont listés dans
> [`documentation/process/AUDIT_CONFORMITE.md`](documentation/process/AUDIT_CONFORMITE.md).

## En trente secondes, sans rien installer

```bash
export SECURITY_ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)
go run ./cmd/server
```

```bash
curl -s -X POST localhost:8080/v1/users \
  -H 'content-type: application/json' \
  -d '{"email":"Alice@Example.COM ","password":"correct horse battery staple"}'
```

```json
{
  "user_id": "019f9b46-3aec-735a-977d-129192ef130f",
  "email": "alice@example.com",
  "status": "pending",
  "created_at": "2026-07-25T21:54:58.924Z"
}
```

Ni base, ni Redis, ni Docker. Trois choses se lisent dans cette réponse :
l'identifiant est un **UUID v7** — ordonné dans le temps, donc utilisable en clé primaire sans
fragmenter l'index ; l'adresse est **normalisée** par le domaine, espace et casse compris ; et le
compte naît **`pending`**, jamais actif, parce que l'adresse n'est pas encore prouvée.

La documentation interactive est sur `/docs`, le contrat sur `/openapi.json` et `/openapi.yaml`.

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

La règle vaut aussi pour les modules **métier** : `user_registration` a son pilote `memory`, et c'est
lui le défaut. Un module métier dont le seul pilote exigerait PostgreSQL briserait cette promesse au
premier module écrit — c'est-à-dire au moment exact où on l'éprouve.

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

Deux binaires :

| Commande | Rôle | Prérequis |
|---|---|---|
| `go run ./cmd/server` | surfaces HTTP | aucun |
| `go run ./cmd/worker` | dépilage de l'outbox vers le courtier | une outbox **partagée entre processus** |

> ⚠️ Le dépileur **refuse de démarrer** sur le pilote `outbox: memory`, et c'est voulu : ce pilote
> vit dans le processus, donc un worker lancé séparément dépilerait son propre magasin — vide —
> pendant que les événements du serveur resteraient dans la mémoire du serveur. Il tournerait sans
> rien publier **et sans aucune erreur**. Un composant silencieusement inerte est le seul défaut qui
> ne se signale jamais.

## Repartir de ce socle

```bash
task rename -- github.com/{org}/{projet}
```

Le chemin de module est la **seule** valeur nominative du dépôt. Aucun pseudo, aucune équipe, aucun
`CODEOWNERS` : les contraintes portent sur des **règles** vérifiées par la CI, pas sur des personnes.
Le socle fonctionne à un contributeur comme à vingt.

Ensuite : supprimer `internal/modules/user_registration/` et créer son propre module métier sur le
même patron. **Un seul fichier du socle le nomme** — `cmd/server/main.go`, qui le monte et l'expose.
C'est le composition root, et c'est précisément son rôle de connaître les modules ; aucun autre code
ne le mentionne.

`user_registration` est la **tranche de référence**, pas l'application. Elle existe pour montrer la
forme complète — domaine pur, ports en types fonction, pipeline composé, pilotes interchangeables,
adaptateurs par surface — parce que c'est cette forme qui sera copiée pour écrire `billing` ou
`crm`. Tout dossier qui lui manquerait serait reproduit comme « pas nécessaire ».

## Structure

```
rules/                       règlement d'ingénierie — FAIT FOI
documentation/adr/           décisions d'architecture — FONT FOI
documentation/AMORCAGE.md                    amorçage et état réel : à lire en premier

config/*.yaml                configuration par groupes, secrets par ${VAR} uniquement
cmd/{server,worker}          composition root — le seul code qui connaît tout
cmd/cli                      ⟨absent⟩
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
migrations/{moteur}/         SQL versionné, rétro-compatible N-1 · `postgres/` seul aujourd'hui
deploy/postgres/             provision.sql — les RÔLES, exécuté une fois, hors goose
api/openapi.yaml             ⟨absent — le contrat est SERVI, pas encore versionné⟩
tests/{e2e,perf}             tags `e2e` — hors du `go test ./...` par défaut
```

## Ce qui rend le cadre tenable

Une règle non outillée n'existe pas. Chaque contrainte a son garde — et la colonne « éprouvé » dit
si le garde a **déjà tourné** sur ce dépôt, parce qu'un garde jamais exécuté ne garde rien.

| Contrainte | Garde | Éprouvé |
|---|---|---|
| Le cœur n'importe ni transport, ni persistance, ni logger | `arch-go` · `depguard` | **oui** |
| Un module métier n'importe pas un autre module métier | `arch-go` | **oui** |
| Un module noyau ne connaît aucun module métier | `arch-go` | **oui** |
| Un port est un type fonction, pas une interface | `arch-go` | **oui** |
| Un binaire n'exporte que `main` | `arch-go` | **oui** |
| Fonctions courtes, peu de paramètres, complexité bornée | `funlen` · `cyclop` · `arch-go` | **oui** |
| Aucune erreur ignorée, aucun `switch` non exhaustif | `errcheck` · `exhaustive` | **oui** |
| Aucun état global, aucune `func init()` | `gochecknoglobals` · `gochecknoinits` | **oui** |
| Aucun ORM | `depguard` | **oui** |
| Le code compile, `go vet` passe, les tests passent | `go build` · `go vet` · `go test` | **oui** |
| La configuration livrée charge et n'exige aucun service | tests de `internal/config/tests/` | **oui** |
| Aucun commit direct sur le tronc | crochet `pre-push` | **oui** |
| Un module n'atteint pas le schéma SQL d'un autre | `deploy/postgres/verify.sql` | CI seulement |
| Le journal d'audit refuse `UPDATE` et `DELETE` | job CI `migrations` | CI seulement |
| Le retour arrière d'une migration fonctionne | job CI `migrations` (il le **rejoue**) | CI seulement |
| Aucun secret dans l'historique | `gitleaks` | CI seulement |
| Couverture ≥ 70 % global, ≥ 90 % sur le cœur | CI, cliquets | **NON — 52,4 % mesurés** |
| Toucher au règlement exige un ADR | CI, job `inertia` | CI seulement |
| Aucune dette dissimulée en `TODO` | CI, job `inertia` | CI seulement |
| Aucune vulnérabilité connue | `govulncheck` · CodeQL | **oui** |

**`golangci-lint` rend 0 signalement** (~50 analyseurs, parti de 239), **`arch-go` 18 règles sur 18
avec 100 % de couverture**, et **`govulncheck` 0 vulnérabilité**. Tous ont réellement tourné, dans
l'enchaînement `task check`, et le code de retour a été vérifié — pas seulement la sortie.

Le dépôt épingle **`go 1.25.12`** dans `go.mod`. Avec `GOTOOLCHAIN=auto` — le défaut — Go télécharge
la chaîne demandée : aucune installation système n'est nécessaire, et la correction vaut pour tout
le monde, CI comprise. C'est ce qui a fermé les 20 vulnérabilités de la bibliothèque standard que
portait la chaîne précédente.

⚠️ **La couverture réelle est de 52,4 %, sous le seuil de 70 % que ce tableau annonce.** Elle était
mesurée à **3,6 %** jusqu'au 2026-07-26 : les tests étant en boîte noire dans `{paquet}/tests/`, le
profil n'attribuait la couverture qu'au paquet de test. `-coverpkg=./...` corrige la mesure — pas le
manque. `messaging`, `modulebus`, `httpserver`, `telemetry` et `cache` compilent sans aucun test.
Le seuil reste à 70 % : on couvre, on ne baisse pas la barre.

> ⚠️ **`task check` ne peut pas être vert depuis `C:\xampp\htdocs\`.** Sur cette machine, aucun
> binaire Go n'y a le droit de créer un fichier — le shell si. `go test -coverprofile` échoue donc,
> et l'étape `test` avec lui. Ce n'est pas un défaut du dépôt : `go test ./...` sans couverture est
> vert, et les cinq autres étapes passent. Sortir le dépôt du répertoire web de XAMPP, ou travailler
> sous WSL. Voir friction **F008**.

## Stack

| Couche | Choix | Pourquoi |
|---|---|---|
| Routage | `chi` | 100 % `http.Handler` — réversible en une journée ([ADR 008](documentation/adr/008-chi-huma-plutot-qu-un-framework.md)) |
| Contrat | `huma` v2, *code-first* | Servi sur `/openapi.{json,yaml}` ; les SDK clients en découlent |
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
date, est dans [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md).

Ces trois avertissements étaient tous devenus **faux** avant d'être corrigés — c'est ce que l'audit
[#107](https://github.com/SteelHeart/go-hexa-fp-starter/issues/107) est venu chercher. Leur version
à jour, et ce qu'ils cachaient :

- ~~« aucune authentification ni autorisation »~~ — **`auth` existe**, avec sa surface HTTP et son
  garde d'autorisation ; l'[ADR 017](documentation/adr/017-authentification-et-autorisation.md)
  tranche que *le jeton authentifie, il n'autorise pas*, donc un droit révoqué est refusé à l'appel
  suivant. ⚠️ **Ne pas en conclure « zéro faille »** : ce module est neuf, il n'a jamais été éprouvé
  ailleurs qu'ici, et rien n'a été audité par un tiers. `GET /v1/users/availability` permet toujours
  d'**énumérer** les adresses enregistrées — acceptable derrière une limitation de débit, et le
  module `ratelimit` **n'existe pas**.
- ~~« personne ne consomme les événements »~~ — la chaîne `inscription → outbox → dépileur → relais
  → garde d'idempotence → notification` a **tourné sur les binaires réels**, en local puis en CI,
  avec l'adresse masquée et le corps non journalisé. ⚠️ Le consommateur est câblé dans `cmd/worker`
  et **non monté en `adapters/primary/events/`** : le troisième adaptateur primaire n'existe pas
  encore ([#9](https://github.com/SteelHeart/go-hexa-fp-starter/issues/9)). Et `notification` n'a
  qu'un pilote `log` — **aucun courriel n'est parti nulle part**
  ([#27](https://github.com/SteelHeart/go-hexa-fp-starter/issues/27)).
- ~~« les pilotes `postgres` n'ont jamais tourné ici »~~ — le niveau `integration` les exerce contre
  un vrai Postgres et un vrai Redis, et le job CI du même nom l'exécute à chaque PR.

## Documentation

| Je cherche | C'est ici |
|---|---|
| Par où commencer, et l'état réel | [`documentation/AMORCAGE.md`](documentation/AMORCAGE.md) |
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
| Les écarts connus entre l'écrit et le réel | [`documentation/process/AUDIT_CONFORMITE.md`](documentation/process/AUDIT_CONFORMITE.md) |

## Licence

[`LICENSE`](LICENSE) — **tous droits réservés**. Le dépôt est **public en lecture** : consultable,
analysable, citable. **Aucun droit d'usage, de copie, de modification ni de redistribution n'est
accordé**, y compris pour l'entraînement de modèles.

C'est un état *source-available*, **choisi** — pas une licence ouverte qu'on aurait oublié de poser.
La transparence sert la revue de sécurité et l'évaluation ; l'adoption n'est pas l'objectif
aujourd'hui. Une ouverture est envisagée, **sans date et sans engagement**, et sera annoncée ici
avant de l'être ailleurs.

Pour un usage nécessitant des droits, ouvrir une issue.
