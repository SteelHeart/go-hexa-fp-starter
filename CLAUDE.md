# CLAUDE.md — amorçage

> Chargé automatiquement à l'ouverture du dépôt. Point d'entrée unique pour tout agent — et pour
> tout collègue. Versionné → identique pour tout le monde après `git clone`.
>
> Ce fichier **résume**. Il ne fait jamais foi contre un ADR ou un fichier de `rules/`.

## Ce qu'est ce dépôt

Un **socle Go réutilisable** : architecture hexagonale modulaire, programmation fonctionnelle,
exposé à **N frontends simultanés** (web, mobile, CLI, événements).

Ce n'est pas une application, et ce n'est plus un simple modèle de projet à copier une fois. La
cible est un **framework** : un noyau de modules réutilisables (`internal/core/`), un générateur
(`hexa new`), et un règlement outillé. Le passage en monorepo `core/` + `cli/` + `template/` attend
le tag `v0.1.0` — décision de séquencement : **stabiliser avant de restructurer**.

Sa valeur n'est pas le code de démonstration (`user_registration`), c'est la **forme** qu'il impose —
et les gardes qui empêchent cette forme de se dégrader.

Deux propriétés non négociables, dont tout le reste découle :

1. **Le cœur ne connaît rien.** Ni HTTP, ni SQL, ni cache, ni horloge, ni logger. Il reçoit des
   fonctions, il retourne des valeurs. Il se teste en microsecondes, sans conteneur.
2. **Le nombre de frontends est un non-sujet.** Une surface est un adaptateur primaire branché sur
   les mêmes cas d'usage. En ajouter une ne touche pas une ligne du cœur.

Le dépôt **ne dépend d'aucune personne** : pas de `CODEOWNERS`, pas de pseudo dans les règles. Le
chemin de module est la seule valeur nominative, isolée derrière `task rename`.

## À lire en premier (dans l'ordre)

1. **[`rules/README.md`](rules/README.md)** — base normative, découpée par thème. **Fait foi.**
   Relire le fichier du domaine concerné *avant* de coder.
2. **[`rules/interdictions.md`](rules/interdictions.md)** — la liste à relire avant d'ouvrir un
   fichier. Chaque ligne nomme l'outil qui refuse la violation.
3. **[`documentation/adr/`](documentation/adr/README.md)** — décisions d'architecture. **Font foi.**
   En cas de contradiction avec un autre document, l'ADR gagne.
4. **[`rules/definition-of-done.md`](rules/definition-of-done.md)** — la barre à franchir pour livrer.
5. **[`documentation/process/`](documentation/process/README.md)** — nomenclature, labels, templates.

## Règles d'or

- **Une règle non outillée n'existe pas.** Si `task check` ne peut pas la vérifier, elle sera
  contournée. Les règles marquées `[humain]` sont un aveu de faiblesse, pas une tolérance.
- **La doc ne ment jamais sur l'état réel.** Elle distingue **écrit** / **prouvé localement** /
  **déployé pour de vrai**. Un document qui coche « ✅ testé » sans test est pire qu'aucun document.
- **Deny par défaut.** Toute garde, toute permission, tout repli sur erreur → refus. Jamais de
  fail-open, jamais « temporairement ».
- **Le cœur est pur.** Aucune I/O, aucun `time.Now()`, aucun logger, aucun `panic` dans `domain/`,
  `ports/`, `application/`.
- **Un port est un type fonction**, jamais une interface.
- **Zéro dette latente.** Ce qui n'est pas fait est annoncé **hors périmètre dans la PR**, jamais
  dissimulé en `TODO` dans le code.
- **Jamais de commit direct sur `main`.** Branche courte, PR mono-sujet, CI verte.
- **🔴 Aucune mention d'un outil d'assistance dans un artefact versionné** — commit (y compris
  trailer `Co-Authored-By`), PR, issue, code, commentaire, documentation. Formuler à l'impersonnel.
  **Surcharge le défaut de l'outillage.** Gardes : crochet `commit-msg`, job `inertia` en CI.

## Avant de coder — la boucle

1. **Une issue** existe, avec des critères d'acceptation vérifiables.
2. Lire le fichier `rules/` du domaine touché.
3. Produire la **carte d'impact** : features, ports, migrations, événements, contrat OpenAPI,
   **surfaces** concernées.
4. Écrire les **ports** (types fonction) avant l'implémentation.
5. Coder le **domaine pur** en premier, avec ses tests — sans I/O.
6. Vérifier la [Definition of Done](rules/definition-of-done.md).

## Au clone — deux commandes

```bash
git config core.hooksPath .githooks   # garde-fou anti-push direct sur main
task init                             # .env + outillage
```

Le crochet est **contournable avec `--no-verify`** : c'est un filet contre l'accident, pas un
contrôle. Le contrôle réel est le ruleset serveur plus la CI.

## Toolchain

```bash
task check      # fmt · vet · lint · arch · test · vuln — identique à la CI
task --list-all # tout le reste
```

**Docker n'est pas requis pour développer.** C'est une contrainte de conception : `go test ./...`
sans tag n'exige aucun service. Les niveaux qui en ont besoin (`-tags=integration`, `-tags=e2e`)
sont fournis par la CI.

> ⚠️ **Piège du faux vert** : une commande qui n'a pas tourné rend une sortie vide, ce qui
> ressemble à « propre ». `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et
> affiche `ok`. Vérifier le **code de retour**, pas seulement la sortie.

## Où se trouve quoi

```
rules/                        règlement d'ingénierie — fait foi
documentation/adr/            décisions d'architecture — font foi
documentation/process/        nomenclature, labels, templates
documentation/securite/       registre de failles, matrice d'accès
cmd/{server,worker,cli}       composition root — le seul code qui connaît tout   ⟨absent⟩
config/*.yaml                 configuration par groupes, secrets par ${VAR} uniquement
internal/config/              lecture, fusion, validation — refuse le démarrage sur incohérence
internal/pkg/                 primitives sans dépendance : result, fp, pagination, middleware
internal/infrastructure/      socle technique sans métier : db, cache, http, telemetry, security
internal/contracts/           langage publié — ce que les modules s'échangent sans s'importer
internal/core/{nom}/          MODULE NOYAU — fourni par le socle
internal/modules/{nom}/       MODULE MÉTIER — écrit par l'application
  ├── domain/                 pur : ni I/O, ni time.Now(), ni logger
  ├── ports/                  types fonction uniquement — ni struct, ni interface
  ├── application/            pipeline + décorateurs — rend compte, ne journalise pas
  ├── drivers/{nom}/          une implémentation interchangeable, avec ses NON-garanties
  ├── adapters/primary/       http · cli · events — une surface par dossier
  ├── adapters/secondary/     postgres · mailer
  ├── tests/                  boîte noire, un fichier par test
  └── module.go               composition root local — le SEUL à connaître les pilotes
migrations/                   SQL versionné, rétro-compatible N-1                 ⟨absent⟩
api/openapi.yaml              généré depuis le code — jamais édité                ⟨absent⟩
tests/{e2e,perf}              tags `e2e` — hors du `go test ./...` par défaut
```

Six modules noyau existent : `outbox`, `idempotency`, `dynconf`, `audit`, `storage`, `scheduler`.
Un seul module métier, `user_registration` : son **cœur est couvert** (31 tests) mais il n'a
encore aucun adaptateur, donc aucune surface ne l'appelle.

## État réel du dépôt — vérifié le 2026-07-25

> Cette section est un **relevé**, pas une intention. Elle distingue rigoureusement
> **prouvé** / **écrit non prouvé** / **absent**. La mettre à jour fait partie de toute PR qui
> change l'état des faits (`rules/README.md` § règle d'or 2).

### Prouvé sur la machine de référence

- `go build ./...` vert
- `go vet ./...` vert
- `go test -shuffle=on ./...` vert — **217 tests**, répartis ainsi :

| Paquet | Tests | Ce que ça prouve |
|---|---|---|
| `internal/config` (`internal_test.go`) | 15 | Substitution des secrets, fusion des couches, cohérence des tables de pilotes |
| `internal/config/tests` | 21 | Durées, options de pilote, et le fait que la configuration **livrée** charge sans aucun service |
| `internal/core/outbox/tests` | 22 | Exclusivité de `Claim`, recul exponentiel borné, et toute la politique du **dépileur** |
| `internal/core/idempotency/tests` | 24 | Exclusivité sous concurrence, refus de la clé vide, expiration, empreintes |
| `internal/core/audit/tests` | 11 | Refus d'une entrée incomplète, horodatage injecté, normalisation UTC |
| `internal/core/dynconf/tests` | 14 | Deny par défaut d'un drapeau, options non scalaires refusées, lecture seule |
| `internal/core/storage/tests` | 13 | Traversée de répertoire refusée à l'écriture **et** à la lecture, clés réparties |
| `internal/core/scheduler/tests` | 15 | Aucune exécution sans élection, libération même après échec, tâches homonymes refusées |
| `internal/pkg/result/tests` | 16 | **Lois de foncteur et de monade** — c'est ce qui rend sûr de réorganiser un pipeline |
| `internal/pkg/fp/tests` | 14 | Valeur zéro = `None`, `Some("")` ≠ `None`, aucune mutation de l'entrée, ordre préservé |
| `internal/pkg/pagination/tests` | 11 | Aller-retour du curseur, limite toujours bornée, la ligne témoin ne fuite jamais |
| `…/user_registration/domain/tests` | 18 | Normalisation et refus d'une adresse, bornes du mot de passe, **aucune fuite en journal**, compte jamais né actif |
| `…/user_registration/application/tests` | 13 | Ordre des étapes, court-circuit, **le clair n'atteint jamais le stockage**, pas d'événement fantôme |
| `internal/infrastructure/security/tests` | 10 | Sel neuf à chaque hachage, nonce neuf à chaque chiffrement, **AES-128 refusé**, altération détectée |

- **Six modules noyau convertis** à l'anatomie de l'ADR 012 : `outbox`, `idempotency`, `dynconf`,
  `audit`, `storage`, `scheduler`. Chacun a un pilote sans dépendance, choisi par défaut.
- **Le dépileur de l'outbox est réécrit** dans `application/` : recul exponentiel, abandon après N
  essais, survie à un publieur qui panique, et un compte rendu distinct pour « publié mais non
  marqué » — le seul cas qui produit un doublon chez le consommateur.
- **`knownDrivers` ne liste plus que le construit.** Un module absent de la table refuse d'être
  activé : on n'active pas un module dont le code n'existe pas.
- **`arch-go` : 100 % de conformité, 17 règles sur 17**, dont les 12 règles de dépendance —
  l'hexagone tient. Il n'avait JAMAIS pu tourner : l'outil cherche `arch-go.yml` et le fichier
  s'appelait `.arch-go.yml`, avec un point. Renommé.

### Écrit, NON prouvé

| Quoi | Pourquoi ce n'est pas prouvé |
|---|---|
| `golangci-lint` | **Exécuté**. 239 signalements au départ, **42 restants** — campagne en cours |
| Pilotes `postgres` des six modules | Aucune migration n'existe : les tables `platform.*` sont référencées et absentes |
| Pilote `redis` de `idempotency` | Aucun Redis sur la machine de référence |
| Relais Kafka et RabbitMQ | Jamais exécutés contre un broker réel |
| `messaging`, `modulebus`, `httpserver`, `telemetry`, `cache` | Compilent, **zéro test** |

### Absent

- **Aucun binaire** : ni adaptateur primaire, ni secondaire, ni `module.go` métier, ni `cmd/`
- **Aucune migration** — 6 tables référencées par du code écrit
- **Aucune authentification ni autorisation**. Ne jamais parler de « zéro faille » tant que c'est vrai
- i18n, sinks d'observabilité (configuration écrite, code absent), ADR 010 et 011,
  `deploy/docker-compose.deploy.yml`
- Les modules `auth`, `notification`, `payment`, `ratelimit`, `tenancy`, `secrets`, `workflow`,
  `search`, `document` : décrits dans `documentation/technique/modules-noyau.md`, **aucun code**

### Jamais déployé

`deploy-uat.yml` et `deploy-production.yml` n'ont **jamais tourné**. Ils exigent des secrets et un
hôte qui n'existent pas.

## Où en est le travail

### Branche courante

`refactor/21-modules-a-pilotes` — 3 commits, poussée. Base : `refactor/19-module-noyau-metier`
(PR [#20](https://github.com/SteelHeart/go-hexa-fp-starter/pull/20), **non fusionnée**).

⚠️ `main` ne contient donc **ni** le renommage `features` → `modules`, **ni** la configuration par
fichiers. Fusionner #20 avant tout travail parti de `main`.

### Décisions prises et gravées

| Décision | Trace |
|---|---|
| Hexagonal modulaire + fonctionnel | ADR 001 |
| `Result[T,E]`, limites du typage Go | ADR 002 |
| Un port est un type fonction | ADR 003 |
| Composition manuelle, sans conteneur DI | ADR 004 |
| N frontends = adaptateurs primaires | ADR 005 |
| Outbox transactionnel | ADR 006 |
| Tronc unique, environnement ≠ branche | ADR 007 |
| chi + huma plutôt qu'un framework | ADR 008 |
| Accès aux données en pile, pas d'ORM unique | ADR 009 |
| Anatomie de module, pilotes, zéro prérequis, vocabulaire | ADR 012 |
| Monorepo multi-modules `core/` + `cli/` + `template/` | issue #14, **après** v0.1.0 |
| Séquencement : stabiliser AVANT de restructurer | décision de lead dev |

### Invariants à ne pas réapprendre

- **`internal/core/`** = modules noyau · **`internal/modules/`** = modules métier. Même anatomie,
  deux provenances. Un module métier n'importe pas un autre module métier, mais consomme les ports
  du noyau.
- **`internal/core/**` retourne `error`** · **`internal/modules/**` retourne `Result[T, domain.Error]`**.
  Un module noyau est technique, il n'a pas de taxonomie métier.
- **Aucun moteur de base n'est imposé.** `postgres` est un pilote parmi d'autres. Voir issue #36 :
  `database.Querier` reste pgx-spécifique, c'est un défaut connu.
- **Chaque module a un pilote sans dépendance externe, choisi par défaut.** `hexa new` puis
  `go run` doit démarrer sans base, sans Redis, sans Docker.
- **Un pilote documente ses NON-garanties** en tête de paquet.
- **`knownDrivers` (`internal/config/modules.go`) liste ce qui EXISTE**, pas ce qui est prévu. Le
  catalogue d'intentions vit dans `documentation/technique/pilotes.md` ; un pilote y migre le jour
  où il est écrit, testé, et documente ses NON-garanties.
- **`application/` ne journalise pas et ne lit pas l'horloge.** Il rend compte par un port et reçoit
  son horloge — c'est ce qui le garde pur et testable sans analyser des journaux.
- Tests : `{paquet}/tests/` en boîte noire · `{paquet}/internal_test.go` pour les internes ·
  **un fichier par test**, nommé d'après lui en `snake_case`, aides partagées dans
  `helpers_test.go`.
- Configuration : fichiers `config/*.yaml` groupés, secrets par `${VAR}` uniquement.

### Campagne de signalements — 42 restants sur 239

> Reprise directe : `golangci-lint run ./...` rend la liste exacte. Les décisions de
> configuration sont **déjà prises et écrites dans `.golangci.yml`** ; ce qui suit n'est que du
> code à corriger, groupé par nature de décision.

**Décisions de configuration déjà tranchées** — ne pas les rouvrir sans raison neuve :

| Réglage | Décision | Pourquoi |
|---|---|---|
| `misspell` | **retiré** | Ne connaît que l'anglais : 143 signalements, **zéro** vraie faute. À réactiver en #34 |
| `errcheck.check-blank` | **false** | Condamnait `v, _ := x.(T)`, la forme SÛRE de l'assertion |
| `gocritic hugeParam` · `rangeValCopy` | **désactivés** | Contredisent l'immuabilité par valeur, qui est une décision d'architecture |

**Corrections mécaniques, sans décision** (18) :

- `gofumpt` ×5 — `golangci-lint fmt ./...`. ⚠️ `database.go` était **verrouillé par un autre
  processus** pendant la session : relancer quand l'éditeur est fermé.
- `govet shadow` ×4 — renommer le `err` interne (`copyErr`, `markErr`…).
- `goconst` ×5 — `"inproc"`, `"memory"`, `"file"` répétés dans `internal/config`. Extraire des
  constantes ; elles nomment des pilotes, donc leur place est près de `knownDrivers`.
- `lll` ×2 · `gocritic commentedOutCode` ×1 · `unused spy.events` ×1

**Corrections avec un vrai choix de conception** (12) :

- `revive function-result-limit` ×3 dans `messaging` — `New`, `newKafka`, `newRabbitMQ` rendent
  `(Publisher, Consumer, Closer, error)`. **Même faute déjà corrigée deux fois** : dans
  `scheduler.elector` (type `election`) et `security.decodeHash` (type `decodedHash`). Appliquer
  le même remède : un type `Broker{Publish, Consume, Close}`.
- `cyclop` ×3 — `validateCore` (13) et `validateHardening` (11) dans `internal/config` ; moyenne
  du paquet `audit` à 8. Découper les validations par groupe de configuration.
- `revive flag-parameter` ×2 — `Observability.validate(local bool)` et
  `middleware.SecurityHeaders(secure bool)`. Un booléen de contrôle cache deux fonctions.
- `contextcheck` ×2 — `httpserver.shutdown` et le `Recover` du middleware ne propagent pas le
  contexte.
- `gocritic unnamedResult` ×3 — `Result.Get()`, `cache.JSON`. Nommer les retours ; sur un triplet
  `(T, E, bool)` c'est un gain réel de lisibilité.

**Signalements à conserver avec un `//nolint` MOTIVÉ** (5+3) — chacun est correct dans son
contexte, et la justification doit être écrite :

- `gochecknoglobals` ×4 — `txKey`, `tenantKey`, `requestIDKey` sont des clés de contexte : le type
  privé au niveau paquet **est** l'idiome Go qui empêche les collisions. `RegisterRoute` est une
  constante de langage publié.
- `govet nilness` ×1 — `panicking_handler_does_not_kill_the_dispatcher_test.go` écrit dans une
  carte nil **exprès**, pour provoquer la panique que le test vérifie.
- `gocritic redundantSprint` ×2 — `password_never_appears_in_a_log_test.go` passe par
  `fmt.Sprintf("%v")` **exprès** : c'est le chemin de fuite qu'on teste, `String()` le
  contournerait.
- `gosec G304` ×1 — `disk.Get` ouvre un fichier depuis une clé, mais `domain.IsWithin` l'a validée
  **avant**. `gosec G104` ×1 — `hijacked.Close` sur un chemin d'erreur déjà signalé.
- `unparam` ×1 — l'aide de test `hash` ne reçoit qu'une valeur ; élargir un test plutôt que
  restreindre l'aide.

### Pièges d'outillage découverts — ne pas les redécouvrir

| Piège | Effet | Remède |
|---|---|---|
| `arch-go` cherche **`arch-go.yml`**, sans point | Le fichier s'appelait `.arch-go.yml` : l'outil n'a JAMAIS pu le lire | Renommé. Ne pas remettre le point. Le garde d'inertie de la CI le nommait aussi — corrigé |
| `arch-go` a changé de chemin de module | `go install github.com/fdaines/arch-go` échoue | C'est `github.com/arch-go/arch-go`. Corrigé dans `Taskfile.yml` et la CI |
| Écriture PowerShell mal encodée | 408 séquences d'accents doublement encodées dans 8 fichiers | Réparé. Toujours écrire en UTF-8 sans BOM |
| Fichier verrouillé par l'éditeur | `golangci-lint fmt` échoue sur « Accès refusé » | Fermer l'éditeur, relancer |

### Frictions ouvertes (`documentation/process/JOURNAL_FRICTION.md`)

| Réf | Effet concret |
|---|---|
| F001 | Docker absent : migrations, intégration et e2e ne tournent qu'en CI |
| F002 | Aucune protection de branche serveur (plan gratuit) — le crochet est un filet, pas un contrôle |
| F003 | Aucun test de mutation |
| F004 | Outillage en `latest` : CI non reproductible |
| F005 | `-race` exige CGO : `task test` sans `-race` en local, `task test:race` en CI |

### Prochaines actions, dans l'ordre

1. **Finir la campagne `golangci-lint`** — voir § « Campagne de signalements » plus bas
2. **#2** : migrations, un schéma et un rôle SQL par module
3. **#3 / #4 / #5 / #8 / #10** : adaptateurs puis binaires — le premier `curl` qui répond, et le
   premier worker qui dépile réellement
4. **#1** : `task check` vert de bout en bout → tag `v0.1.0`

### Arbitrages en attente côté produit

- **#9 `auth`** : session cookie / jeton porteur par surface ? fournisseur d'identité externe
  (Keycloak, Zitadel, Auth0) ou magasin interne ? `rbac`, `permissions`, `abac`, ou ReBAC ?
- **#18 F002** : dépôt public, GitHub Pro, ou assumer l'absence de protection ?
- **#34** : langue du **code** — recommandation posée, anglais dès maintenant pour `godoc` et les
  identifiants, français pour `rules/` jusqu'à la PR de traduction
- **#36** : SQLite comme pilote SQL par défaut ?
