# Personas, périmètre et matrice par version

> **Ce que ce document est** : la définition du **pour qui**, dont découle le **quoi**. C'est lui
> qu'on interroge quand un arbitrage est bloqué, et c'est lui qui autorise à dire **non**.
>
> **Ce qu'il n'est pas** : une liste de souhaits. Chaque persona doit pouvoir **tuer** une
> fonctionnalité, pas seulement en demander.
>
> **Ce qu'il ne remplace pas** : [`parite-frameworks.md`](../technique/parite-frameworks.md) reste
> une **comparaison** avec Spring, Laravel et Symfony. Une comparaison dit quoi *avoir* ; elle ne dit
> jamais quoi refuser, ni pour qui. C'est ce document-ci qui définit le périmètre.
>
> État constaté le **2026-07-27**. Chaque verdict est **sourcé** — commande ou fichier. Aucun n'est
> écrit de mémoire.

## Les règles de l'exercice

Un exercice de personas raté coûte plus cher que pas de personas du tout. Quatre pièges, quatre
gardes :

| Piège | Garde appliquée ici |
|---|---|
| **Personas alibis**, inventés pour justifier ce qu'on voulait déjà construire | Chaque persona nomme ce qu'elle **tue**. Une persona est **explicitement refusée** |
| **Inflation** — union de tous les besoins, périmètre infini | **Cinq maximum**, **une seule primaire** : sa satisfaction *est* la définition de la version |
| **Non-falsifiable** — « il veut de la performance » ne se vérifie pas | Chaque persona porte des critères **mesurables en une commande** |
| **Confondre l'utilisateur et le décideur** | Deux personas **ne codent pas** |

---

## Les personas

### P1 — L'équipe produit d'ImpactOne · **PRIMAIRE**

**Qui** : l'équipe qui construit le produit d'ImpactOne sur ce socle. C'est le premier consommateur
réel du framework, et le seul dont l'insatisfaction est rédhibitoire.

**Ce qu'elle exige** : livrer des fonctionnalités métier sans réécrire d'infrastructure, et sans que
la forme du socle se dégrade au bout de trois mois.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Fichiers du **framework** à modifier pour ajouter un module métier | **0** | **≥ 2** — `internal/config/config.go`, `internal/config/modules.go` |
| Commandes qui **créent** un module | ≥ 1 | **0** — `task migrate:new` est la seule commande de création des 40, et elle ne crée qu'un fichier de migration vide |
| Délai avant premier succès depuis un clone nu | < 10 min | ~5 min ✅ |
| Modules métier livrables sans authentification | 0 | tous — `auth` n'existe pas |

**Ce qu'elle tue** : la parité pour la parité. `search`, `document`, `workflow` ne l'intéressent pas
tant que `auth`, `notification` et `payment` manquent.

> ⚠️ **Hypothèse à confirmer, marquée comme telle.** Le code laisse deviner un contexte
> **mobile-first sur réseaux peu fiables, en zone franc CFA** : `currency: XOF` et `mobile_money`
> avec un champ `operator` dans le catalogue des modules noyau, et l'**idempotence revendiquée
> « au-delà de la parité »** avec pour motif explicite « le mobile rejoue ». Un stockage objet
> compatible S3 semble par ailleurs déjà en usage côté ImpactOne. **Si c'est exact**, `notification`
> (SMS, TOTP) et `ratelimit` remontent devant une partie de `auth`, et l'idempotence cesse d'être un
> raffinement pour devenir l'argument central. **Si c'est un simple exemple**, cette hypothèse tombe
> et l'ordre écrit tient.

### P2 — Le développeur backend à fort trafic

**Qui** : reprend le socle pour un backend modulaire à fort débit — flux, gros calculs, mémoire
tenue, plusieurs frontends simultanés, configuration modifiable à volonté.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Réponse en flux possible | oui | **non** — `write_timeout: 10s` la tue |
| Taille d'ingestion | configurable | **1 MiB en dur** (`max_body_bytes`) |
| File de travaux longs | oui | **aucune** — `cmd/worker` ne dépile que l'outbox |
| Benchmarks de non-régression | ≥ 1 | **0** |
| Profilage mémoire sous charge | possible | **aucun `pprof`, aucun `GOMEMLIMIT`** |

**Ce qu'elle tue** : la configuration fermée et les délais HTTP en dur. Les deux sont **rédhibitoires**
pour elle, pas gênants.

### P3 — L'équipe qui adopte de l'extérieur

**Qui** : découvre le socle, l'évalue, et voudrait en **dépendre** plutôt que le copier.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Paquets importables depuis l'extérieur du module | > 0 | **0** — les 75 paquets sont sous `internal/` |
| Politique de versions et de dépréciation | écrite | **inexistante** |
| Frontière API publique / interne | déclarée | **inexistante** |
| Langue de l'API et du règlement | lisible par l'équipe | **français** (#34) |

**Ce qu'elle tue** : `internal/` partout. Tant qu'il tient, ce dépôt **ne peut qu'être copié** — donc
ce n'est pas encore un framework, quelle que soit l'intention.

### P4 — L'exploitant · *ne code pas au quotidien*

**Qui** : tient l'astreinte. Son unique question à 3 h du matin : *où est passé le temps, et
qu'est-ce que je redémarre ?*

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Traces exploitables en production | oui | **non** — `telemetry.Setup` n'est appelée **nulle part** (#13) |
| Sondes de vie et de disponibilité | oui | **oui** ✅ — `/healthz`, `/readyz` |
| Publication garantie malgré un incident | oui | **oui** ✅ — outbox transactionnel |
| Rejeu sans effet de bord | oui | **oui** ✅ — idempotence |
| Journal inaltérable | oui | **oui** ✅ — `UPDATE`/`DELETE` révoqués, constaté |

**Ce qu'elle tue** : tout ce qui n'est pas observable. Une configuration d'observabilité sans câblage
est pire qu'aucune : elle fait croire que la question est traitée.

### P5 — Le décideur technique · *ne code pas*

**Qui** : choisit ou refuse le socle. Arbitre sur le risque, le recrutement et la pérennité.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| La barrière qualité a **réellement tourné** | oui | **non** — 66 exécutions, 66 `startup_failure` (#47) |
| Le socle dépend-il d'une personne ? | non | **non** ✅ — aucun `CODEOWNERS`, aucun pseudo dans les règles |
| Documentation en accord avec l'état réel | oui | **oui depuis le 2026-07-27** ✅ (#42) |
| Coût d'entrée d'un développeur | faible | **élevé** — ~70 règles, en français |

**Ce qu'elle tue** : le règlement s'il devient un obstacle au recrutement. C'est la tension centrale
du projet : **la densité du règlement est à la fois la valeur et la barrière à l'entrée.**

### ⛔ P0 — Le prototypeur jetable · **REFUSÉE**

**Qui** : veut un CRUD administratif généré, un ORM, et un prototype en une heure.

**Pourquoi on la refuse** : ses besoins sont **incompatibles par construction** avec la proposition
de valeur. Un CRUD généré produit des écrans qui contournent les cas d'usage, donc les règles métier.
Un ORM fait fuir la persistance dans le domaine — le défaut exact que toute l'architecture vise à
empêcher.

**Pourquoi l'écrire** : un refus non écrit est un refus qu'on renégocie tous les six mois. Celui-ci
est désormais **testable** : toute demande qui ne sert que P0 est hors périmètre, sans débat.

---

## La grille — 15 questions, 7 axes

Verdicts au **2026-07-27**, sur `refactor/21-modules-a-pilotes`. Vocabulaire du dépôt :
**✅ prouvé** · **⚠️ écrit non prouvé** · **🔴 absent** · **⛔ refusé assumé**.

| # | Question | État | Source | Persona la plus touchée |
|---|---|---|---|---|
| **Amorçage** ||||
| 1 | Comment je crée mon projet ? | 🔴 | `task rename` seul ; aucun générateur (#17) | P1 |
| 2 | De quoi ai-je besoin sur ma machine ? | ✅ | `deploy/toolbox/` — rien d'installé, `go run` démarre sans service | P1 |
| **Écriture** ||||
| 3 | Comment je crée un module ? | 🔴 | `task --list-all` : **1 seule commande de création sur 40**, `migrate:new`, qui produit une migration vide. Rien ne crée un module, une surface, un port ni un test — copie manuelle de `user_registration` | P1 |
| 4 | Quelles conventions m'impose-t-on ? | ✅ | `arch-go` 20 règles, `golangci-lint` ~50 analyseurs, `rules/` | P1 · P5 |
| 5 | Comment mon module se branche-t-il ? | 🔴 | `knownDrivers` dans `internal/config/modules.go` — **fichier du framework** | P1 |
| **Surfaces** ||||
| 6 | Plusieurs frontends simultanés ? | ⚠️ | doctrine juste, **une seule surface** : `adapters/primary/http` ; `cmd/` = `server`, `worker` | P2 |
| 7 | Streaming et temps réel ? | 🔴 | aucun SSE, aucun WebSocket, aucun `Flusher` ; `write_timeout: 10s` ; `max_body_bytes: 1 MiB` | P2 |
| **Configuration** ||||
| 8 | Ajouter mon groupe de configuration ? | 🔴 | `Config` est une **struct fermée de 16 champs** (`internal/config/config.go`) — aucun point d'extension | **P1 · P2** |
| 9 | Changer de configuration à chaud ? | ⚠️ | `dynconf` existe ; pilote par défaut `file`, donc **pas à chaud** ; pilote `postgres` jamais exécuté | P1 |
| 10 | Secrets et environnements ? | ✅ | feuilletage `config/*.yaml` → `env/{env}.yaml` → `local.yaml` → `${VAR}`, 36 tests | P1 · P4 |
| **Charge** ||||
| 11 | Où passent mes travaux longs ? | 🔴 | `cmd/worker` **ne dépile que l'outbox** — aucune file généraliste | P2 |
| 12 | Comment je maîtrise la mémoire ? | 🔴 | aucun `GOMEMLIMIT`, `GOMAXPROCS`, pool ni borne de goroutines ; `limits` ne couvre que le débit | P2 |
| 13 | Comment je vérifie que ça tient ? | 🔴 | **0 benchmark** ; `tests/perf/` **n'existe pas** alors que `task test:perf` le référence — commande morte | P2 · P5 |
| **Exploitation** ||||
| 14 | Comment j'authentifie ? | 🔴 | `auth` : **rien** (#11) | P1 · P4 |
| 15 | Où est passé le temps ? Comment je reprends ? | 🔴 / ✅ | traces 🔴 (`telemetry.Setup` appelée nulle part) · outbox, idempotence, audit, isolation SQL ✅ | P4 |
| **Durée** ||||
| 16 | Comment je monte de version ? | 🔴 | tout sous `internal/` → **rien n'est importable** ; ni versions, ni frontière API | P3 |

**Lecture d'ensemble** : **10 verdicts 🔴 sur 16.** Le socle est excellent sur l'axe **correction**
(conventions, fiabilité, isolation) et quasi vide sur l'axe **débit et flux**. Les deux personas les
plus mal servies aujourd'hui sont **P2** et **P3** ; **P1**, la primaire, est bloquée sur trois
points : créer un module, brancher un module, configurer un module.

---

## La matrice persona × version

| Version | Promesse en une phrase | Sert | Ce qui y entre |
|---|---|---|---|
| **v0.1** | « Le socle tient ce qu'il affiche » | **P1** partiellement, **P5** | Barrière CI réelle (#47) · `auth` (#11) · 2ᵉ surface CLI (#8) · tests d'intégration (#37) · observabilité branchée (#13) · consommateur d'événements (#9) |
| **v1.0** | « Le framework s'utilise **sans le modifier** » | **P1** · **P2** · **P3** | **Configuration ouverte** · **déclaration des pilotes hors framework** · `hexa new` (#17) · file de travaux · streaming et délais par route · frontière API publique et politique de versions · `notification`, `tenancy`, `i18n` |
| **v2.0** | « Le framework se déploie à l'échelle » | **P4** · **P5** | Cloud native, microservices, devops avancé, multi-région, découverte de services, disjoncteur, passerelle d'API |
| **⛔ jamais** | — | — | CRUD administratif généré · Active Record / ORM · conteneur d'injection · façades statiques · système de plugins |

**Trois lectures qui comptent** :

1. **La ligne `v1.0` est la vraie version 1.** Ce qui la définit n'est pas une liste de modules :
   c'est **« sans le modifier »**. Tant qu'ajouter un module métier exige de toucher deux fichiers du
   framework, il n'y a pas de version 1, quel que soit le nombre de modules livrés.
2. **`v2.0` est un report assumé, pas un oubli.** La posture déjà écrite tient : *on ne construit pas
   les microservices, on s'interdit de les rendre impossibles*. Le schéma par module, `modulebus`
   `inproc → http` et les contrats publiés sont les portes laissées ouvertes, à faible coût.
3. **La ligne ⛔ est la plus utile du tableau.** Elle transforme cinq refus arbitraires en refus
   défendables.

---

## Les hypothèses qui restent à confirmer

| # | Hypothèse | Ce qui change si elle est fausse |
|---|---|---|
| ~~H1~~ | ~~P1 évolue dans un contexte **mobile-first, paiement mobile (XOF)**~~ | **INFIRMÉE le 2026-07-27.** P1 construit **tout à la fois** : aucune verticale dominante. Et le paiement doit être **universel**, pas XOF — voir ci-dessous |
| ~~H2~~ | ~~L'adoption **externe** est visée~~ | **CONFIRMÉE le 2026-07-27.** P3 reste au jeu — voir « Ce que l'adoption externe change » ci-dessous |
| ~~H3~~ | ~~`v0.1.0` est un **jalon interne**~~ | **CONFIRMÉE le 2026-07-27.** `v0.1.0` = framework opérationnel, usage interne · `v0.2.0+` = nouveaux modules · `v1.0.0` = frontière publique gelée. Publication = décision distincte |

**Les trois hypothèses sont tranchées** au 2026-07-27. Ce qui suit remplace H1 et
précise ce que sa réfutation change.

### H1 infirmée — P1 ne se spécialise sur rien

P1 construit **tout à la fois** : il n'existe pas de verticale dominante à
optimiser. C'est une **contrainte**, pas un confort — le socle n'a pas le droit
de se spécialiser, sous peine de gêner la moitié des projets de l'équipe.

Conséquence directe sur `payment` : il doit être **universel**, jamais lié à une
devise ni à un fournisseur. Le port sera un type fonction, chaque fournisseur un
pilote, exactement comme les modules noyau (ADR 012). Le socle ne nommera **ni
XOF, ni Stripe, ni aucun opérateur** — et la formule de P1 vaut règle : *un
adaptateur pour tous les ports futurs*.

Ce qui change dans le classement : `payment` ne redescend plus au motif d'un
contexte régional, et `notification` / `ratelimit` ne remontent plus au motif de
réseaux peu fiables. Aucun module ne se justifie par une géographie.

---

## Ce que l'adoption externe change — décidé le 2026-07-27

**P3 est confirmée comme persona réelle.** Cinq conséquences, dont une bonne surprise et une urgence.

### ✅ La bonne surprise : l'API publique est déjà en anglais

Vérifié : **530 identifiants exportés, aucun en français, aucun accentué.** `RegisterUser`,
`EmailIsTaken`, `Broker`, `Cursor`, `Dispatcher`… Les noms de fichiers de test le sont aussi
(`zero_value_is_err_test.go`).

Conséquence : **#34 n'est pas une rupture d'API**, contrairement à ce qu'on pouvait craindre. Le
périmètre réel est la **traduction du godoc et de `rules/`** — coûteux en pages, nul en compatibilité.
La contrainte « traduire avant de figer l'API » tombe : on peut publier puis traduire.

### 🔴 L'urgence : aucun fichier de licence

Le dépôt est **public**, vise l'adoption, et le `Dockerfile` déclare `MIT` — mais il n'existe
**aucun `LICENSE`**. Sans lui, le droit d'auteur s'applique par défaut : **tous droits réservés**.
P3 ne peut légalement rien faire, et l'image affirme une licence sans fondement. Issue #61.

### Ce qui devient bloquant pour `v1.0`

| Sujet | Statut | Pourquoi maintenant |
|---|---|---|
| **Licence, `CONTRIBUTING`, `CHANGELOG`** | 🔴 #61 | Prérequis légal, le moins cher de tous |
| **Sortir de `internal/`** (#16) | 🔴 | 75 paquets, **0 importable** : le dépôt ne peut qu'être **copié**. Sans ça, il n'y a pas de framework, quelle que soit l'intention |
| **Frontière API publique / interne** | 🔴 | Extraire 530 identifiants sans les classer publierait une API par accident, et on ne la reprendrait plus |
| **Politique de versions et de dépréciation** | 🔴 | Ce qu'un framework promet et qu'un boilerplate n'a jamais eu à promettre |
| **Traduction du godoc et de `rules/`** (#34) | ⚠️ | Réel mais **non bloquant** : aucune rupture d'API à la clé |

### Ce que ça change dans le séquencement

La décision écrite « **stabiliser avant de restructurer** » ne tient plus telle quelle : on ne
stabilise pas une API qu'on n'a pas le droit de publier, et rien ne distingue aujourd'hui l'API
publique du détail interne. La **classification des 75 paquets** doit précéder #16, et #16 doit
précéder toute promesse de compatibilité.

Elle reste juste sur un point : classer n'oblige à **déplacer aucun fichier**. C'est un exercice de
décision, faisable avant `v0.1.0` sans rien restructurer.
