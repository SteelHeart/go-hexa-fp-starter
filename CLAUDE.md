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

0. **[`documentation/process/REPRISE.md`](documentation/process/REPRISE.md)** — **poste neuf, ou
   reprise après une interruption** : amorçage vérifié, état du travail en cours, ordre de fusion,
   et ce qui attend un arbitrage. Le lire d'abord fait gagner les heures que coûtent F008 et les
   quatre commits mal numérotés.
1. **[`rules/README.md`](rules/README.md)** — base normative, découpée par thème. **Fait foi.**
   Relire le fichier du domaine concerné *avant* de coder.
2. **[`rules/interdictions.md`](rules/interdictions.md)** — la liste à relire avant d'ouvrir un
   fichier. Chaque ligne nomme l'outil qui refuse la violation.
3. **[`documentation/adr/`](documentation/adr/README.md)** — décisions d'architecture. **Font foi.**
   En cas de contradiction avec un autre document, l'ADR gagne.
4. **[`rules/definition-of-done.md`](rules/definition-of-done.md)** — la barre à franchir pour livrer.
4 bis. **[`documentation/produit/personas.md`](documentation/produit/personas.md)** — **le pour qui**.
   À interroger **avant** d'ouvrir un ADR : une décision qui ne sert aucune persona n'a pas à être
   prise. Porte aussi la **matrice par version** et la ligne de ce qu'on ne fera **jamais**.
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
documentation/produit/        personas, périmètre, matrice par version — le POUR QUI
documentation/process/        nomenclature, labels, templates
documentation/securite/       registre de failles, matrice d'accès
cmd/{server,worker}           composition root — le seul code qui connaît tout
cmd/cli                       ⟨absent⟩
config/*.yaml                 configuration par groupes, secrets par ${VAR} uniquement
internal/config/              lecture, fusion, validation — refuse le démarrage sur incohérence
internal/pkg/                 primitives sans dépendance : result, fp, pagination, middleware
internal/infrastructure/      socle technique sans métier : db, cache, http, telemetry, security
internal/contracts/           langage publié — ce que les modules s'échangent sans s'importer
internal/generator/           la logique de `hexa` — cmd/hexa n'est qu'une coquille (ADR 016)
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
tools/covergate/              cliquets de couverture — LA source unique des seuils,
                              lancée à l'identique par `task test` et par la CI
```

Six modules noyau existent : `outbox`, `idempotency`, `dynconf`, `audit`, `storage`, `scheduler`.
Un seul module métier, `user_registration` : cœur couvert (18 tests de domaine, 13 de cas d'usage),
plus 4 tests de module en boîte noire et **une surface HTTP** de 3 tests. La phrase précédente
affirmait qu'« il n'a encore aucun adaptateur, donc aucune surface ne l'appelle » — faux depuis la
tranche verticale : `POST /v1/users` répond.

⚠️ Mais **une seule** surface existe. `cmd/cli` (#8) et le consommateur d'événements (#9) sont
absents, donc la promesse « le nombre de frontends est un non-sujet » n'a **qu'une instance** — elle
est énoncée, pas démontrée. Et personne ne s'abonne à `user.registered.v1`.

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
- `arch-go` — **100 %, 20 règles sur 20**, couverture 100 %. Deux règles ajoutées pour `tools/**` :
  l'outillage de build ne dépend de **rien** du dépôt, et tient aux mêmes limites de forme.
  Motif mécanique autant qu'architectural : le seuil `coverage` d'arch-go mesure la part des paquets
  couverts par au moins une règle, donc **ajouter `tools/covergate` a fait tomber `task check`** à
  l'étape `arch` (98 % < 100 %). Le garde de couverture a fait échouer un autre garde — comportement
  exactement voulu, découvert parce que le code de retour a été vérifié, pas la sortie.
- **`go run ./cmd/worker` REFUSE de démarrer sur le pilote `memory`**, avec un code de retour non
  nul et le motif : ce pilote vit dans le processus, un dépileur séparé ne verrait jamais les
  événements du serveur. Il tournerait à vide **sans aucune erreur** — le seul défaut qui ne se
  signale jamais.
- `go test -shuffle=on ./...` vert — **285 tests de premier niveau**. La table ci-dessous en détaille
  227 ; les 58 suivants sont dans `internal/pkg/middleware/tests` (12),
  `internal/infrastructure/messaging/tests` (13), `internal/infrastructure/modulebus/tests` (10),
  `internal/infrastructure/httpserver` (3 internes) + `…/httpserver/tests` (11), et
  `internal/infrastructure/telemetry/internal_test.go` (9) :

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

- **Deux défauts LATENTS trouvés en écrivant les tests de `telemetry`**, tous deux dans la fonction
  d'arrêt : (1) `fmt.Errorf("…: %w", nil)` **ne rend pas nil** — il rend une erreur portant
  `%!w(<nil>)`, donc un arrêt réussi remontait une erreur et chaque déploiement aurait été compté en
  échec ; (2) l'aide s'appelait `errJoin` et **ne joignait rien** — elle rendait la première erreur
  et jetait la seconde. Latents parce que `telemetry.Setup` n'est appelée **nulle part** (#13) ;
  ils se seraient déclenchés le jour du branchement. Remplacés par `errors.Join`, avec un retour nil
  explicite. Mutation vérifiée.
- **Six modules noyau convertis** à l'anatomie de l'ADR 012 : `outbox`, `idempotency`, `dynconf`,
  `audit`, `storage`, `scheduler`. Chacun a un pilote sans dépendance, choisi par défaut.
- **Le dépileur de l'outbox est réécrit** dans `application/` : recul exponentiel, abandon après N
  essais, survie à un publieur qui panique, et un compte rendu distinct pour « publié mais non
  marqué » — le seul cas qui produit un doublon chez le consommateur.
- **Un module absent du CATALOGUE refuse d'être activé** : on n'active pas un module dont le code
  n'existe pas. La table globale `knownDrivers` a disparu avec l'ADR 014 — chaque module déclare
  ses pilotes dans son propre `catalog.go`.
- **`arch-go` : 100 % de conformité, 17 règles sur 17**, dont les 12 règles de dépendance —
  l'hexagone tient. Il n'avait JAMAIS pu tourner : l'outil cherche `arch-go.yml` et le fichier
  s'appelait `.arch-go.yml`, avec un point. Renommé.

### Écrit, NON prouvé

| Quoi | Pourquoi ce n'est pas prouvé |
|---|---|
| Pilotes `postgres` des modules noyau | Le SCHÉMA est désormais appliqué et vérifié en local, mais les pilotes eux-mêmes n'ont **aucun test, à aucun niveau** (#37) |
| ~~`migrations/postgres/` + `deploy/postgres/provision.sql`~~ | **PROUVÉ en local le 2026-07-28, depuis un volume DÉTRUIT** (`task reset`) : provision, mots de passe, `up`, `down`, rejeu, `verify.sql`, et les deux refus de l'ADR 011 — code de retour 0. ⚠️ La mesure du 2026-07-27 était trop généreuse : elle portait sur un cluster où les rôles avaient été fabriqués à la main. Sur un volume neuf, `task up` échouait — **quatre** défauts en cascade (#84), dont `verify.sql` refusant l'état que `provision.sql` documentait. Un environnement déjà amorcé ne prouve rien sur l'amorçage |
| Pilote `redis` de `idempotency` | Aucun Redis sur la machine de référence |
| Relais Kafka et RabbitMQ | Jamais exécutés contre un broker réel |
| `cache` | Compile, **zéro test** : `New` fait un `Ping` Redis et `JSON` exige un client — donc niveau intégration (#37), pas ici. `messaging` (13), `modulebus` (10), `httpserver` (14) et `telemetry` (9) sont désormais couverts |

### `task check` — cinq étapes sur six, la sixième bloquée par la MACHINE

`task check` enchaîne `fmt · vet · lint · arch · test · vuln`. Il a enfin été exécuté tel quel.

| Étape | État |
|---|---|
| `fmt` `vet` `lint` `arch` `vuln` | **verts**, dans l'enchaînement réel |
| `test` | **échoue** — et pas à cause du code |

> ✅ **VÉRIFIÉ le 2026-07-27 : `task check` est VERT de bout en bout, code de retour 0**, depuis un
> clone du dépôt placé hors du dossier protégé (`C:\Users\MAC\hexa-check`), **cliquets de couverture
> compris**. Les six étapes s'exécutent réellement — vérifié en listant les commandes lancées, pas
> en lisant « pas d'erreur » :
>
> ```
> fmt · vet · lint · arch · test (+ covergate) · vuln     →  exit 0
> ```
>
> `govulncheck` : **0 vulnérabilité** appelée par ce code. Le blocage ci-dessous est donc
> **entièrement environnemental** — rien dans le dépôt ne s'y oppose.

**🔴 F008 — un programme Go ne peut créer AUCUN fichier sous `C:\xampp\htdocs\`.**

**Cause identifiée avec certitude** : l'**accès contrôlé aux dossiers** de Windows Defender est
actif (`EnableControlledFolderAccess = 1`) et `C:\xampp\htdocs\dev` figure dans sa liste de dossiers
protégés. Il bloque l'écriture par les applications non reconnues — dont tout binaire Go fraîchement
compilé, et la chaîne d'outils téléchargée. Le shell, lui, est reconnu, d'où l'asymétrie.

**Ne pas désactiver cette protection** : c'est une garde anti-rançongiciel posée délibérément.
Les remèdes sont de déplacer le dépôt, de travailler sous WSL, ou d'ajouter `go.exe` aux
applications autorisées — ce dernier point relève de l'utilisateur, pas de l'outillage.

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

### Couverture — trois cliquets, un seul programme

`tools/covergate` applique les cliquets ; `task test` et la CI lancent la **même** commande, donc
ils ne peuvent plus diverger. Mesuré le 2026-07-26 :

| Cliquet | Valeur | Seuil | État |
|---|---|---|---|
| **Périmètre unitaire** — ce que `go test ./...` sans tag peut atteindre | **74,3 %** | 70 % | ✅ |
| **Cœur** `domain/` + `application/`, pondéré par instruction | **95,2 %** | 90 % | ✅ |
| **Code produit** — tout, pilotes compris | **60,2 %** | cliquet 59 % | ✅ |

**Le seuil de 70 % n'a PAS été abaissé.** Il portait sur un profil produit `go test ./...` **sans
tag**, donc incapable par construction d'exécuter une ligne de pilote Postgres ou Redis : il était
**inatteignable**, et un seuil inatteignable finit toujours par être abaissé. Il s'applique désormais
au périmètre que ce lot peut atteindre. Trois choses empêchent que ce soit un contournement — liste
d'exclusions énumérée et motivée, affichée à chaque exécution, et un cliquet distinct qui garde le
total du code produit. Détail dans `rules/tests.md` § 5.

> 🔴 **La dette est nommée, pas cochée : issue #37.** Huit paquets de pilotes ont **zéro test, à
> aucun niveau** — le tag `integration` n'est porté par **aucun fichier** du dépôt, et la CI n'a pas
> de job `integration`. Le pilote `memory` d'`idempotency` a 24 tests prouvant l'exclusivité ; le
> pilote `postgres`, celui qui tournera en production, en a **zéro**.

Un **garde anti-pourriture** fait échouer la CI si une exclusion ne correspond plus à aucun code —
donc le jour où un pilote est couvert, la CI **exige** qu'on retire son exclusion.

**Deux façons de mesurer faux, les deux rencontrées, les deux dans le sens « trop bas »** — le pire
sens, celui qui fait échouer un cliquet pour une raison inexistante :

| Faute | Effet | Remède |
|---|---|---|
| `-coverpkg=./...` oublié | Tests en boîte noire = autre paquet que le code exercé → **3,6 %** au lieu de 52 % | Drapeau ajouté au `Taskfile` et à la CI |
| Plages non **fusionnées** | Avec `-coverpkg`, chaque binaire de test émet un profil pour **tous** les paquets : une plage apparaît ~20 fois, presque toujours à zéro → **3,4 %** au lieu de 56,9 % | Fusion par plage, « couverte » = **OU** sur les occurrences |

**Toujours recouper un chiffre de couverture avec `go tool cover -func`** avant d'en tirer une
conclusion. Et ne jamais ajuster la barre pour qu'elle passe : soit on couvre, soit on retire du
périmètre **en le disant**.

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
| Un garde est livré avec le cas qui le fait échouer | ADR 013 |
| Le catalogue des modules est passé au chargeur, pas écrit dans le framework | ADR 014 |
| La frontière publique est dérivée d'un usage mesuré, pas décidée d'avance | ADR 015 |
| Dépôt INTERNE — `LICENSE` tous droits réservés, ouverture visée sans date | décision du 2026-07-27 |
| `v0.1.0` opérationnel · `v0.2.0+` nouveaux modules · `v1.0.0` frontière GELÉE | décision du 2026-07-27 |
| Monorepo multi-modules `core/` + `cli/` + `template/` | issue #16, **après** v0.1.0 |
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
- **Le `catalog.go` d'un module liste ce qui EXISTE**, pas ce qui est prévu — et il partage ses
  constantes avec le `switch` de `New`, dans le même paquet (ADR 014). `internal/config` ne nomme
  aucun module. Le catalogue d'intentions vit dans `documentation/technique/pilotes.md` ; un pilote
  en migre le jour où il est écrit, testé, et documente ses NON-garanties.
- **`application/` ne journalise pas et ne lit pas l'horloge.** Il rend compte par un port et reçoit
  son horloge — c'est ce qui le garde pur et testable sans analyser des journaux.
- Tests : `{paquet}/tests/` en boîte noire · `{paquet}/internal_test.go` pour les internes ·
  **un fichier par test**, nommé d'après lui en `snake_case`, aides partagées dans
  `helpers_test.go`.
- **Un fichier par fonction publique**, la même règle que pour les tests, appliquée au CODE. Un
  fichier long se découpe dès qu'il porte plusieurs responsabilités publiques indépendantes ; le
  paquet ne change pas, aucun appelant ne bouge. Motif concret : le limiteur de débit était cassé
  depuis toujours au milieu d'un fichier de 350 lignes, et personne ne relit une fonction qu'on
  n'est pas venu chercher. **Quatre paquets sont découpés** : `middleware` (1→8), `security` (1→4),
  `config` (1→9), `messaging` (2→7). Le fichier qui garde le nom du paquet porte le **langage** du
  paquet et une **carte des fichiers** en tête.
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
| `go run` ne relaie pas `SIGTERM` au binaire | L'arrêt propre ne s'exécute jamais : connexions coupées net, et **spans en tampon perdus**. Mesuré — 0 ligne « arrêt du serveur HTTP », et aucune trace reçue par le collecteur | Compiler puis lancer le binaire pour tout ce qui vérifie un ARRÊT. `Ctrl+C` reste correct : le signal va à tout le groupe de processus |
| `psql -c "…"` n'**interpole pas** ses variables `-v` | La commande part littéralement et le serveur répond `syntax error at or near ":"` — un message qui accuse la syntaxe SQL, jamais l'interpolation | Passer le SQL par l'**entrée standard** (`<<'SQL'`). C'est là que `:'litteral'` et `:"identifiant"` sont substitués — et c'est la seule façon sûre d'injecter un mot de passe |
| Un environnement **déjà amorcé** ne prouve rien sur l'amorçage | `task up` était réputé vert : il l'était sur un cluster où les rôles avaient été fabriqués à la main. Sur un volume neuf, quatre échecs en cascade (#84) | Mesurer depuis `task reset`, volume détruit. Même forme que le faux vert d'un garde : le succès observé portait sur autre chose que ce qu'on croyait mesurer |
| PowerShell tronque `-flag=nom.ext` | `-coverprofile=coverage.out` arrive à Go en `-coverprofile=coverage`. L'erreur nomme un fichier qu'on n'a jamais demandé | Passer par le shell POSIX pour les commandes à arguments `=`, ou quoter |

### Frictions ouvertes (`documentation/process/JOURNAL_FRICTION.md`)

| Réf | Effet concret |
|---|---|
| ~~F001~~ | **Résolue** — WSL + Podman rootless + [`deploy/toolbox/`](deploy/toolbox/README.md) : l'outillage en image, rien sur le poste |
| F002 | Aucune protection de branche serveur (plan gratuit) — le crochet est un filet, pas un contrôle |
| F003 | Aucun test de mutation |
| F004 | Outillage en `latest` : CI non reproductible |
| ~~F005~~ | **Résolue** — `gcc` dans la toolbox, `task test:race` vert en local |
| **F010** | Le garde CI « Isolation des schémas Postgres » rend des **faux positifs** (il analyse la prose des commentaires) — déjà rouge sur `main`, issue #40 |
| **F008** | **Aucun binaire Go ne peut écrire sous `C:\xampp\htdocs\`** → `task check` ne peut pas être vert ici. Sortir le dépôt de `htdocs`, ou passer sous WSL |
| ~~F006~~ | **Résolue** — `task` et `govulncheck` installés par `go install` |
| ~~F007~~ | **Résolue** — `go 1.25.12` dans `go.mod`, `GOTOOLCHAIN=auto` fait le reste |

### Prochaines actions, dans l'ordre

**Terminés** : #2 (barrière verte), #20, #37 (niveau `integration`), #17 (`hexa new`), et la
campagne de signalements. F001, F005, F006, F007 sont **résolues**. Ce ne sont plus des actions.

1. **#75** : `Deploy UAT` échoue **à chaque poussée sur `main`**, sur son garde « CI verte », en
   0,4 s, sans jamais interroger la CI. Un rouge permanent apprend à ignorer le rouge — c'est le
   coût réel, pas le job lui-même
2. **Tag `v0.1.0`** : la barrière est verte et l'amorçage fonctionne sur un volume neuf (#84). Ce
   qui reste avant de graver, ce sont les deux rouges permanents ci-dessus et #72
3. **Une application RÉELLE construite avec `hexa new`** — c'est l'étape que l'ADR 015 impose avant
   toute frontière publique : *sa liste d'imports EST la mesure*. Aucun paquet n'est importable
   aujourd'hui, `go list ./... | grep -v /internal/` ne rend que des binaires et un outil de build.
   ⚠️ Douze des treize règles de dépendance d'`arch-go` sont indexées sur `internal.` : toute PR de
   déplacement doit porter son témoin, sinon elle rend 100 % de conformité en ne gardant plus rien
4. **#11 `auth`** et **#23 `tenancy`** : les deux « non » que reçoit tout évaluateur produit, avant
   d'avoir pu découvrir que l'outbox est excellente

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
- **Il manque ce qu'il vient chercher** — `auth` (#11), `tenancy` (#23), `notification`, `payment`,
  `ratelimit` : tous 🔴. Un évaluateur pose deux questions, obtient deux « non », et repart sans
  jamais découvrir que l'outbox est excellente.
- **Le délai avant premier succès était infini** jusqu'à cette tranche. Il ne l'est plus.
- **`user_registration` est la TRANCHE DE RÉFÉRENCE, pas « l'application ».** Sa forme sera copiée
  pour écrire `billing`. Tout dossier manquant serait reproduit comme « pas nécessaire » — d'où
  l'exigence qu'elle soit canoniquement complète.

### Déclaration des pilotes d'un module métier — RÉSOLU (ADR 014, #76)

**Tranché ET implémenté.** Un module métier se déclare désormais dans
`config/modules.yaml` sans qu'aucun fichier du framework ne le nomme.

`internal/config` ne contient plus **aucun nom de module** : les trois tables globales —
`knownDrivers`, `defaultDrivers`, `sqlBackedDrivers` — ont disparu. Chaque module déclare ses
pilotes dans son propre `catalog.go`, dans le même paquet que le `switch` de `New`, et **partage
une constante avec lui** : la divergence entre catalogue et fabrique est devenue *impossible*, là
où l'ADR ne promettait que de la rendre improbable. C'est `goconst` qui l'a signalée.

Le catalogue appartient à l'**application** (`internal/modules/catalog.go`), pas au binaire : les
deux composition roots lisent le même `config/modules.yaml`, donc le même ensemble de noms
déclarables. Monter un module reste une décision par binaire.

⚠️ **`RequiresSQL` et `RequiresCache` ne sont appelées par aucun binaire** — seuls des tests les
utilisent. La promesse « démarre sans base » est donc *assertée*, pas *exercée*. Défaut
préexistant, indépendant de l'ADR 014.

> **Terminé** : la campagne `golangci-lint` (239 → 0) et **#5** (schéma `platform`, rôles,
> ADR 011, garde `verify.sql`, job CI `migrations`). Ce paragraphe créditait **#2** par erreur —
> #2 est la porte de sortie de phase 0, et elle est **ouverte**.

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
- **`hexa_migrator` est le rôle de connexion des migrations**, `NOINHERIT`, membre de `hexa_owner`
  qu'il **endosse** par `ALTER ROLE … SET ROLE` à l'ouverture de session. Il n'existait nulle part
  avant #84 : `.env.example` le nommait, `provision.sql` ne le créait pas, et les deux commandes
  données en commentaire produisaient un état que `verify.sql` **refuse**. Le garde n'a pas bougé ;
  c'est la provision qui s'y est conformée. `has_schema_privilege` respecte l'héritage, donc
  `NOINHERIT` = privilège **endossable, non tenu**.
- **Les mots de passe des rôles de connexion ne sont pas provisionnés.** `task db:credentials` les
  **extrait de `.env`** — source unique — et **refuse** hors `development`/`test`.

### Arbitrages en attente côté produit

- **#11 `auth`** : session cookie / jeton porteur par surface ? fournisseur d'identité externe
  (Keycloak, Zitadel, Auth0) ou magasin interne ? `rbac`, `permissions`, `abac`, ou ReBAC ?
- **#18 F002** : dépôt public, GitHub Pro, ou assumer l'absence de protection ?
- **#34** : langue du **code** — recommandation posée, anglais dès maintenant pour `godoc` et les
  identifiants, français pour `rules/` jusqu'à la PR de traduction
- **#36** : SQLite comme pilote SQL par défaut ?
