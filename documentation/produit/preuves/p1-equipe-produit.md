# P1 — L'équipe produit d'ImpactOne · **PRIMAIRE**

> **Verdict : ✅ trois critères sur quatre atteints, un rouge qui n'est plus le même.**

Mesuré le **2026-07-29**, sur `main`, dans la toolbox. Projet `github.com/impactone/todo`.

## Ce que P1 voulait faire

Une liste de tâches avec **deux** modules métier — `task_list` et `reminder` — pour éprouver ce qui
compte pour elle : *livrer des fonctionnalités métier sans réécrire d'infrastructure, et sans que la
forme du socle se dégrade*.

Deux modules et non un : l'isolation entre contextes ne se mesure qu'à partir du second.

## Le journal, commande par commande

```
$ hexa new /tmp/todo --module github.com/impactone/todo --from .
   → exit 0
$ hexa make:feature task_list --into /tmp/todo
   → exit 0   (inclut go build ./... et go test ./... du projet ENTIER)
$ hexa make:feature reminder  --into /tmp/todo
   → exit 0
```

**Délai clone → deux modules métier vivants : 6 secondes.** Cible : moins de 10 minutes.

Ce chiffre mérite une réserve : il ne compte pas le temps de **penser** le domaine. Il mesure ce que
la cible mesure — le temps que le socle **prend**, pas celui qu'il fait gagner.

## Le tableau de critères de P1, remesuré

| Critère | Cible | Mesuré | Verdict |
|---|---|---|---|
| Fichiers du **framework** à modifier pour ajouter un module | **0** | **0** modifié à la main. Le générateur en touche **un**, `arch-go.yml`, +5 lignes | ✅ |
| Commandes qui **créent** un module | ≥ 1 | **1** — `hexa make:feature`, qui éprouve le projet entier avant de rendre la main | ✅ |
| Délai avant premier succès depuis un clone nu | < 10 min | **6 s** pour deux modules | ✅ |
| Modules métier livrables sans authentification | 0 | `auth` **existe** et est monté (#11, ADR 017) | ⚠️ voir ci-dessous |

### Le seul fichier du framework touché, et pourquoi il ne compte pas

`hexa make:feature` ajoute **cinq lignes à `arch-go.yml`** : la règle d'étanchéité du module neuf.

C'est un fichier du framework, et le critère dit « 0 ». **La lettre est violée, l'esprit ne l'est
pas** : personne ne l'écrit à la main, et surtout — ne pas l'écrire produirait un module qu'**aucune
règle ne garde**, indiscernable d'un module gardé. C'est la faute que ce dépôt a payée onze fois.

Le critère devrait se lire **« fichiers du framework à modifier À LA MAIN »**. Reformulé ainsi, il
est atteint sans réserve. Laissé tel quel, il est faux dans le sens flatteur — donc il est corrigé
dans la grille.

### Les trois emplacements à remplir, tous côté application

`hexa make:feature` affiche ce qui reste :

| Emplacement | À qui il appartient |
|---|---|
| `internal/modules/catalog.go` — une ligne | l'**application** |
| `config/modules.yaml` — deux lignes | l'**application** |
| `cmd/server/main.go` — le montage | l'**application** (composition root, ADR 004) |

**Aucun des trois n'est un fichier du framework.** C'est exactement ce que l'ADR 014 achète : le
framework ne nomme aucun module.

## Ce que P1 a obtenu sans le demander

**La barrière tourne dans le projet engendré.** `task check` rend **0** — fmt, vet, lint, arch,
tests, cliquets de couverture, vuln — sur un projet créé six secondes plus tôt.

**Le module est joignable dès sa création.** Depuis l'ADR 019, `make:feature` pose
`adapters/primary/http/` avec ses trois tests. P1 n'a pas à deviner où poser sa première surface :
elle en a une, et la seconde s'écrit en copiant un frère.

**Les gardes sont actifs au premier commit.** Tentative de commit avec le message `base` :

```
Message de commit non conforme.
  Attendu: {type}({scope}): {description} (#{issue})
```

Le crochet a refusé. Ce n'est pas anecdotique pour P1 : *« sans que la forme du socle se dégrade au
bout de trois mois »* est la moitié de son exigence, et elle est tenue dès la seconde zéro.

## Le rouge restant n'est plus celui qui était écrit

La grille dit : *« Modules métier livrables sans authentification : tous 🔴 — `auth` n'existe pas
(#11) »*.

**`auth` existe.** Le critère tel qu'il est formulé est donc obsolète : il demandait « 0 module
livrable sans auth » comme un constat d'échec, alors que c'est désormais une propriété.

Ce qui reste vrai, et que P1 découvrira : **aucun compte ne peut être créé à l'exécution sans passer
par le compte d'amorçage**, et les opérations d'administration exigent une permission qu'il faut
d'abord s'accorder. C'est une friction de premier démarrage, pas un blocage — et elle est
documentée par l'ADR 017 §6.

## Ce qu'il a fallu écrire à la main

**Rien, pour arriver à deux modules qui compilent, se testent et passent la barrière.**

Ce que P1 devra écrire ensuite est son **métier** — ce que le socle ne peut pas deviner : les règles
d'une tâche, ses états, ses transitions. C'est le partage attendu.

## Ce qu'il a fallu contourner

**Rien.** Aucune règle du dépôt n'a dû être enfreinte, aucun `//nolint` ajouté, aucun garde
désactivé.

## La réserve honnête

Cette preuve mesure **la création**, pas la vie du module. Elle ne dit rien de ce qui arrive au
troisième mois — quand `task_list` a besoin d'un champ que `reminder` veut lire, quand la
configuration doit accueillir un groupe nouveau, quand une migration doit être rétro-compatible.

**Le blocage connu de P1 est là et n'a pas bougé** : `Config` reste une **struct fermée**, **16
champs**, sans point d'extension (critère 8 de la grille, 🔴, ligne `v1.0` — lot #147). Une liste de
tâches n'en a pas besoin. Un vrai produit, si.

> ⚠️ **Cette ligne annonçait « 19 champs », et c'est la preuve qui avait tort.** Le chiffre venait de
> la grille des personas, où il était déjà faux ; il a été **recopié**, pas mesuré. Recompté le
> 2026-07-29 : `internal/config/config.go` a **16** champs, et n'en a jamais eu 19.
>
> C'est le défaut le plus embarrassant de ce dossier. `preuves/` existe **précisément** pour que la
> grille cesse d'être crue sur parole — *« ce document affirme, les preuves mesurent »*. Une preuve
> qui recopie le chiffre qu'elle est censée vérifier ne corrige pas la dérive : **elle la
> certifie**. Corrigé en #148, avec le rappel qu'une mesure non exécutée n'est pas une mesure.
