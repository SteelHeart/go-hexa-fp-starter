# Audit de conformité — avant transfert vers l'organisation

> Issue **#107**. Mesuré le **2026-07-29**, sur `main` à `82e0723`, dépôt **public**.
>
> Ce document est un **relevé d'écarts**, pas un plan. Chaque écart y est nommé avec sa gravité et
> son devenir : **corrigé dans ce lot**, **issue ouverte**, ou **dérogation écrite et motivée**.
> Aucun écart n'est laissé implicite — c'est le critère d'acceptation de #107.

## Ce que l'audit a mesuré, et comment

Un audit qui se lit n'est pas un audit. Tout ce qui suit est **exécutable**, et les commandes sont
données pour que le prochain relevé n'ait pas à réinventer la mesure.

| Contrôle | Commande |
|---|---|
| Paquets sans aucun importeur | `go list ./...` puis `grep -rl "\"$p\""` par paquet |
| Anatomie des modules (ADR 012) | parcours de `internal/{core,modules}/*/` |
| Convention de test | `git ls-files '*_test.go' \| grep -v '/tests/' \| grep -v internal_test` |
| Marqueurs de dette | `grep -rn 'TODO\|FIXME\|XXX' --include=*.go` |
| Fuite de secrets, historique entier | `gitleaks detect --source .` — **32 commits** |
| Mention d'outillage, racine → HEAD | `tools/verifie-mention-outillage.sh <racine> HEAD` |
| Barrière complète | `task check` |

### Deux mesures fausses écartées en route

L'audit s'est trompé deux fois avant de trouver, et les deux méritent d'être écrites — ce sont les
formes que reprendra le prochain.

1. **La recherche de paquets orphelins a d'abord rendu « aucun ».** Cause : `go` n'est pas sur le
   `PATH` du shell — il vit dans la toolbox. `go list ./...` rendait une liste **vide**, la boucle
   ne tournait sur rien, et le résultat ressemblait à un succès. **Exactement le faux vert du
   dépôt**, cette fois dans l'outil d'audit lui-même. Corrigé en passant par `./deploy/toolbox/tb`.
2. **Le contrôle « carte des fichiers » a d'abord accusé `internal/config`**, qui la porte
   parfaitement — sous une autre forme. Un motif de recherche trop littéral produit un faux
   positif, et un faux positif dans un rapport d'audit coûte plus cher qu'un oubli : il envoie
   corriger ce qui est juste. Voir **É-10**, où c'est la CONVENTION qui est en cause, pas les
   paquets.

---

## Tableau des écarts

| Réf | Gravité | Écart | Devenir |
|---|---|---|---|
| **É-01** | 🔴 haute | `internal/infrastructure/dynconf` est un paquet **mort** | corrigé dans ce lot |
| **É-02** | 🔴 haute | L'ADR 012 impose `surfaces/`, le code écrit `adapters/primary/` | **issue #116** |
| **É-03** | 🔴 haute | Le garde « Aucune dette dissimulée » n'a **aucun témoin** | **issue #117** |
| **É-04** | 🟠 moyenne | `pilotes.md` compte onze pilotes ; il y en a quinze | corrigé dans ce lot |
| **É-05** | 🟠 moyenne | La carte du dépôt omet huit chemins et en décrit un qui n'existe pas | corrigé dans ce lot |
| **É-13** | 🔴 haute | Le fichier d'amorçage portait le nom d'un outil d'assistance | **corrigé** — #114 |
| **É-06** | 🟠 moyenne | La langue du code n'est pas celle de la recommandation #34 | **issue #34** — arbitrage |
| **É-07** | 🟠 moyenne | Le godoc de `config` renvoie vers du code mort et nomme un dossier inexistant | corrigé dans ce lot |
| **É-08** | 🟡 basse | Un chemin de poste nominatif figure dans deux artefacts versionnés publics | corrigé dans ce lot |
| **É-09** | 🟡 basse | Quatre modules noyau n'ont pas d'`application/` | **dérogation écrite** — amendement dans #116 |
| **É-10** | 🟡 basse | La convention « carte des fichiers » n'a pas de forme unique | **issue #118** |
| **É-11** | 🟡 basse | Le garde de dette ne balaie ni `*.md` ni `*.sh` | **issue #117** |
| **É-12** | 🟡 basse | Le relevé dit que `user_registration` n'a pas d'adaptateur secondaire — il en a deux | corrigé dans ce lot |

**Ce que ce tableau ne contient pas** compte autant que ce qu'il contient. Les contrôles suivants
sont passés **sans un seul écart**, et il ne faut pas conclure qu'ils n'ont pas été faits :

- `gitleaks` sur les **32 commits** de l'historique — aucune fuite ;
- le garde de mention d'outillage de la **racine** à `HEAD` — aucune mention ;
- **aucun marqueur de dette** dans le code — les deux occurrences trouvées sont le garde qui les
  cherche ;
- **aucun fichier de test hors convention** : tout est en `{paquet}/tests/` ou `internal_test.go`,
  noms anglais, `snake_case`, un fichier par test ;
- le vocabulaire proscrit par l'ADR 012 — **aucun paquet, aucun fichier « service »** ;
- `task check` **vert, code de retour 0**, cliquets à **79,3 % · 92,4 % · 62,8 %** après ce lot.

---

## É-13 🔴 Le fichier d'amorçage portait le nom d'un outil d'assistance

**Ajouté après coup**, sur décision : un socle qu'une communauté adopte doit pouvoir être lu comme
l'œuvre d'une équipe de développeurs, pas comme la sortie d'un outil. C'est la même exigence que
« le dépôt ne dépend d'aucune personne », appliquée aux outils.

Le fichier d'amorçage — le premier que voit quiconque ouvre le dépôt, à la racine, désormais
publique — portait le nom d'un assistant. **La règle 🔴 du dépôt, violée par son propre point
d'entrée.**

**Aucun garde ne pouvait le voir.** Celui du contenu cherche son motif dans le TEXTE, et il écarte
explicitement les en-têtes `+++ b/…` du diff — commentaire d'origine : *« éviter d'accuser un
fichier pour son propre nom »*. La ligne était juste pour le cas qu'elle visait ; elle a rendu
celui-ci invisible pendant toute la phase 0.

**Correction, sans coquille de compatibilité.** La substance vit dans
`documentation/AMORCAGE.md`. Aucun fichier au nom d'outil ne subsiste : garder une coquille aurait
préservé le confort d'un outil au prix de la règle, et la règle est publique.

**Outillé** — sinon c'est une préférence : `verifie-mention-outillage.sh` gagne un cinquième volet
sur les NOMS des fichiers versionnés. **Liste énumérée, jamais un motif** : un motif accuserait
`cursor_round_trips_test.go`, où « cursor » est le mot anglais du domaine, et un garde qui crie au
loup sur du code légitime finit désarmé avant d'avoir servi.

**Un défaut trouvé en écrivant son témoin.** Le premier cas — « un artefact interdit est refusé » —
passait alors que la fonction de contrôle **n'existait pas** : elle rendait « commande introuvable »,
donc un code non nul, donc « refus » aux yeux du témoin. C'est le second cas, celui du nom légitime,
qui a révélé la panne. **Une moitié de témoin ne prouve rien** : sans le cas qui doit être ACCEPTÉ,
un garde cassé et un garde sévère sont indiscernables.

## É-01 🔴 `internal/infrastructure/dynconf` est un paquet mort

**Mesure.** Sur 106 paquets, c'est le **seul** sans aucun importeur. 188 lignes, **zéro test**,
zéro appelant. Il fait doublon avec `internal/core/dynconf`, qui est le module noyau converti à
l'anatomie de l'ADR 012 et qui, lui, a 14 tests et deux pilotes.

**Pourquoi il a survécu.** Il est antérieur à la conversion en module noyau. Rien ne signale un
paquet que personne n'importe : il compile, il passe `vet`, il passe `lint`, et `arch-go` mesure la
part des paquets **couverts par une règle**, pas la part des paquets **vivants**.

**Ce que ça coûtait en silence, MESURÉ.** Ses 188 lignes à 0 % étaient comptées dans les cliquets.
Sa suppression, seul changement de code de ce lot :

| Cliquet | Avant | Après | Seuil |
|---|---|---|---|
| Périmètre unitaire | 77,6 % | **79,3 %** | 70 % |
| Cœur | 92,4 % | 92,4 % | 90 % |
| Code produit | 61,7 % | **62,8 %** | 59 % |

**Retirer du code mort a fait gagner 1,7 point de couverture sans écrire un seul test.** C'est le
signe qui manquait : un cliquet tiré vers le bas par du code que personne n'exécute finit par
bloquer, et la tentation sera alors d'ajouter une exclusion plutôt que de constater que le code n'a
pas lieu d'être. Le cliquet de code produit a désormais **3,8 points de marge** — assez pour que
quelqu'un puisse supprimer des tests sans le faire échouer. Le relever est une décision à part,
délibérément hors de ce lot d'audit.

**Le doublon était pire que la mort.** Deux paquets nommés `dynconf`, l'un vivant et testé, l'autre
mort et non testé, avec le même vocabulaire. Un lecteur qui tombe sur le mauvais lit une
implémentation plausible qui n'est branchée nulle part.

→ **Supprimé dans ce lot.**

## É-02 🔴 L'ADR 012 impose `surfaces/`, le code écrit `adapters/primary/`

L'anatomie gravée dans l'[ADR 012](../adr/012-anatomie-d-un-module-et-pilotes.md) dit :

```
├── surfaces/              adaptateurs primaires optionnels : http · cli · events
```

Le dépôt n'a **aucun** dossier `surfaces/`. Il a trois `adapters/primary/`, et la carte de
`documentation/AMORCAGE.md` documente `adapters/primary/` + `adapters/secondary/`.

**Ce n'est pas une question de goût, c'est une question d'autorité.** La hiérarchie du dépôt est
explicite — *en cas de contradiction avec un autre document, l'ADR gagne*. Donc, en l'état, **le
code viole une décision d'architecture**, et la carte documente la violation.

Deux issues distinctes s'y cachent :

1. **Quel nom ?** `adapters/primary/` + `adapters/secondary/` est le vocabulaire hexagonal
   canonique et se lit sans glossaire. `surfaces/` est plus court et l'ADR 012 en fait un terme du
   vocabulaire imposé — mais il ne couvre alors que le côté primaire, et l'ADR ne dit **nulle part**
   où vivent les adaptateurs secondaires. C'est le vrai défaut de l'ADR : il a nommé une moitié.
2. **Qui bouge ?** Si c'est le code, la migration touche trois dossiers et le générateur. Si c'est
   l'ADR, il faut un **amendement écrit** — pas une réécriture silencieuse, sinon la trace de la
   décision disparaît.

**Ce que le générateur en fait — vérifié, et c'est une troisième réponse.** `hexa make:feature`
n'engendre **ni `surfaces/`, ni `adapters/`**. Ses gabarits couvrent `domain/`, `ports/`,
`application/`, `drivers/memory/`, `tests/` — et s'arrêtent là.

Il ne reconduit donc pas le mauvais choix : il **esquive la question**. Conséquence concrète, plus
gênante que l'écart de nom : **un module engendré n'est joignable par aucune surface**, et la
personne qui vient de le créer doit deviner où poser son premier adaptateur — sans que la structure
ne le lui dise, et alors que `user_registration` est censée être la tranche de référence à copier.

Un dossier absent est reproduit comme « pas nécessaire ». C'est écrit noir sur blanc dans le relevé
d'état, à propos de cette même tranche.

→ **Issue #116.** Ne se tranche pas dans un lot d'audit : c'est un amendement d'ADR.

## É-03 🔴 Le garde « Aucune dette dissimulée » n'a aucun témoin

C'est **le dernier garde encore écrit en YAML dans `ci.yml`** — tous les autres vivent dans
`tools/` et portent leur cas d'échec (ADR 013). Il n'a donc :

- **aucun mode témoin** — rien ne prouve qu'il sait encore refuser ;
- **aucune exécution locale possible** — on ne peut pas le lancer avant de pousser.

C'est **exactement** la situation du garde d'inertie avant #111. Là-bas, l'inspection avait révélé
que son échappatoire documentée ne pouvait pas fonctionner — un défaut invisible depuis l'origine,
trouvé seulement en extrayant le script. Le précédent est trop précis pour être ignoré.

→ **Issue #117** : extraire vers `tools/verifie-dette.sh`, avec `--temoin` et son cas d'échec
versionné.

## É-04 🟠 `pilotes.md` compte onze pilotes ; il y en a quinze

Le document affirme **« Onze existent »** et donne un tableau de six modules. Mesure :

| Module | Pilotes déclarables | Dans `pilotes.md` ? |
|---|---|---|
| `outbox` | `memory` · `postgres` | ✅ |
| `idempotency` | `memory` · `postgres` · `redis` | ✅ |
| `dynconf` | `file` · `postgres` | ✅ |
| `audit` | `log` · `postgres` | ✅ |
| `storage` | `disk` | ✅ |
| `scheduler` | `cron-inproc` · `advisory-lock` | ✅ |
| **`auth`** | `memory` | 🔴 **absent** |
| **`notification`** | `log` | 🔴 **absent** |
| **`user_registration`** | `memory` | 🔴 **absent** |

Et le paragraphe d'avertissement est **doublement faux** aujourd'hui :

> Les pilotes `postgres` et `redis` ci-dessus sont **écrits mais jamais exécutés** : aucune
> migration n'existe (issue #2) et aucun service ne tourne sur la machine de référence (F001).

Les migrations existent et sont appliquées en CI depuis #5 et #84 ; F001 est résolue ; et #37 a
livré le niveau `integration` qui exerce ces pilotes contre un vrai Postgres et un vrai Redis.

**La gravité n'est pas le chiffre.** C'est qu'un document dont l'en-tête proclame *« ce qui fait
autorité est le `catalog.go` de chaque module »* se contredit trois lignes plus bas en donnant un
inventaire — et que cet inventaire est celui que lit un évaluateur pressé.

→ **Corrigé dans ce lot.**

## É-05 🟠 La carte du dépôt omet huit chemins, et en décrit un qui n'existe pas

Critère de #107 : *chaque dossier existant figure dans la carte, et réciproquement.*

| Chemin | État |
|---|---|
| `cmd/hexa/` | existe, **hors carte** (mentionné en passant dans la ligne du générateur) |
| `documentation/technique/` | existe, **hors carte** — c'est pourtant là que vivent `pilotes.md` et `modules-noyau.md` |
| `deploy/toolbox/` | existe, **hors carte** — c'est la résolution de F001 |
| `.github/` · `.githooks/` | existent, **hors carte** — dont les crochets nommés par les règles d'or |
| `tools/*.sh` | sept gardes, **hors carte** ; seul `tools/covergate/` y figure |
| `internal/core/tests/` | existe, **hors carte** — tests transverses aux modules |
| `tests/integration/` | existe, **hors carte** — la carte dit `tests/{e2e,perf}` |
| `tests/perf/` | **cartographié, n'existe pas** — c'est aussi la cause de #91 |
| `catalog.go` | **absent de l'anatomie de module**, alors que l'ADR 014 le rend obligatoire |

→ **Corrigé dans ce lot.**

## É-06 🟠 La langue du code n'est pas celle de la recommandation #34

Mesuré sur `internal/` et `cmd/` :

- **godoc : ≈148 commentaires français contre ≈8 anglais** ;
- **identifiants mixtes** dans les mêmes paquets : `abonner`, `composer`, `demarrer`, `depiler`,
  `envoyer`, `peupler` côtoient `newServer`, `mustBroker`, `newDispatcher`, `discardLogger`.

La recommandation posée est *« anglais dès maintenant pour `godoc` et les identifiants, français
pour `rules/` »*. Elle n'est appliquée **nulle part**.

**Un dépôt à moitié traduit coûte plus cher qu'un dépôt d'une seule langue** — c'est écrit dans #107
lui-même. Le mélange est ici **à l'intérieur d'une même fonction**, ce qui est le cas le plus
coûteux : il n'existe aucune frontière à laquelle s'arrêter pour lire.

Le passage en public change le calcul : le godoc est désormais lisible par tout le monde, et c'est
la première chose qu'un évaluateur ouvre.

→ **Issue #34, requalifiée en bloquant du transfert.** Un arbitrage — pas une tâche qu'un audit
tranche. Trois options, dont une seule est mauvaise : tout anglais, tout français, ou **rester à
moitié**.

## É-07 🟠 Le godoc de `config` renvoie vers du code mort et nomme un dossier inexistant

`internal/config/config.go` :

- renvoie le lecteur vers `internal/infrastructure/dynconf` — le paquet **mort** de É-01 ;
- parle deux fois de **`conf/`** alors que le dossier s'appelle **`config/`**.

Le second point paraît anodin. Il ne l'est pas : c'est le godoc du paquet qui **explique où vit la
configuration**, et il envoie chercher dans un dossier qui n'existe pas.

→ **Corrigé dans ce lot.**

## É-08 🟡 Un chemin de poste nominatif dans deux artefacts versionnés publics

`C:\Users\MAC\` apparaît dans `documentation/AMORCAGE.md` (deux fois) et dans `JOURNAL_FRICTION.md`. Le dépôt
affirme *« ne dépendre d'aucune personne : pas de `CODEOWNERS`, pas de pseudo dans les règles »*.

Ce n'est pas un secret, et ce n'est pas grave — mais c'est un nom d'utilisateur de poste, dans un
dépôt désormais public, qui ne sert **rien** : ce qui compte dans F008 est *« hors du dossier
protégé »*, pas le chemin exact.

`LICENSE` porte le nom du titulaire du droit d'auteur. C'est **nécessaire** et hors de portée de
cette règle.

→ **Corrigé dans ce lot** — le chemin est remplacé par sa propriété.

## É-09 🟡 Quatre modules noyau n'ont pas d'`application/` — dérogation écrite

`audit`, `dynconf`, `idempotency` et `storage` ont `domain/`, `ports/`, `drivers/`, `tests/`,
`module.go`, `catalog.go` — **pas d'`application/`**.

L'ADR 012 liste `application/` dans l'anatomie sans dire qu'il est facultatif, à la différence de
`surfaces/` qu'il marque explicitement « optionnels ».

**Dérogation, avec son motif** : ces quatre modules n'ont **aucun cas d'usage à orchestrer**. Un
port d'`idempotency` est `Reserve`/`Complete`/`Release` : il n'y a ni pipeline, ni décorateur, ni
séquence. Créer un `application/` qui ne ferait que réexporter le pilote produirait une couche de
transfert pure — précisément ce que l'architecture hexagonale reproche aux couches de service.

**La preuve que ce n'est pas de la complaisance** : les quatre modules qui ont *vraiment* une
politique — `outbox` (recul exponentiel, abandon après N essais), `scheduler` (élection),
`notification` (envoi), `auth` (authentification, autorisation) — ont **tous** leur `application/`.
Le partage suit le besoin, pas la commodité.

→ **Dérogation retenue.** L'ADR 012 doit néanmoins l'**écrire** : dans son état actuel, il fait
croire à une obligation. Repris dans l'**issue #116**, qui amende déjà cette même anatomie.

## É-10 🟡 La convention « carte des fichiers » n'a pas de forme unique

Sur les paquets découpés en trois fichiers ou plus, la « carte des fichiers » en tête du fichier
homonyme prend trois formes différentes — liste indentée dans `config.go`, prose dans
`middleware.go`, absente de `messaging.go` et `security.go` — et huit paquets n'ont pas de fichier
homonyme du tout.

**Aucun outil ne peut vérifier une convention qui n'a pas de forme.** C'est la règle d'or n°1 :
tant qu'elle est formulée en prose, elle sera appliquée à peu près.

→ **Issue #118** : fixer la forme, puis l'outiller.

## É-11 🟡 Le garde de dette ne balaie ni `*.md` ni `*.sh`

Il filtre `'*.go' '*.sql' '*.yml' '*.yaml'`. Un `TODO` dans un script de `tools/`, dans un fichier
de `rules/` ou dans un ADR passe sans bruit. Aucun n'existe aujourd'hui — c'est vérifié — mais le
garde ne le doit pas à sa couverture.

→ **Joint à l'issue #117**, qui réécrit ce garde de toute façon.

## É-12 🟡 `user_registration` a deux adaptateurs secondaires, pas zéro

La section « Absent » affirmait : *« `user_registration` n'a pas d'adaptateur secondaire, donc pas
de schéma »*. Il en a deux — `hashing` et `outboxpub`.

La conclusion reste juste, la prémisse est fausse : il n'a pas d'adaptateur secondaire **de
persistance**, et c'est ça qui explique l'absence de schéma. Une prémisse fausse qui porte une
conclusion juste est le pire cas : personne ne la corrige, puisque le résultat semble bon.

→ **Corrigé dans ce lot.**

---

## Ce que cet audit dit du dépôt

**Onze écarts sur douze portent sur la DOCUMENTATION, pas sur le code.** Le code tient : zéro
paquet hors anatomie sauf un mort, zéro test hors convention, zéro marqueur de dette, zéro fuite sur
32 commits, zéro mention d'outillage sur tout l'historique, `task check` vert.

Ce n'est pas un hasard, et ce n'est pas rassurant. **Le code est outillé, la documentation ne l'est
pas.** Chaque règle de forme du code a son garde qui refuse ; aucune affirmation de `documentation/AMORCAGE.md`,
`pilotes.md` ou d'un ADR n'a quoi que ce soit qui la confronte au dépôt. La dérive s'installe donc
exactement là où rien ne regarde — et c'est la partie sur laquelle tout le monde s'appuie pour
décider.

**Deux écarts sont de la même famille que les quatre gardes défaillants déjà trouvés** (E2E ne
vérifiant qu'un refus, garde de fuite incapable de correspondre, échappatoire d'inertie inopérante,
comptage des personas jamais recompté) : É-03, un garde sans témoin, et É-10, une convention sans
forme vérifiable. **Cinquième et sixième occurrences.**

La conclusion pratique n'est pas « écrire mieux ». C'est : **outiller la véracité de la
documentation comme on a outillé la forme du code.** Trois candidats immédiats, tous mécaniques :

1. la carte de `documentation/AMORCAGE.md` contre `git ls-files` — É-05 aurait été impossible ;
2. le tableau de `pilotes.md` contre les `catalog.go` — É-04 aurait été impossible ;
3. l'anatomie de l'ADR 012 contre l'arborescence réelle — É-02 et É-09 auraient sauté au premier
   passage.

→ Les trois sont réunis dans l'**issue #118**.

## Verrous restants avant le transfert

| Verrou | Nature |
|---|---|
| **#113 — la licence** | Décision. Tous droits réservés sur un dépôt public : lisible par tous, utilisable par personne |
| **#34 — la langue** | Décision. É-06 la rend bloquante : le godoc est désormais public |
| **#116 — l'anatomie** | Amendement d'ADR. Le générateur reconduit le choix à chaque module engendré |
| **#89 — le tag** | Séquencement. `v0.1.0` déclenche deux publications vers un hôte inexistant |
| **#9 · #27** | Périmètre. Deux issues rouvertes après avoir été comptées faites |

Aucun de ces cinq n'est un défaut de code. **Le transfert n'attend plus que des décisions** — c'est
le meilleur endroit où être, et c'est aussi celui où l'on s'arrête le plus longtemps.
