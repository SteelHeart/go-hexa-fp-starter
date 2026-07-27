# Outillage

## 1. Une seule commande à connaître

```bash
task check      # fmt · vet · lint · arch · test · vuln — identique à la CI
```

Tout le reste est dans `task --list-all`. Si `task check` passe en local et que la CI échoue, c'est
un défaut du **Taskfile**, à corriger : les deux doivent être la même barrière.

## 2. Docker n'est pas requis pour développer

C'est une contrainte de conception, pas un accident : **l'intégralité du cœur se teste sans
conteneur.** Sur une machine sans Docker :

| Commande | Fonctionne sans Docker | Remarque |
|---|---|---|
| `go build ./...` | ✅ | |
| `go vet ./...` | ✅ | |
| `golangci-lint run` | ✅ | |
| `arch-go` | ✅ | |
| `go test ./...` | ✅ | **aucun test sans tag n'exige de service** |
| `task check` | ✅ | c'est la barrière complète |
| `task up`, `task migrate:*` | ❌ | nécessite la pile locale |
| `go test -tags=integration` | ❌ | Postgres réel |
| `go test -tags=e2e` | ❌ | pile complète |

Les niveaux qui exigent des services sont **couverts par la CI**, qui les fournit en `services:`.
Une PR peut donc être écrite et vérifiée sur une machine sans Docker, et être bloquée par la CI si
elle casse un test d'intégration : c'est le fonctionnement attendu, pas une défaillance.

> Corollaire à ne pas oublier : sur une machine sans Docker, `task check` **ne prouve pas** que le
> SQL est correct. Ne pas écrire « vérifié » dans une PR sur cette seule base
> ([`README.md`](README.md) § règle d'or 2).

## 3. Installation

```bash
task init     # copie .env.example → .env, installe l'outillage
task tools    # golangci-lint, arch-go, goose, govulncheck
```

Prérequis : Go (version de `go.mod`) et [Task](https://taskfile.dev). Docker uniquement pour la
pile locale.

**Sur un poste où l'on ne veut rien installer**, ces prérequis tombent :
[`deploy/toolbox/`](../deploy/toolbox/README.md) porte l'outillage en image et le poste ne reçoit
qu'un script `sh`. Toute commande se préfixe alors par `./deploy/toolbox/tb`.

## 4. Crochets Git

```bash
git config core.hooksPath .githooks   # à faire une fois après le clone
```

| Crochet | Rôle |
|---|---|
| `commit-msg` | refuse un message non conforme et toute mention d'outillage d'assistance |
| `pre-push` | refuse un push direct sur le tronc |

⚠️ **Contournables avec `--no-verify`.** Ce sont des filets contre l'accident, pas des contrôles.
Le contrôle réel est le **ruleset serveur** plus la CI.

## 4 bis. Un garde est livré avec le cas qui le fait échouer

> Décision de référence : [ADR 013](../documentation/adr/013-un-garde-doit-savoir-echouer.md).

**Un garde qui n'a jamais rougi n'a pas été vérifié : il a seulement été écrit.**

Le 2026-07-27, huit gardes de ce dépôt ont été allumés pour la première fois — **les huit étaient
défectueux**. Les 12 jobs de la CI n'avaient jamais démarré (66 exécutions, 66 `startup_failure`) ;
les deux crochets étaient versionnés non exécutables, donc ignorés par git ; le garde d'isolation
rendait 9 faux positifs ; le garde `inertia` se signalait lui-même ; `task rename` laissait deux
fichiers derrière lui ; le niveau `e2e` exécutait zéro test en affichant `ok`.

Dans chaque cas le dispositif *paraissait* en place. Ce qui manquait était la seule chose qui
distingue un garde d'un décor : **l'avoir vu refuser quelque chose.**

En pratique :

| Obligation | Pourquoi |
|---|---|
| Le cas d'échec est **versionné et exécuté** | Un garde sans cas d'échec est incomplet, comme un correctif sans test de non-régression |
| Le **code de retour** fait foi, jamais la sortie | Voir §5 ci-dessous |
| Quand un compte nul est indiscernable d'un succès, le garde **compte** | `e2e` compte ses `=== RUN` et échoue à zéro |
| Un garde **ne s'applique pas aux fichiers qui le définissent** | L'exclusion est nommée et motivée dans le code du garde — jamais un assouplissement du motif |
| Toute exception porte **son motif écrit à côté** | Une liste d'exception non motivée grossit toujours, et le garde finit par ne plus rien garder |

## 4 ter. `task ci` — la barrière complète, hors CI

`task check` couvre 4 des 12 jobs. **`task ci` en rejoue 10**, dans la toolbox, sous un code de
retour unique : migrations avec `Down` réellement exécuté, `e2e` avec les tests comptés, compilation
croisée, `gitleaks`, images. Les deux restants dépendent du contexte d'une PR :
`task ci:pr-title -- "…"` et `task ci:inertia -- <base>`.

> ⚠️ **`task ci` n'est pas une garde.** Elle s'exécute sur la machine de l'auteur, qui peut ne pas la
> lancer. C'est un **instrument de mesure reproductible**, pas un contrôle — même distinction qu'au
> §4 entre le crochet local et le ruleset serveur.

## 5. Le piège du faux vert

Une commande qui n'a pas tourné rend une sortie vide, ce qui ressemble à « propre ».
**Toujours vérifier le code de retour.**

Cas déjà rencontrés dans ce type de socle :

- `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et affiche `ok`.
- `grep` dans un dossier vide retourne 1 et une sortie vide, indiscernable d'un « rien à signaler »
  si l'on ne teste pas le code de retour.
- `golangci-lint` absent du `PATH` : `task lint` échoue en 127, ce qui n'est pas « pas de warning ».

## 6. Reproductibilité

- La version de Go vient de `go.mod`, lue par la CI (`go-version-file`) — impossible de diverger.
- `go.sum` est versionné et vérifié : `go mod tidy` doit être un *no-op* en CI, sinon échec.
- Les builds sont `-trimpath`, `CGO_ENABLED=0`, avec version et commit injectés par `-ldflags`.
- Les images sont référencées par **digest** en déploiement, jamais par tag mobile.

## 7. Changer le nom du module

Le chemin de module est la **seule** valeur nominative du dépôt. Pour repartir de ce socle :

```bash
task rename -- github.com/{org}/{projet}
```

La tâche réécrit `go.mod`, tous les imports, et les références dans l'outillage, puis vérifie que
tout compile. C'est ce qui rend le socle indépendant de son dépôt d'origine.
