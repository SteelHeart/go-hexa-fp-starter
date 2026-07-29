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
> État constaté le **2026-07-28**, **corrigé le 2026-07-29** sur les points listés en #148. Chaque
> verdict est **sourcé** — commande ou fichier. Aucun n'est écrit de mémoire.
>
> ⚠️ **Ce document a affirmé trois faits faux**, trouvés en cherchant de quels lots P2 attendait :
> un compte de champs qui n'a jamais existé, un dossier déclaré absent le jour où il a été créé, et
> une phrase se disant « revérifiée » alors qu'elle ne l'était plus. Les trois dérivaient dans le
> sens qui renforçait le propos. Ce qui suit vaut donc pour ce document lui-même : **remesurer, pas
> recopier** — et une phrase qui annonce avoir été vérifiée n'est pas une phrase vérifiée.

## Comment recompter, plutôt que relire

Le relevé du 2026-07-28 annonçait « 6 rouges » alors que sa grille en portait 7. L'écart venait
d'une relecture, pas d'un comptage. Une seule commande tranche, et elle doit être exécutée à chaque
remesure :

```bash
awk -F'|' '/^\| [0-9]+ \|/ {gsub(/ /,"",$4); print $4}' documentation/produit/personas.md \
  | sort | uniq -c
```

Elle lit la colonne d'état de chaque ligne numérotée et compte les verdicts. Si le total ne
correspond pas au texte de la lecture d'ensemble, c'est le TEXTE qui a tort.

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

## Les preuves — mesurées, pas affirmées

Depuis le **2026-07-29**, chaque persona a une **preuve tangible** : un mini-projet de liste de
tâches construit avec `hexa new`, puis mesuré contre le tableau de critères de sa propre persona
(issue #138). Voir [`preuves/`](preuves/README.md).

| Persona | Verdict mesuré |
|---|---|
| [P1](preuves/p1-equipe-produit.md) · **PRIMAIRE** | ✅ trois critères sur quatre |
| [P2](preuves/p2-fort-trafic.md) | 🔴 quatre sur cinq rouges |
| [P3](preuves/p3-adoption-externe.md) | 🔴 impossible en l'état |
| [P4](preuves/p4-exploitant.md) | ✅ cinq sur cinq |
| [P5](preuves/p5-decideur.md) | ⚠️ trois sur quatre |

⚠️ **Ce document affirme, les preuves mesurent.** Quand les deux divergent, c'est la preuve qui a
raison — et **cinq** critères de cette grille ont déjà été corrigés par elles : P1 sur les fichiers
du framework, P2 sur la taille d'ingestion, puis les trois faits faux relevés en #148 (le compte de
champs de `Config`, l'existence de `tests/perf/`, et « aucun lot n'a touché P2 »).

## Les personas

### P1 — L'équipe produit d'ImpactOne · **PRIMAIRE**

**Qui** : l'équipe qui construit le produit d'ImpactOne sur ce socle. C'est le premier consommateur
réel du framework, et le seul dont l'insatisfaction est rédhibitoire.

**Ce qu'elle exige** : livrer des fonctionnalités métier sans réécrire d'infrastructure, et sans que
la forme du socle se dégrade au bout de trois mois.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Fichiers du **framework** à modifier **À LA MAIN** pour ajouter un module métier | **0** | **0** ✅ — ADR 014. Mesuré : aucun fichier de `internal/config/`, `internal/pkg/` ni `internal/infrastructure/` ne nomme un module métier. Ajouter `billing` demande son dossier, **une ligne** dans `internal/modules/catalog.go` et son montage dans le composition root — trois emplacements qui appartiennent tous à l'**application** |
| Commandes qui **créent** un module | ≥ 1 | **1** ✅ — `hexa make:feature <nom>` écrit l'anatomie complète, la teste, et déclare la règle d'étanchéité `arch-go` du module neuf |
| Délai avant premier succès depuis un clone nu | < 10 min | ~5 min ✅ — et vrai **avec une base** depuis #84 : `task init && task up` échouait jusque-là sur un volume neuf |
| Modules métier livrables sans authentification | 0 | tous 🔴 — `auth` n'existe pas (#11) |

> **Un blocage sur trois subsiste.** P1 était bloquée sur *créer*, *brancher*, *configurer* un
> module. **Brancher** est levé (ADR 014), **créer** l'est par `hexa make:feature` (#17).
> **Configurer** ne l'est pas : `Config` reste une **struct fermée**, sans aucun point d'extension
> (#147). **16 champs**, recomptés le 2026-07-29 — autant qu'au premier relevé.
>
> ⚠️ Ce paragraphe a longtemps annoncé « **19 aujourd'hui** contre 16 à l'origine — elle s'est
> refermée un peu plus ». `Config` n'a **jamais** eu 19 champs : 15 le 2026-07-25, 16 depuis. Le
> critère reste rouge, mais l'aggravation était inventée — et elle l'était dans le sens qui
> renforçait le propos, seule direction où une dérive ne se fait pas remarquer. Corrigé le
> 2026-07-29 (#148).
>
> Reste, hors de cette liste, le blocage qui n'est pas technique : **`auth` n'existe pas**, et il
> attend des arbitrages produit, pas du code.

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

| Critère | Cible | Aujourd'hui | Lot |
|---|---|---|---|
| Réponse en flux possible | oui | **non** — `write_timeout: 10s` la tue | #143 |
| Taille d'ingestion | configurable | **oui** ✅ depuis le 2026-07-29 — la clé gouverne enfin. Elle était lue, validée, et **sans effet** : huma pose sa borne sur **chaque opération**, avec un défaut de 1 MiB, et c'est elle qui répondait | #141 |
| File de travaux longs | oui | **aucune** — `cmd/worker` ne dépile que l'outbox | #144 |
| Benchmarks de non-régression | ≥ 1 | **0 `func Benchmark`** — `tests/perf/` existe et tourne depuis le 2026-07-29 (#91), mais un scénario k6 exige une API démarrée et ne dit pas **quelle** fonction a régressé | #145 |
| Profilage mémoire sous charge | possible | **aucun `pprof`, aucun `GOMEMLIMIT`, aucun `GOMAXPROCS`** | #146 |

**Ce qu'elle tue** : la configuration fermée et les délais HTTP en dur. Les deux sont **rédhibitoires**
pour elle, pas gênants.

> 🔴 **Quatre critères sur cinq rouges**, le cinquième en demi-teinte. Un seul lot a touché cette
> persona depuis le premier relevé : #91, qui a rendu `task test:perf` réel — la commande lançait
> `k6 run tests/perf/registration.js` sur un dossier absent, et le scénario mesure désormais pour de
> bon (3 483 inscriptions, p95 à 288 ms).
>
> ⚠️ **Ce constat a longtemps dit « cinq sur cinq, aucun lot n'a touché cette persona ». C'était
> faux dans les deux moitiés** : #91 avait touché le critère n°4, et la même phrase affirmait que
> `tests/perf/` n'existait pas le jour où il existait. Corrigé le 2026-07-29 (#148).
>
> **Ce qui reste vrai, et qui est le vrai constat** : jusqu'au 2026-07-29, **aucun des cinq critères
> de P2 n'avait d'issue**. Pas une issue dépriorisée — **aucune issue du tout**. Un lot non priorisé
> apparaît dans une liste et se discute ; un lot inexistant n'apparaît nulle part, et six relevés
> peuvent le constater sans que rien ne bouge. Les cinq lots ci-dessus ont été ouverts ce jour-là.

### P3 — L'équipe qui adopte de l'extérieur

**Qui** : découvre le socle, l'évalue, et voudrait en **dépendre** plutôt que le copier.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Paquets importables depuis l'extérieur du module | > 0 | **0** — les **cinq** paquets hors `internal/` sont **tous** `package main`, et Go interdit d'importer un `main`. Il n'y en a aucun à promouvoir : il faut en **créer** |
| Politique de versions et de dépréciation | écrite | **inexistante** — zéro tag, aucun `CHANGELOG`. Dépend de #89 |
| Frontière API publique / interne | déclarée | **inexistante** — l'ADR 015 en fixe la **méthode**, pas le résultat |
| **Licence permettant l'usage** | oui | **oui** ✅ — **Apache-2.0** depuis le 2026-07-29 (ADR 020, #155), concession de brevet comprise |
| Langue de l'API et du règlement | lisible par l'équipe | le **code est en anglais** (ADR 018) ; `rules/` reste en **français**, ~4 600 lignes — coût d'entrée réel pour une équipe non francophone |

> ⚠️ **Le critère « Licence » a été REFORMULÉ, et c'est la preuve P3 qui l'a exigé.** Il disait
> « Licence — existe », et il était **formellement atteint** alors que P3 n'avait le droit de rien
> faire : *tous droits réservés* est un fichier de licence parfaitement existant. Un critère qui
> passe au vert sans que la persona puisse agir ne mesure pas ce qu'il prétend mesurer. Il dit
> désormais **« licence permettant l'usage »**, et il passe au vert pour la bonne raison.

**Ce qu'elle tue** : `internal/` partout. Tant qu'il tient, ce dépôt **ne peut qu'être copié** — donc
ce n'est pas encore un framework, quelle que soit l'intention.

### P4 — L'exploitant · *ne code pas au quotidien*

**Qui** : tient l'astreinte. Son unique question à 3 h du matin : *où est passé le temps, et
qu'est-ce que je redémarre ?*

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| Traces exploitables en production | oui | **oui** ✅ — `telemetry.Setup` et le serveur de métriques sont câblés dans les deux binaires (#13). Prouvé de bout en bout : `trace_id` dans chaque ligne de journal, `/metrics` en 200, et la trace **reçue par Jaeger** |
| Sondes de vie et de disponibilité | oui | **oui** ✅ — `/healthz`, `/readyz` |
| Publication garantie malgré un incident | oui | **oui** ✅ — outbox transactionnel, **éprouvé contre un vrai Postgres** depuis #37 |
| Rejeu sans effet de bord | oui | **oui** ✅ — idempotence, exclusivité sous concurrence réelle (#37) |
| Journal inaltérable | oui | **oui** ✅ — `UPDATE`/`DELETE` révoqués, constaté. Et le constat lui-même est désormais fiable : le garde tournait sous un rôle qui n'a plus le droit d'endosser `hexa_platform`, il aurait conclu « refus confirmé » sans rien tenter (#84) |

**Ce qu'elle tue** : tout ce qui n'est pas observable. Une configuration d'observabilité sans câblage
est pire qu'aucune : elle fait croire que la question est traitée.

### P5 — Le décideur technique · *ne code pas*

**Qui** : choisit ou refuse le socle. Arbitre sur le risque, le recrutement et la pérennité.

**Critères mesurables** :

| Critère | Cible | Aujourd'hui |
|---|---|---|
| La barrière qualité a **réellement tourné** | oui | **oui** ✅ — verte de bout en bout sur `main`. C'était le critère le plus lourd du document : 66 exécutions, 66 `startup_failure` (#47) |
| Le socle dépend-il d'une personne ? | non | **non** ✅ — aucun `CODEOWNERS`, aucun pseudo dans les règles |
| Documentation en accord avec l'état réel | oui | **⚠️ à surveiller** — deux affirmations fausses corrigées le 2026-07-28 : `REPRISE` déclarait `task up` vérifié alors qu'il échouait sur un volume neuf, et **ce document-ci** écrivait « le dépôt est public », faux depuis la décision du 2026-07-27. Le critère n'est pas acquis une fois pour toutes, il se re-mesure |
| Coût d'entrée d'un développeur | faible | **élevé** 🔴 — ~70 règles, en français |

> ⚠️ **Deux contrôles restent rouges en permanence, et il faut le dire à P5** avant qu'elle ne le
> découvre : **CodeQL** (#72 — analyse indisponible en dépôt privé sur ce plan) et **Deploy UAT**
> (#75, corrigé mais non encore constaté en conditions réelles). Un rouge permanent apprend à
> ignorer le rouge : c'est son coût, pas le job lui-même.

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

Verdicts **remesurés le 2026-07-28**, sur `main`, après `auth` (#11), `notification` et le
consommateur d'événements (#9), et l'ouverture du pool (#103). Les critères **7, 8, 11, 12 et 13**
ont été **recorrigés le 2026-07-29** (#148). Vocabulaire du dépôt :
**✅ prouvé** · **⚠️ écrit non prouvé** · **🔴 absent** · **⛔ refusé assumé**.

| # | Question | État | Source | Persona la plus touchée |
|---|---|---|---|---|
| **Amorçage** ||||
| 1 | Comment je crée mon projet ? | ✅ | `hexa new` — et la garantie est **outillée** : un job CI génère un projet et y lance `task check`, cliquets de couverture compris | P1 |
| 2 | De quoi ai-je besoin sur ma machine ? | ✅ | `deploy/toolbox/` — rien d'installé, `go run` démarre sans service | P1 |
| **Écriture** ||||
| 3 | Comment je crée un module ? | ✅ | `hexa make:feature <nom>` — domaine, ports, cas d'usage, pilote sans dépendance, catalogue, composition root, **surface HTTP** et tests aux quatre niveaux. Il **éprouve** le projet entier avant de rendre la main, et écrit la règle d'étanchéité `arch-go` du module neuf. Le module engendré est **joignable dès sa création** (ADR 019) : la seconde surface s'écrit en copiant un frère | P1 |
| 4 | Quelles conventions m'impose-t-on ? | ✅ | `arch-go` **21 règles**, `golangci-lint` ~50 analyseurs, `rules/` | P1 · P5 |
| 5 | Comment mon module se branche-t-il ? | ✅ | ADR 014 — chaque module déclare ses pilotes dans son propre `catalog.go`, le catalogue est **passé** au chargeur. Les trois tables globales de `internal/config` ont disparu | P1 |
| **Surfaces** ||||
| 6 | Plusieurs frontends simultanés ? | ✅ | **trois** adaptateurs primaires — `user_registration/adapters/primary/{http,cli}` et `auth/adapters/primary/http` — plus un **consommateur d'événements** dont la chaîne complète a tourné (#9, #103). La démonstration tient parce que HTTP et CLI appellent le **même port** sur le **même module** : la carte d'impact de #8 ne touche ni `domain/`, ni `ports/`, ni `application/`. ⚠️ Le consommateur reste câblé dans `cmd/worker` et non monté en `adapters/primary/events/` — la doctrine est démontrée, cet adaptateur-là ne l'est pas | P2 |
| 7 | Streaming et temps réel ? | 🔴 | aucun SSE, aucun WebSocket, aucun `Flusher` ; `write_timeout: 10s` borne le temps **total** d'écriture, donc coupe toute réponse en flux (#143). La borne d'ingestion effective est **1 MiB**, et ce n'est **pas** `max_body_bytes` qui la fixe : cette clé est validée puis ignorée (#141) | P2 |
| **Configuration** ||||
| 8 | Ajouter mon groupe de configuration ? | 🔴 | `Config` est une **struct fermée**, **16 champs** (`internal/config/config.go`, recomptés le 2026-07-29) — aucun point d'extension : un groupe `billing:` exige de modifier un fichier du framework, alors que **brancher** un module n'exige plus rien (ADR 014). Lot #147 | **P1 · P2** |
| 9 | Changer de configuration à chaud ? | ⚠️ | `dynconf` existe ; pilote par défaut `file`, donc **pas à chaud** ; pilote `postgres` jamais exécuté | P1 |
| 10 | Secrets et environnements ? | ✅ | feuilletage `config/*.yaml` → `env/{env}.yaml` → `local.yaml` → `${VAR}`, 36 tests | P1 · P4 |
| **Charge** ||||
| 11 | Où passent mes travaux longs ? | 🔴 | `cmd/worker` **ne dépile que l'outbox** — aucune file généraliste. L'outbox n'en tient pas lieu : son contrat est *au moins une fois* sur un événement, sans résultat à rendre ni état à consulter. Lot #144 | P2 |
| 12 | Comment je maîtrise la mémoire ? | 🔴 | aucun `GOMEMLIMIT`, `GOMAXPROCS`, pool ni borne de goroutines ; `limits` ne couvre que le débit. En conteneur, le ramasse-miettes ignore donc la limite mémoire et se fait tuer sans trace applicative. Lot #146 | P2 · P4 |
| 13 | Comment je vérifie que ça tient ? | ⚠️ | `tests/perf/` **existe et tourne** depuis le 2026-07-29 (#91) — 3 483 inscriptions, p95 à 288 ms. Mais **0 `func Benchmark`** : k6 exige une API démarrée, ne tourne pas sur une PR, et ne dit pas **quelle** fonction a régressé. Lot #145 | P2 · P5 |
| **Exploitation** ||||
| 14 | Comment j'authentifie ? | ✅ | `auth` livré : jeton **opaque**, session ouverte/résolue/révoquée, compte d'amorçage à secret engendré (ADR 017). **Prouvé sur le binaire réel** — 201 → 200 → 204 → 401 après révocation. ⚠️ L'**autorisation** est prouvée par 39 tests et son garde est en revue (#109) : tant qu'il n'est pas fusionné, `Authorize` n'est joignable par aucune surface | P1 · P4 |
| 15 | Où est passé le temps ? Comment je reprends ? | ✅ | traces ✅ depuis #13 — `trace_id` et `span_id` dans chaque ligne de journal, `/metrics` servi, trace reçue par le collecteur · outbox, idempotence, audit, isolation SQL ✅, désormais **éprouvés contre de vrais services** (#37) | P4 |
| **Durée** ||||
| 16 | Comment je monte de version ? | 🔴 | tout sous `internal/` → **rien n'est importable** ; ni versions, ni frontière API. L'ADR 015 en fixe la **méthode**, pas encore le résultat | P3 |

**Lecture d'ensemble** : **5 verdicts 🔴 sur 16**, 9 ✅ et 2 ⚠️ — compté par la commande `awk`
ci-dessus, le 2026-07-29, pas relu.

> 🔴 **Le relevé précédent annonçait « 6 rouges » alors que sa propre grille en portait 7.** L'écart
> a été trouvé en recomptant la colonne, pas en la relisant. C'est exactement ce que la règle d'or
> n°2 interdit — *la doc ne ment jamais sur l'état réel* — et c'est aussi la raison pour laquelle ce
> document exige une **remesure**, jamais une recopie. Le compte est désormais vérifié
> mécaniquement : `awk` sur la colonne d'état.

Deux gains dans ce lot. **`auth` (#14)** — le critère le plus lourd de P1 et de P4 — passe de
« rien » à « prouvé sur le binaire réel ». Et **le critère 6 passe au vert** : la surface CLI (#8)
appelle le même port que HTTP sur le même module, et sa carte d'impact ne touche aucune couche
interne. C'est la propriété n°2 du socle enfin *mesurée* plutôt qu'énoncée — elle avait attendu
trois relevés.

Le socle reste excellent sur l'axe **correction** (conventions, fiabilité, isolation) et **quasi vide
sur l'axe débit et flux**. **P1**, la primaire, était bloquée sur trois points : **brancher**,
**créer** et **authentifier** sont levés ; **configurer** ne l'est pas (#147) — et c'est désormais
son DERNIER blocage.

> ⚠️ **Ce que ce relevé dit du séquencement, et qui n'est pas confortable.** Les premiers lots
> livrés après le 2026-07-27 — niveau `integration`, amorçage réparé, gardes de déploiement —
> servaient surtout **P5** et **P4** : *faire dire vrai à la barrière*. C'était nécessaire, et ça ne
> levait aucun blocage de la persona primaire. Le constat a été écrit ici avant d'être corrigé, et
> c'est à ça que sert ce document.
>
> **P2 reste presque intouchée.** Sur ses cinq critères, un seul a bougé, et seulement en nuance :
> le critère 13, par #91. Les quatre autres — streaming, file de travaux, mémoire, point d'extension
> de la configuration — sont rouges depuis le premier relevé.
>
> 🔴 **Et la cause n'est pas un ordre de priorité.** En cherchant *de quels lots P2 attend-elle*, la
> réponse mesurée le 2026-07-29 est : **aucun. Il n'en existait pas un seul.** Recherche sur
> `stream`, `SSE`, `Flusher`, `queue`, `benchmark`, `pprof`, `GOMEMLIMIT`, `extension`, dans les
> issues ouvertes **et** fermées : zéro résultat. Une seule issue de tout le dépôt citait P2 avant
> ce jour-là, et elle est fermée (#91).
>
> **La nuance n'est pas de vocabulaire.** Un lot dépriorisé figure dans une liste : on le voit, on
> le discute, on assume de ne pas le prendre. Un lot **inexistant** ne figure nulle part — et six
> relevés successifs peuvent constater le même rouge sans que rien ne se passe, parce qu'il n'y a
> rien à prendre. C'est la même famille de défaut que les gardes qui ne gardaient rien : **une
> absence qui ne produit aucun symptôme**.
>
> Les cinq lots manquants ont été ouverts le 2026-07-29 : #143, #144, #145, #146, #147. Deux d'entre
> eux — la file de travaux et le point d'extension de configuration — engagent l'architecture et
> portent `needs-decision` : ils attendent un arbitrage, pas du code.

---

## La matrice persona × version

| Version | Promesse en une phrase | Sert | Ce qui y entre |
|---|---|---|---|
| **v0.1** | « Le socle tient ce qu'il affiche » | **P1** partiellement, **P5** | **Toute la ligne est livrée** : barrière CI réelle (#47) ✅ · `auth` et son garde d'autorisation (#11) ✅ · 2ᵉ surface CLI (#8) ✅ · tests d'intégration (#37) ✅ · observabilité branchée (#13) ✅ · consommateur d'événements (#9) ✅. Ne restent que des rouges d'ENVIRONNEMENT : #72 (CodeQL indisponible en dépôt privé) et #89 (séquencement du tag) |
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

### ~~🔴 L'urgence : aucun fichier de licence~~ — RÉSOLUE le 2026-07-27

**Le texte précédent était doublement faux, et il faut dire pourquoi plutôt que l'effacer.** Il
affirmait que le dépôt était **public** — il est **privé**, et la décision du 2026-07-27 le maintient
**interne** jusqu'à preuve sur des projets réels. Et il s'inquiétait d'un `Dockerfile` déclarant
`MIT` sans fondement.

Les deux sont traités (#61) : `LICENSE` existe, **tous droits réservés**, avec l'intention d'ouvrir
écrite noir sur blanc ; l'étiquette du `Dockerfile` est passée de `MIT` à `NONE`.

Ce que cela ne fait pas : **P3 ne peut toujours rien faire légalement**. La différence est qu'il n'y
a plus d'ambiguïté, et que le refus est daté et motivé au lieu d'être un silence. Le choix de la
licence d'ouverture — `Apache-2.0` recommandé plutôt que MIT, pour sa clause de brevets — **n'est pas
tranché**, pas plus que le titulaire du droit d'auteur.

> ✅ **TRANCHÉ le 2026-07-29 — la licence est `Apache-2.0`** (ADR 020, #155). La recommandation
> ci-dessus est retenue, pour le motif exact qui y était écrit.
>
> **Ce qui a forcé la décision est la preuve P3**, pas un changement d'avis : la condition de
> transfert exige que **toutes** les personas soient éprouvées, et P3 était rouge **par décision**.
> Aucune quantité de travail ne l'aurait fait passer au vert. La contradiction n'était visible nulle
> part avant que les personas ne soient **mesurées** plutôt qu'affirmées.
>
> ⚠️ **Et l'irréversibilité commence là.** Tout paquet rendu importable devient utilisable pour
> toujours ; un paquet publié par excès ne se retire plus. La règle de l'ADR 015 — *la frontière se
> dérive d'un usage mesuré, elle ne se décrète pas* — passe de prudente à indispensable.

### Ce qui devient bloquant pour `v1.0`

| Sujet | Statut | Pourquoi maintenant |
|---|---|---|
| **Licence** | ✅ #155 | **Apache-2.0** (ADR 020). `CONTRIBUTING` et `CHANGELOG` restent 🔴 — §5 règle le **juridique** des contributions entrantes, pas le **processus** |
| **Sortir de `internal/`** (#16) | 🔴 | 75 paquets, **0 importable** : le dépôt ne peut qu'être **copié**. Sans ça, il n'y a pas de framework, quelle que soit l'intention |
| **Frontière API publique / interne** | 🔴 | Extraire 530 identifiants sans les classer publierait une API par accident, et on ne la reprendrait plus |
| **Politique de versions et de dépréciation** | 🔴 | Ce qu'un framework promet et qu'un boilerplate n'a jamais eu à promettre |
| **Traduction du godoc et de `rules/`** (#34) | ⚠️ | Réel mais **non bloquant** : aucune rupture d'API à la clé |

### Ce que ça change dans le séquencement — RÉVISÉ par l'ADR 015

Le paragraphe précédent concluait que la **classification des 75 paquets** devait précéder #16, et
qu'elle était faisable tout de suite « sans rien restructurer ». **L'ADR 015 a renversé cette
conclusion**, et le raisonnement mérite d'être gardé parce qu'il se reproduira :

- **Rien n'est urgent.** Le dépôt est privé ; aucun paquet n'est importable ; donc
  **l'irréversibilité n'a pas commencé**. Classer aujourd'hui n'achète aucune sécurité.
- **Classer au jugé, c'est décider deux fois.** Aucune application n'a jamais été construite sur ce
  socle. Une frontière posée sans usage mesuré serait une hypothèse — et une hypothèse qu'on ne
  reprendrait plus une fois publiée.

L'ordre retenu est donc : **`hexa new` → une application réelle → la frontière, DÉRIVÉE de sa liste
d'imports.** Le premier maillon est livré (#17, première moitié).

> ⚠️ **Le vrai coût du déplacement, sous-estimé partout ailleurs** : douze des treize règles de
> dépendance d'`arch-go` sont indexées sur des chemins `internal.`. Les déplacer les fait toutes
> cesser de correspondre — et le rapport afficherait **100 % de conformité en ne gardant plus rien**.
> Toute PR de déplacement doit porter son témoin d'échec (ADR 013).

La décision « **stabiliser avant de restructurer** » tient donc, mais pour une raison différente de
celle écrite au départ : non pas parce que la stabilité précède la structure, mais parce que **la
structure se déduit d'un usage qui n'existe pas encore**.
