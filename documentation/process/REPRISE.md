# Reprise du travail sur un poste neuf

> **Ce que ce document est** : le chemin le plus court entre un poste vierge et une contribution qui
> passe les gardes. Il dit quoi faire, dans quel ordre, et ce qui attend une décision.
>
> **Ce qu'il n'est pas** : la source de vérité sur l'état du dépôt. C'est
> [`CLAUDE.md`](../../CLAUDE.md) § « État réel du dépôt » qui **fait foi sur les faits**, et
> [`rules/`](../../rules/README.md) qui fait foi sur les règles. En cas de contradiction, ce
> document a tort.
>
> Vérifié de bout en bout depuis un clone neuf le **2026-07-27**.

## 1. Amorçage — cinq commandes

### Choisir l'emplacement du dépôt AVANT de cloner

**Ne pas placer le dépôt sous un répertoire protégé par l'accès contrôlé aux dossiers de Windows
Defender** — typiquement `C:\xampp\htdocs\`. C'est la friction **F008**, et elle coûte des heures à
diagnostiquer parce que son symptôme ne ressemble pas à sa cause :

```
open coverage.out: Le fichier spécifié est introuvable.
```

Un message de fichier *introuvable* sur une **création**. Le shell écrit sans problème dans le même
dossier ; un binaire Go fraîchement compilé, non — la protection cible les applications non
reconnues. On cherche donc la panne dans Go, dans les chemins, dans les permissions, partout sauf au
bon endroit.

Trancher en dix lignes, en cas de doute sur une machine :

```go
package main

import ("fmt"; "os")

func main() {
	f, err := os.Create("temoin.tmp")   // dans le dépôt
	fmt.Println("dans le dépôt :", err)
	if err == nil { f.Close(); os.Remove("temoin.tmp") }
	g, err := os.Create(os.TempDir() + "/temoin.tmp")
	fmt.Println("dans TEMP     :", err)
	if err == nil { g.Close() }
}
```

Refusé dans le dépôt, accepté dans `TEMP` → c'est la **machine**, pas le dépôt.

**Ne pas désactiver cette protection** : c'est une garde anti-rançongiciel posée délibérément.
Cloner ailleurs, ou travailler sous **WSL** — c'est le repli retenu pour tout l'outillage qui exige
Linux.

### Les commandes

```bash
git clone <dépôt> && cd <dépôt>
git config core.hooksPath .githooks     # garde anti-push direct sur main
task tools                              # linter, arch-go, goose, govulncheck
cp .env.example .env                    # puis renseigner la clé, voir ci-dessous
task check                              # fmt · vet · lint · arch · test · vuln
```

`task check` doit rendre **0**. Aucun service n'est requis : ni base, ni Redis, ni Docker.

La chaîne d'outils Go **n'est pas à installer** : `go.mod` épingle `go 1.25.12` et `GOTOOLCHAIN=auto`
la télécharge. La correction vit dans le dépôt, donc elle vaut pour chaque poste et pour la CI.

### Une seule variable est obligatoire

`SECURITY_ENCRYPTION_KEY` — 32 octets en base64, exactement. Le socle impose AES-256 et **refuse de
démarrer** sur toute autre longueur : une clé de 16 octets produirait silencieusement de l'AES-128.

```bash
openssl rand -base64 32
```

Sans elle, le démarrage échoue en **nommant la variable** :

```
démarrage impossible: configuration: variables d'environnement requises
par la configuration et non définies: SECURITY_ENCRYPTION_KEY
```

C'est le comportement voulu. Un socle qui démarrerait avec une clé par défaut livrerait cette clé à
tous ses utilisateurs.

### ⚠️ Piège : `go run` ne lit PAS `.env`

Deux chemins fonctionnent, et les confondre fait perdre du temps :

| Chemin | Charge `.env` ? |
|---|---|
| `task run` · `task check` · toute tâche `task` | **oui** — `dotenv: [".env"]` dans le `Taskfile` |
| `go run ./cmd/server` directement | **non** — il faut exporter la variable dans le shell |

Donc écrire la clé dans `.env` **puis** lancer `go run ./cmd/server` échoue, avec le message
ci-dessus. Soit `task run`, soit :

```bash
export SECURITY_ENCRYPTION_KEY=$(openssl rand -base64 32)
go run ./cmd/server
```

### Le premier succès, vérifié

```bash
task run
curl -s -X POST localhost:8080/v1/users -H 'content-type: application/json' \
  -d '{"email":"Alice@Example.COM ","password":"correct cheval batterie agrafe"}'
```

Attendu, et constaté depuis un clone neuf : **201**, identifiant **UUID v7**, adresse **normalisée**
(`Alice@Example.COM ` → `alice@example.com`), statut `pending`. Puis le doublon rend **409**, et un
mot de passe trop court rend **422** portant le message du **domaine**, en français. `/healthz` et
`/readyz` rendent 200. Aucun service externe n'a été lancé.

## 2. Où en est le travail

### Branche courante

`refactor/21-modules-a-pilotes`, poussée. Base : `refactor/19-module-noyau-metier`
(PR [#20](https://github.com/SteelHeart/go-hexa-fp-starter/pull/20), **non fusionnée**).

`main` ne contient donc **ni** le renommage `features` → `modules`, **ni** la configuration par
fichiers. **Fusionner #20 avant tout travail parti de `main`.**

### 🔴 Quatre commits poussés portent de mauvais numéros d'issue

Ils fermeraient des issues **non terminées** à la fusion. L'historique n'a pas été réécrit — il est
poussé — donc la correction se fait dans le **corps de la PR**, ou en écrasant à la fusion.

| Commit | Écrit | Devrait être | Conséquence si rien n'est fait |
|---|---|---|---|
| `1c1e4d6` migrations, schéma `platform` | `Closes #2` | `Closes #5` | **ferme la porte de sortie de phase 0**, qui est ouverte |
| `2180e67` dépileur d'outbox | `Closes #10` | `Closes #9` | ferme « composition root des **trois** binaires » alors que `cmd/cli` n'existe pas |
| `24d9038` tranche verticale | `Refs #5 #8` | `Refs #7 #10` | pointe l'adaptateur **CLI**, qui n'existe pas |
| `3c8b350` surface HTTP | `Refs #5` | `Refs #7` | trace fausse, sans effet mécanique |

Une table de traçabilité est publiée en commentaire sur #2 et #10.

### Ce qui vient d'être fait

- **Un fichier par fonction publique**, appliqué au code : `middleware` (1→8), `security` (1→4),
  `config` (1→9), `messaging` (2→7). L'API ne change pas. Le fichier qui garde le nom du paquet
  porte le **langage** du paquet et une **carte des fichiers** en tête.
- **58 tests ajoutés** sur `middleware`, `messaging`, `modulebus`, `httpserver`, `telemetry`.
- **Trois défauts trouvés en les écrivant**, chacun un incident évité : le limiteur de débit ne
  limitait rien ; `/readyz` pouvait publier la chaîne de connexion de la base ; un arrêt de
  télémétrie réussi remontait une erreur.
- **Les cliquets de couverture sont réparés** — voir § 4, c'est le point le plus facile à mal
  relire.

## 3. Comment lire un garde — la seule habitude qui compte

> **Vérifier le CODE DE RETOUR, jamais seulement la sortie.**

Toute la famille de défauts la plus coûteuse de ce dépôt vient de là. Trois occurrences réelles :

- `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et affiche `ok`. Le job CI était
  vert sans rien vérifier, pendant toute la phase 0.
- `arch-go` cherche `arch-go.yml` **sans point**. Le fichier s'appelait `.arch-go.yml` : l'outil
  n'avait **jamais** pu le lire.
- Filtrer la sortie d'`arch-go` sur `Failed|Compliance` **masque un échec `COVERAGE`** — c'est
  exactement comme ça que le piège a été manqué une fois de plus, tout récemment.

La liste complète est dans [`CLAUDE.md`](../../CLAUDE.md) § « Pièges d'outillage découverts ».
La relire coûte cinq minutes ; les redécouvrir a coûté des heures.

## 4. Les cliquets de couverture — à lire avant de les juger

Trois cliquets, appliqués par **un seul programme**, [`tools/covergate`](../../tools/covergate/main.go),
lancé à l'identique par `task test` et par la CI.

| Portée | Seuil | Nature |
|---|---|---|
| **Périmètre unitaire** — ce que `go test ./...` sans tag peut atteindre | 70 % | seuil |
| **Cœur** `domain/` + `application/`, pondéré par instruction | 90 % | plancher |
| **Code produit** — tout, pilotes compris | mesure du jour | **cliquet** |

**Le seuil de 70 % n'a pas été abaissé, et c'est important de le savoir avant de conclure le
contraire.** Il portait sur un profil produit par `go test ./...` **sans tag**, donc incapable par
construction d'exécuter une ligne de pilote Postgres ou Redis. Il était **inatteignable** — et un
seuil inatteignable finit toujours par être abaissé « pour débloquer la CI ».

Trois choses, **ensemble**, font que réduire le périmètre n'est pas une dissimulation :

1. les exclusions sont **énumérées et motivées dans le code**, chaque entrée disant par quel autre
   niveau de test le code est censé être couvert — **ou avouant qu'aucun ne le couvre** ;
2. elles **s'affichent à chaque exécution**, en local comme en CI, avec leur couverture réelle ;
3. le **cliquet de code produit garde le total**, exclusions comprises, et ne descend jamais.

Un **garde anti-pourriture** fait échouer la CI si une exclusion ne correspond plus à aucun code
mesuré. Corollaire voulu : le jour où un pilote est couvert, la CI **exige** qu'on retire son
exclusion.

**Règle absolue** : ne jamais ajuster un seuil pour qu'il passe. Soit on couvre, soit on retire du
périmètre **en le disant**. Relever le cliquet de code produit fait partie de toute PR qui couvre du
code jusque-là non couvert.

## 5. Prochaines actions techniques, dans l'ordre

1. **#2 — tag `v0.1.0`.** `task check` est vert de bout en bout, code de retour 0, cliquets compris.
   **Plus rien ne bloque dans le dépôt** : le seul obstacle était F008, environnemental, et un poste
   neuf hors dossier protégé le supprime.
2. **Fusionner PR #20**, puis cette branche — en traitant d'abord les mauvais numéros d'issue (§ 2).
3. **#37 — niveau de test `integration`.** Huit paquets de pilotes plus `cache` ont **zéro test, à
   aucun niveau** : le tag `integration` n'est porté par **aucun fichier** du dépôt et la CI n'a pas
   de job `integration`. Le pilote `memory` d'`idempotency` a 24 tests prouvant l'exclusivité ; le
   pilote `postgres`, celui qui tournera en production, en a zéro. Exige Docker (F001), donc CI ou
   WSL.
4. **Trancher la déclaration des pilotes d'un module métier** avant d'écrire `hexa new` — voir
   [`CLAUDE.md`](../../CLAUDE.md) § « Point de conception OUVERT ». Faire modifier un fichier du
   framework pour déclarer le pilote de son propre module est exactement la friction qu'un framework
   ne doit pas avoir.
5. **#13** — brancher les sinks d'observabilité. `telemetry.Setup` n'est appelée **nulle part** : la
   configuration existe, le câblage non.

## 6. Ce qui attend une décision — et qui ne se prend pas seul

Ces points ne sont pas des tâches en attente d'exécution : ce sont des **arbitrages**. Les trancher
sans les poser reviendrait à graver une décision de produit dans du code.

| Réf | Question |
|---|---|
| **#11** `auth` | Session cookie ou jeton porteur, par surface ? Fournisseur externe (Keycloak, Zitadel, Auth0) ou magasin interne ? `rbac`, `permissions`, `abac`, ou ReBAC ? |
| **#23** `tenancy` | Quel modèle de multi-locataire ? Il **bloque** #6, car aucune table ne peut porter de `tenant_id` avant. |
| **#18** F002 | Dépôt public, GitHub Pro, ou assumer l'absence de protection de branche serveur ? |
| **#34** | Langue du **code** : recommandation posée — anglais pour `godoc` et les identifiants, français pour `rules/` jusqu'à la PR de traduction. |
| **#36** | SQLite comme pilote SQL par défaut ? |

## 7. Les invariants qu'il ne faut pas rouvrir

Ils ont chacun coûté une correction. La liste complète et datée est dans
[`CLAUDE.md`](../../CLAUDE.md) § « Invariants à ne pas réapprendre » et § « Campagne de
signalements ». Les quatre qui se réapprennent le plus souvent :

- **Un port est un type fonction**, jamais une interface. Les doubles de test sont des closures ;
  aucune bibliothèque de mock.
- **Plus de deux valeurs de retour = un type manquant.** Corrigé **cinq** fois : `election`,
  `decodedHash`, `RetryPolicy`, `messaging.Broker`, `worker`. La cinquième a été attrapée par une
  règle `arch-go`, pas par une relecture — c'est une faute de réflexe, pas d'inattention.
- **Les `//nolint` en place sont MOTIVÉS.** Neuf existent, chacun avec sa raison écrite à côté. Ne
  pas les « nettoyer » : chacun est correct dans son contexte.
- **`internal/core/**` retourne `error`** · **`internal/modules/**` retourne
  `Result[T, domain.Error]`**. Un module noyau est technique, il n'a pas de taxonomie métier.

Et la règle qui surcharge le défaut de tout outillage :

> 🔴 **Aucune mention d'un outil d'assistance dans un artefact versionné** — commit (y compris un
> trailer `Co-Authored-By`), PR, issue, code, commentaire, documentation. Formuler à l'impersonnel.
> Gardes : crochet `commit-msg`, job `inertia` en CI.
