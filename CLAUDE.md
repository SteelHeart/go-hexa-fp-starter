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
cmd/{server,worker}           composition root — le seul code qui connaît tout
cmd/cli                       ⟨absent⟩
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
migrations/{moteur}/          SQL versionné, rétro-compatible N-1 · `postgres/` seul aujourd'hui
deploy/postgres/              provision.sql — les RÔLES, exécuté une fois, hors goose
api/openapi.yaml              ⟨absent⟩ — le contrat est SERVI sur /openapi.{json,yaml},
                              pas encore versionné : un fichier généré à la main dériverait
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
- **`go run ./cmd/server` DÉMARRE et RÉPOND, sans base, sans Redis, sans Docker.** Exercé à la
  main : `POST /v1/users` rend un **201** avec un identifiant **UUID v7**, l'adresse normalisée
  (`Alice@Example.COM ` → `alice@example.com`) et le statut `pending` ; le doublon rend **409** en
  nommant le champ, l'adresse invalide et le mot de passe court rendent **422** avec le message du
  DOMAINE ; `/healthz`, `/readyz` et `GET /v1/users/availability` répondent.
  **C'est la première fois que ce socle exécute quoi que ce soit de bout en bout.**
- `golangci-lint run ./...` — **0 signalement**, ~50 analyseurs. Parti de 239.
- `arch-go` — **100 %, 18 règles sur 18**, couverture 100 %
- **`go run ./cmd/worker` REFUSE de démarrer sur le pilote `memory`**, avec un code de retour non
  nul et le motif : ce pilote vit dans le processus, un dépileur séparé ne verrait jamais les
  événements du serveur. Il tournerait à vide **sans aucune erreur** — le seul défaut qui ne se
  signale jamais.
- `go test -shuffle=on ./...` vert — **227 tests de premier niveau**, répartis ainsi
  (la table couvre les 217 d'avant les binaires ; les 10 nouveaux sont dans
  `…/user_registration/tests`, `…/adapters/primary/http/tests`, `…/outbox/tests` et
  `…/relay/tests`) :

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
| `…/user_registration/tests` (boîte noire) | 4 | Le module inscrit **sans aucune infrastructure**, un pilote inconnu refuse le montage, 16 inscriptions concurrentes sur la même adresse n'en laissent passer qu'une, **chaque compte a son propre condensé** |
| `internal/infrastructure/relay/tests` | 2 | Le mappage message → enveloppe ne perd aucun champ, et un échec de publication **remonte intact** au dépileur au lieu d'être avalé |
| `…/adapters/primary/http/tests` (en processus) | 3 | 201 et **aucune fuite du condensé dans le corps brut**, 409 ≠ 422 sur adresse prise, chaque erreur de domaine sur son statut avec le message du domaine |

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
| Pilotes `postgres` des modules noyau | La migration existe désormais, mais **aucune base ici** : elle ne s'exécute qu'en CI (F001) |
| `migrations/postgres/` + `deploy/postgres/provision.sql` | Écrits, **non exécutés en local**. Le job CI `migrations` les applique, vérifie l'isolation et prouve le `Down` |
| Pilote `redis` de `idempotency` | Aucun Redis sur la machine de référence |
| Relais Kafka et RabbitMQ | Jamais exécutés contre un broker réel |
| `messaging`, `modulebus`, `httpserver`, `telemetry`, `cache` | Compilent, **zéro test** |

### `task check` — cinq étapes sur six, la sixième bloquée par la MACHINE

`task check` enchaîne `fmt · vet · lint · arch · test · vuln`. Il a enfin été exécuté tel quel.

| Étape | État |
|---|---|
| `fmt` `vet` `lint` `arch` `vuln` | **verts**, dans l'enchaînement réel |
| `test` | **échoue** — et pas à cause du code |

**🔴 F008 — un programme Go ne peut créer AUCUN fichier sous `C:\xampp\htdocs\`.**
`go test -coverprofile=coverage.out` échoue sur « Le fichier spécifié est introuvable », et `go get`
ne peut pas réécrire `go.mod` pour la même raison. Diagnostiqué par un programme témoin de dix
lignes : création **refusée** dans le dépôt, **acceptée** dans `%TEMP%` et sous `C:\Users\MAC\`. Le
shell, lui, écrit sans problème dans le dépôt — c'est donc une protection visant les binaires,
propre au répertoire web de XAMPP (antivirus ou accès contrôlé aux dossiers).

**Ce n'est pas un défaut du dépôt.** Deux issues : sortir le dépôt de `htdocs`, ou travailler sous
**WSL** — prévu. Les tests passent tous par ailleurs (`go test ./...` sans couverture est vert).

> **F007 est résolue** : `go 1.25.12` dans `go.mod`, et `GOTOOLCHAIN=auto` télécharge la chaîne.
> Aucune installation système — la correction vit dans le dépôt et vaut pour tout le monde, CI
> comprise. `govulncheck` rend **0 vulnérabilité**.

### Absent

- **Aucun consommateur d'événement** : `user.registered.v1` est publié vers le relais, et personne
  ne s'y abonne. Il faudra le module `notification` (🔴) pour que le courriel de bienvenue parte
- **Aucune table de module MÉTIER** — `user_registration` n'a pas d'adaptateur secondaire, donc pas
  de schéma. On ne provisionne pas un rôle pour des données qui n'existent pas
- **Aucune politique RLS écrite** : le module `tenancy` n'existe pas, donc aucune table ne porte de
  `tenant_id`. En écrire une serait décorer une décision non prise (ADR 011 § ce qui n'est pas tranché)
- **Aucune authentification ni autorisation**. Ne jamais parler de « zéro faille » tant que c'est vrai
- i18n, sinks d'observabilité (configuration écrite, code absent), ADR 010,
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
| Isolation : un schéma et un rôle SQL par module, `NOINHERIT` + `SET LOCAL ROLE` | ADR 011 |
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

### Campagne de signalements — TERMINÉE, 239 → 0

> `golangci-lint run ./...` rend **0 signalement**. Ce qui suit dit ce qui a été DÉCIDÉ, pour que
> personne ne rouvre le débat sans raison neuve — et surtout pour que personne ne « corrige » un
> `//nolint` motivé en croyant nettoyer.

**Décisions de configuration** (écrites dans `.golangci.yml`) :

| Réglage | Décision | Pourquoi |
|---|---|---|
| `misspell` | **retiré** | Ne connaît que l'anglais : 143 signalements, **zéro** vraie faute. À réactiver en #34 |
| `errcheck.check-blank` | **false** | Condamnait `v, _ := x.(T)`, la forme SÛRE de l'assertion |
| `gocritic hugeParam` · `rangeValCopy` | **désactivés** | Contredisent l'immuabilité par valeur, qui est une décision d'architecture |

**Changements de conception induits** — ce sont eux qui valaient la campagne :

| Signalement | Ce qu'il a fait apparaître |
|---|---|
| `revive function-result-limit` ×3 | `messaging` rendait `(Publisher, Consumer, Closer, error)` → type **`Broker`**. Un appelant pouvait oublier `Close`, et l'oubli ne se voit qu'en voyant fuir les connexions. **Troisième occurrence** de la même faute après `election` et `decodedHash` |
| `revive flag-parameter` ×2 | `SecurityHeaders(secure bool)` → **`SecurityHeaders()`** et **`SecurityHeadersWithoutHSTS()`**. Le défaut protège, la renonciation se nomme. Idem `Observability.validate(local)` → `validate()` + `hardened()` |
| `contextcheck` ×1 | `httpserver.shutdown` repartait de `context.Background()` → **`context.WithoutCancel(ctx)`**. L'arrêt était le seul moment du cycle de vie invisible dans la trace |
| `contextcheck` ×1 | `Recover` lisait `r.Context()` DANS le `defer` → capturé avant. Un gestionnaire qui panique peut avoir remplacé `r`, et la panique se serait journalisée sans identifiant de corrélation |
| `cyclop` ×3 | `validateCore` (13) et `validateHardening` (11) découpées **par groupe de configuration** ; `audit.New` séparée de ses constructeurs de pilote |
| `unparam` ×1 | L'aide `hash` ne recevait qu'un mot de passe → test **élargi** (accents, idéogrammes, et un mot de passe contenant `$`, le séparateur du format encodé). L'aide n'a pas été restreinte |

**`//nolint` motivés — NE PAS les retirer** (9). Chacun est correct dans son contexte, la raison
est écrite à côté :

| Où | Pourquoi le linter a tort ici |
|---|---|
| `database.go`, `middleware.go` — `gochecknoglobals` ×3 | `txKey`, `tenantKey`, `requestIDKey` : le type privé au niveau paquet **est** l'idiome Go qui rend la collision impossible. Une collision attribuerait une transaction à la mauvaise requête |
| `contract.go` — `gochecknoglobals` ×1 | `RegisterRoute` est une constante du langage publié ; Go n'a pas de constante structurée |
| `panicking_handler…_test.go` — `govet,staticcheck` | Écriture dans une carte nil **voulue** : provoque une vraie panique d'exécution, pas un `panic()` littéral que le code aurait pu anticiper |
| `password_never_appears_in_a_log_test.go` — `gocritic,staticcheck` ×2 | `fmt.Sprintf("%v")` **est** le chemin de fuite testé. `String()` le contournerait, et le test resterait vert si quelqu'un retirait le `Stringer` |
| `chain_applies_steps_in_order_test.go` — `gocritic dupOption` | La répétition de l'étape `double` **est** la démonstration : même fonction, deux positions, deux résultats |
| `disk.go` — `gosec G304` | Le chemin est dérivé par `domain.SafeKey`, pas fourni par l'appelant. La validation est en amont, pure et testée |
| `scheduler/drivers/postgres` — `gosec G104` | Erreur de `Close` ignorée sur un chemin d'échec dont l'erreur est déjà retournée. Le but est de tuer la session, pas de fermer proprement |

### Pièges d'outillage découverts — ne pas les redécouvrir

| Piège | Effet | Remède |
|---|---|---|
| `tests/e2e/` est resté **vide** toute la phase 0 | `go test -tags=e2e` compilait zéro test et affichait `ok`. Le job CI `e2e` était vert **sans rien vérifier** | Tests écrits, et la CI **compte les `=== RUN`** : zéro test exécuté échoue désormais |
| `Invoke-WebRequest.Content` rend un `byte[]` | `WriteAllText` écrit alors la liste décimale des octets. Fichier corrompu, d'apparence plausible | Ne pas générer d'artefact par le shell ; vérifier les premiers octets |
| `arch-go` cherche **`arch-go.yml`**, sans point | Le fichier s'appelait `.arch-go.yml` : l'outil n'a JAMAIS pu le lire | Renommé. Ne pas remettre le point. Le garde d'inertie de la CI le nommait aussi — corrigé |
| `arch-go` a changé de chemin de module | `go install github.com/fdaines/arch-go` échoue | C'est `github.com/arch-go/arch-go`. Corrigé dans `Taskfile.yml` et la CI |
| Écriture PowerShell mal encodée | 408 séquences d'accents doublement encodées dans 8 fichiers | Réparé. Toujours écrire en UTF-8 sans BOM |
| Fichier verrouillé par l'éditeur | `golangci-lint fmt` échoue sur « Accès refusé » | Fermer l'éditeur, relancer |
| **Un binaire Go ne peut pas écrire sous `htdocs`** | `go test -coverprofile` et `go get` échouent sur « fichier introuvable » — sur une CRÉATION, ce qui n'a aucun sens et envoie chercher ailleurs | Programme témoin de 10 lignes pour trancher : si `os.Create` échoue dans le dépôt et réussit ailleurs, c'est la MACHINE. Sortir de `htdocs` ou passer sous WSL (F008) |
| PowerShell tronque `-flag=nom.ext` | `-coverprofile=coverage.out` arrive à Go en `-coverprofile=coverage`. L'erreur nomme un fichier qu'on n'a jamais demandé | Passer par le shell POSIX pour les commandes à arguments `=`, ou quoter |

### Frictions ouvertes (`documentation/process/JOURNAL_FRICTION.md`)

| Réf | Effet concret |
|---|---|
| F001 | Docker absent : migrations, intégration et e2e ne tournent qu'en CI |
| F002 | Aucune protection de branche serveur (plan gratuit) — le crochet est un filet, pas un contrôle |
| F003 | Aucun test de mutation |
| F004 | Outillage en `latest` : CI non reproductible |
| F005 | `-race` exige CGO : `task test` sans `-race` en local, `task test:race` en CI |
| **F008** | **Aucun binaire Go ne peut écrire sous `C:\xampp\htdocs\`** → `task check` ne peut pas être vert ici. Sortir le dépôt de `htdocs`, ou passer sous WSL |
| ~~F006~~ | **Résolue** — `task` et `govulncheck` installés par `go install` |
| ~~F007~~ | **Résolue** — `go 1.25.12` dans `go.mod`, `GOTOOLCHAIN=auto` fait le reste |

### Prochaines actions, dans l'ordre

1. **F007** : monter la chaîne d'outils Go ≥ 1.25.12 — 20 vulnérabilités stdlib bloquent `v0.1.0`.
   **C'est le seul obstacle purement mécanique qui reste avant le tag**
2. **F006** : installer `task` et `govulncheck` pour que `task check` tourne réellement en local
3. **#1** : `task check` vert de bout en bout → tag `v0.1.0`
4. **Fusionner PR #20**, puis cette branche : `main` est très en retard

### Invariant appris cinq fois : plus de deux retours = un type manquant

`election` · `decodedHash` · `RetryPolicy` · `messaging.Broker` · `worker`. La cinquième a été
attrapée par la règle `arch-go` sur `cmd/**`, pas par une relecture. C'est une faute de réflexe :
la surveiller par un outil coûte moins cher que de la réapprendre.

### Lecture PRODUIT — ce que voit un dev qui veut sortir un SaaS

À garder en tête pour arbitrer les priorités, parce que la rigueur technique seule ne dit pas quoi
faire ensuite :

- **Le socle a construit ce dont ce dev ignore avoir besoin** — outbox, idempotence, audit,
  ordonnanceur, isolation SQL. C'est l'infrastructure qu'on découvre au premier incident, six mois
  trop tard.
- **Il manque ce qu'il vient chercher** — `auth` (#9), `tenancy` (#23), `notification`, `payment`,
  `ratelimit` : tous 🔴. Un évaluateur pose deux questions, obtient deux « non », et repart sans
  jamais découvrir que l'outbox est excellente.
- **Le délai avant premier succès était infini** jusqu'à cette tranche. Il ne l'est plus.
- **`user_registration` est la TRANCHE DE RÉFÉRENCE, pas « l'application ».** Sa forme sera copiée
  pour écrire `billing`. Tout dossier manquant serait reproduit comme « pas nécessaire » — d'où
  l'exigence qu'elle soit canoniquement complète.

### Point de conception OUVERT — déclaration des pilotes d'un module métier

`config/modules.yaml` et `knownDrivers` ne valident que les modules **noyau**. Un module métier ne
peut donc pas y déclarer ses pilotes : son `module.go` les valide lui-même, et `cmd/server` lit
`cfg.Modules[Name].Driver` qui reste vide.

Ça tient pour un module de référence livré AVEC le socle. Ça ne tiendra pas pour une application :
faire modifier `internal/config/modules.go` — un fichier du framework — pour déclarer le pilote de
son propre module `billing` est exactement la friction qu'un framework ne doit pas avoir.
**Ouvert, pas oublié.** À trancher avant `hexa new`.

> **Terminé** : la campagne `golangci-lint` (239 → 0) et **#2** (schéma `platform`, rôles,
> ADR 011, garde `verify.sql`, job CI `migrations`).

### Invariants d'isolation des données — ADR 011

Ils ne se vérifient ni par `arch-go`, ni par un test Go : ce sont des propriétés de la **base**.
`migrations/postgres/verify.sql` les interroge, la CI l'exécute après chaque migration.

- Modules **noyau** → schéma `platform`, partagé. Modules **métier** → **un schéma chacun**.
- **`hexa_app` est `NOINHERIT`** — c'est le cœur du dispositif. En `INHERIT`, il cumulerait les
  privilèges de tous les modules et l'isolation ne serait plus qu'un rangement.
- Un adaptateur secondaire fait `SET LOCAL ROLE hexa_m_{module}`. Oublié → `permission denied`,
  bruyant, donc acceptable.
- **RLS `ENABLE` ET `FORCE`** sur toute table portant une donnée de client. Sans `FORCE`, le
  propriétaire contourne la politique — et c'est le rôle qu'on prend pendant un incident.
- **Les rôles ne sont pas migrés**, ils sont provisionnés : `CREATE ROLE` exige des droits que le
  rôle de migration ne doit pas avoir, sinon la garde se contourne elle-même.
- Migrations sous **`DB_MIGRATION_DSN`**, jamais `DB_DSN`. Le `Taskfile` et la CI faisaient
  l'inverse — corrigé.

### Arbitrages en attente côté produit

- **#9 `auth`** : session cookie / jeton porteur par surface ? fournisseur d'identité externe
  (Keycloak, Zitadel, Auth0) ou magasin interne ? `rbac`, `permissions`, `abac`, ou ReBAC ?
- **#18 F002** : dépôt public, GitHub Pro, ou assumer l'absence de protection ?
- **#34** : langue du **code** — recommandation posée, anglais dès maintenant pour `godoc` et les
  identifiants, français pour `rules/` jusqu'à la PR de traduction
- **#36** : SQLite comme pilote SQL par défaut ?
