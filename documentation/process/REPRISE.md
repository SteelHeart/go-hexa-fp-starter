# Reprise du travail sur un poste neuf

> **Ce que ce document est** : le chemin le plus court entre un poste vierge et une contribution qui
> passe les gardes. Il dit quoi faire, dans quel ordre, et ce qui attend une décision.
>
> **Ce qu'il n'est pas** : la source de vérité sur l'état du dépôt. C'est
> [`documentation/AMORCAGE.md`](../../../AMORCAGE.md) § « État réel du dépôt » qui **fait foi sur les faits**, et
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

### Poste où l'on ne veut RIEN installer — la toolbox

Ces cinq commandes supposent Go et Task présents. Sur un poste qui n'en a aucun — et qui ne doit
rien recevoir — tout passe par [`deploy/toolbox/`](../../deploy/toolbox/README.md) :

```bash
git config core.hooksPath .githooks
systemctl --user enable --now podman.socket   # une fois, pour `task up`
cp .env.example .env                          # puis la clé, voir ci-dessous
./deploy/toolbox/tb task check                # construit l'image au premier appel
```

`tb` est le **seul** script qui s'exécute sur la machine. Go, Task, les linters, goose et psql
vivent dans une image ; le dépôt y est monté. Le `Taskfile` n'a pas été modifié pour autant : la
toolbox pilote le moteur de conteneurs de l'hôte par sa socket, et `docker compose` y désigne
Podman.

**Vérifié de bout en bout sous WSL + Podman rootless le 2026-07-27**, code de retour 0 à chaque
étape : `task check` (les six), `task test:race`, `task up` (pile, rôles, migrations, invariants
ADR 011), `task run` puis le `curl` du §1.

> ⚠️ **Cette ligne a été trop généreuse pour `task up`, et il faut savoir pourquoi.** Elle disait
> vrai sur un cluster où les rôles avaient déjà été fabriqués à la main pendant la mise au point.
> Sur un **volume neuf**, la commande échouait — quatre défauts en cascade, dont l'un où
> `verify.sql` refusait l'état que `provision.sql` documentait en commentaire (issue #84).
>
> Corrigé, et re-vérifié le **2026-07-28** depuis `task reset`, c'est-à-dire volume détruit :
> `provision → db:credentials → migrate:up → db:verify`, code de retour 0. La leçon est la même
> que pour les gardes : **un environnement déjà amorcé ne prouve rien sur l'amorçage.**

C'est ce qui referme **F001** (Docker absent) et **F005** (`-race` hors CI). Et **F008** ne se pose
plus : sous WSL, il n'y a pas de dossier protégé par Windows Defender.

### Une seule variable est obligatoire

`SECURITY_ENCRYPTION_KEY` — 32 octets en base64, exactement. Le socle impose AES-256 et **refuse de
démarrer** sur toute autre longueur : une clé de 16 octets produirait silencieusement de l'AES-128.

```bash
openssl rand -base64 32
```

Sans elle, le démarrage échoue en **nommant la variable** :

```
démarrage impossible: configuration: environment variables required by the
configuration and not defined: SECURITY_ENCRYPTION_KEY
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

## 2. Où en est le travail — état du 2026-07-27

### La CI a tourné pour la PREMIÈRE fois, et elle est verte

**13 jobs, tous verts.** C'était la 67ᵉ exécution du dépôt : les 66 précédentes rendaient
`startup_failure` en 0 seconde, depuis le tout premier commit. Cause : un verrou de facturation sur
le compte, levé depuis. Issue #47, close.

```
✓ Format, vet & lint   ✓ Guardrails d'architecture   ✓ Tests unitaires
✓ Migrations & isolation   ✓ Tests end-to-end   ✓ Fuite de secrets
✓ 4 builds croisés   ✓ 2 images   ✓ govulncheck   ✓ CI (gate)
```

> ⚠️ **La CI ne garde pas encore les PR de la pile.** `ci.yml` se déclenche sur
> `pull_request: branches: [main]`, or nos PR visent `refactor/21-modules-a-pilotes` :
> `no checks reported`. Vérifié. À corriger avant de compter dessus.

### Branche courante

`refactor/21-modules-a-pilotes`, poussée, **9 commits ajoutés le 2026-07-27** :

| Commit | Sujet |
|---|---|
| `fix(ci)` | garde d'isolation des schémas — il rendait 9 faux positifs (#40) |
| `docs(adr)` | ADR 013 : un garde est livré avec le cas qui le fait échouer (#41) |
| `docs(tech)` | `parite-frameworks.md` remis en accord avec le réel (#42) |
| `fix(ci)` | clé de chiffrement figée dans `ci.yml` (#50) |
| `chore(ci)` | `task ci` — la barrière hors GitHub, 10 jobs sur 12 (#51) |
| `fix(ci)` | `task rename` laissait le chemin de module derrière lui (#48) |
| `fix(data)` | **la chaîne de migration ne s'exécutait nulle part** (#39) |
| `chore(toolbox)` | l'outillage en conteneur, rien sur le poste (#38) |
| `fix(ci)` | crochets git rendus exécutables, donc actifs (#43) |

Base : `refactor/19-module-noyau-metier` (PR #20, **non fusionnée**). `main` ne contient toujours
ni le renommage `features` → `modules`, ni la configuration par fichiers.

### Ce que ces neuf commits ont réparé

Huit gardes existaient **et aucun ne fonctionnait**. La liste complète et datée est dans l'ADR 013 ;
les plus coûteux :

- la **chaîne de migration** échouait dès sa première commande, en local comme en CI. Quatre
  défauts en cascade, chacun masqué par le précédent ;
- les **crochets git** étaient versionnés en `100644`, donc ignorés par git sur toutes les machines ;
- le **niveau `e2e`** exécutait zéro test en affichant `ok`.

### 🔴 Quatre commits poussés portent toujours de mauvais numéros d'issue

Inchangé. Ils fermeraient des issues **non terminées** à la fusion. L'historique est poussé : la
correction se fait dans le **corps de la PR**, ou en écrasant à la fusion.

| Commit | Écrit | Devrait être |
|---|---|---|
| `1c1e4d6` migrations, schéma `platform` | `Closes #2` | `Closes #5` |
| `2180e67` dépileur d'outbox | `Closes #10` | `Closes #9` |
| `24d9038` tranche verticale | `Refs #5 #8` | `Refs #7 #10` |
| `3c8b350` surface HTTP | `Refs #5` | `Refs #7` |

### Deux PR ouvertes

| PR | État |
|---|---|
| **#60** `docs(produit)` personas, périmètre, matrice par version | **En attente de validation** — elle grave qui on sert et ce qu'on ne fera jamais. À ne pas fusionner seul |
| **#20** `refactor(cadre)` features → modules | Non fusionnée, `main` très en retard |

## 3. Comment lire un garde — la seule habitude qui compte

> **Vérifier le CODE DE RETOUR, jamais seulement la sortie.**

Toute la famille de défauts la plus coûteuse de ce dépôt vient de là. Trois occurrences réelles :

- `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et affiche `ok`. Le job CI était
  vert sans rien vérifier, pendant toute la phase 0.
- `arch-go` cherche `arch-go.yml` **sans point**. Le fichier s'appelait `.arch-go.yml` : l'outil
  n'avait **jamais** pu le lire.
- Filtrer la sortie d'`arch-go` sur `Failed|Compliance` **masque un échec `COVERAGE`** — c'est
  exactement comme ça que le piège a été manqué une fois de plus, tout récemment.

La liste complète est dans [`documentation/AMORCAGE.md`](../../../AMORCAGE.md) § « Pièges d'outillage découverts ».
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

1. **#58 — le garde `inertia` signale `.githooks/commit-msg`**, le fichier qui *porte* la règle.
   **Bloquera la fusion vers `main`** — mesuré, pas supposé. Il lui faut une liste d'exclusion
   nommée et motivée (ADR 013 § modalité 4).
2. **Faire déclencher la CI sur les PR de la pile** — `branches: [main]` les exclut aujourd'hui.
3. **Classer les 75 paquets en public / interne.** Préalable à #16, **ne déplace aucun fichier**,
   et irréversible une fois publié : extraire 530 identifiants exportés sans les classer publierait
   une API par accident.
4. **#8 — deuxième surface CLI.** Prouve la promesse « N frontends », qui n'a qu'**une** instance,
   et livre l'équivalent d'`artisan`.
5. **#37 — tests d'intégration** des 8 paquets de pilotes, débloqué par la toolbox.
6. **#13 — brancher `telemetry.Setup`**, appelée nulle part : l'observabilité n'existe pas.

## 6. Ce qui attend une décision — et qui ne se prend pas seul

Le **cadrage produit** existe désormais : [`documentation/produit/personas.md`](../produit/personas.md),
en PR #60. Cinq personas, dont **P1 primaire — l'équipe produit d'ImpactOne, confirmée** — et une
persona **refusée**. Une grille de 16 questions sourcée : **10 verdicts rouges sur 16**. Une matrice
persona × version.

Deux conclusions à ne pas reperdre :

- **`v1.0` se définit par trois mots : « sans le modifier ».** Tant qu'ajouter un module métier
  exige de toucher `internal/config/config.go` et `internal/config/modules.go`, il n'y a pas de
  version 1, quel que soit le nombre de modules livrés.
- **P1 est bloquée sur trois points** : créer un module, brancher un module, configurer un module.

| Réf | Question ouverte |
|---|---|
| **PR #60** | Validation du périmètre produit — personas, matrice, ligne « ⛔ jamais » |
| **#61** | **Licence.** Le `Dockerfile` déclare MIT, aucun `LICENSE` n'existe. MIT confirmée ou Apache-2.0 ? Quel titulaire ? Bloquant **avant publication**, plus urgent depuis le retour en privé |
| **H1** | P1 évolue-t-elle en contexte mobile-first / paiement mobile (XOF) ? Change l'ordre de `notification` et `ratelimit` |
| **H3** | `v0.1.0` : jalon interne ou annonce publique ? |
| **#18** | Protection de branche — **rouvre** : dépôt repassé privé, rulesets en `403` |
| **#11** `auth` | Session cookie ou jeton porteur, par surface ? Fournisseur externe ou magasin interne ? `rbac`, `abac`, ou ReBAC ? |
| **#23** `tenancy` | Quel modèle de multi-locataire ? **Bloque #6** |
| **#34** | Traduction : **pas une rupture d'API** — les 530 identifiants exportés sont déjà en anglais. Seuls le godoc et `rules/` sont en français |
| **#36** | SQLite comme pilote SQL par défaut ? |

## 7. Les invariants qu'il ne faut pas rouvrir

Ils ont chacun coûté une correction. La liste complète et datée est dans
[`documentation/AMORCAGE.md`](../../../AMORCAGE.md) § « Invariants à ne pas réapprendre » et § « Campagne de
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
