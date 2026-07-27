# ADR 015 — La frontière publique est dérivée d'un usage mesuré, pas décidée d'avance

- **Statut** : Accepté
- **Date** : 2026-07-27
- **Remplace** : —
- **Issue** : [#17](https://github.com/SteelHeart/go-hexa-fp-starter/issues/17) · précise [#16](https://github.com/SteelHeart/go-hexa-fp-starter/issues/16)

## Contexte

**Aucun paquet de ce dépôt n'est importable.** Mesuré, pas déduit :

```
$ go list ./... | grep -v /internal/
    …/cmd/server
    …/cmd/worker
    …/tools/covergate
```

Deux binaires et un outil de build. Tout le reste vit sous `internal/`, que Go
interdit d'importer depuis l'extérieur du module. Une application tierce ne peut
donc consommer **aucune ligne** de ce socle.

Le seul mode de distribution possible aujourd'hui est « copier le modèle » —
c'est-à-dire un **boilerplate avec un générateur**, précisément ce dont le projet
a décidé de sortir. La décision est prise : le noyau deviendra une **bibliothèque
importée**.

Reste à savoir **quand**, et **d'après quoi**.

### Ce qu'on ne sait pas, et qui commande tout

**Aucune application n'a jamais été construite sur ce socle.**

`user_registration` vit *dedans*, pas *dessus* — le dépôt le dit lui-même : « la
tranche de référence, pas l'application ». Personne n'a donc jamais eu à écrire
`import` pour consommer ce framework.

Décider maintenant quels paquets deviennent publics reviendrait à classer
soixante-quinze paquets **au jugé**, dans un dépôt dont la règle d'or distingue
rigoureusement *prouvé* de *écrit non prouvé*. Et l'erreur se ferait dans un seul
sens : **trop large**. Un paquet publié par excès de prudence ne se retire plus
sans casser quelqu'un ; un paquet oublié s'ajoute en une ligne.

### Trois choses que la décision « bibliothèque » ne dit pas

| Confusion | Ce qui est vrai |
|---|---|
| importable **=** monorepo multi-modules | déplacer `internal/core` → `core/` suffit à rendre importable, avec **un seul** `go.mod`. Le monorepo de #16 — trois modules, trois versions, des `replace` en développement — est une décision **séparée**, et plus lourde |
| tout le socle devient bibliothèque | `user_registration` est fait pour être **recopié** : sa forme sera celle de `billing`. C'est de la matière à gabarit, pas à bibliothèque |
| la frontière est urgente | le dépôt est **privé**, et le reste (LICENSE). Rien n'est publié, donc rien n'est figé : l'irréversibilité n'a pas commencé |

### Le risque qu'on sous-estime en déplaçant les paquets

**Douze des treize règles de dépendance d'`arch-go`, et trois des cinq règles de
forme, sont indexées sur des chemins `**.internal.…`.** Déplacer les paquets les
fait toutes cesser de correspondre à quoi que ce soit.

Dans un dépôt dont la leçon fondatrice est *« onze gardes existaient, aucun ne
fonctionnait »*, c'est **le** danger de l'opération : `arch-go` afficherait
100 % de conformité en ne gardant plus rien. Exactement la forme que l'ADR 013
combat.

## Décision

**La frontière publique est dérivée de la liste d'imports d'une application
réelle. Elle ne se décrète pas avant qu'une telle application existe.**

Modalités, dans cet ordre :

1. **`hexa new` d'abord, en mode gabarit.** Il ne présuppose aucune frontière —
   il recopie et réécrit. Il livre de la valeur à l'équipe immédiatement, et il
   est la seule façon de produire l'application qui manque.

2. **Une application réelle est construite avec.** Sa liste d'imports **est** la
   mesure. Ce que personne n'importe n'est pas public ; ce qui est importé sans
   avoir été prévu est une découverte, pas un accident.

3. **Alors seulement la frontière**, par un ADR qui cite cette mesure. Le
   déplacement se limite d'abord à un seul module Go : `core/`, `pkg/`,
   `config/`, `contracts/`. Le monorepo multi-modules de #16 reste une décision
   distincte, à prendre quand `cli` et `core` évolueront à des rythmes
   différents — pas avant.

4. **Toute PR de déplacement porte son propre témoin** (ADR 013) : une violation
   d'architecture délibérée qui doit faire **rougir** `arch-go` après le
   déplacement. Une règle qui ne correspond plus à aucun paquet est un garde
   mort, et un garde mort est vert.

5. **Les modules métier ne deviennent pas une bibliothèque.** `core`, `pkg`,
   `config` et `contracts` s'importent ; `internal/modules/**` se **génère**.
   C'est le partage que font Rails et Laravel, et c'est celui que l'architecture
   de ce dépôt implique déjà.

## Conséquences

### Ce que ça achète

- **La frontière sera fondée sur du mesuré.** Le dépôt n'aura pas publié une API
  qu'aucun consommateur n'a jamais exercée — la façon la plus sûre de figer la
  mauvaise.
- **L'équipe démarre maintenant.** `hexa new` ne dépend d'aucune décision
  restante, et il produit précisément la donnée qui manque.
- **Le coût de l'erreur reste récupérable.** Tant que rien n'est publié, une
  frontière mal placée se corrige sans casser personne.
- **Les gardes d'architecture survivent au déménagement**, parce que la modalité 4
  l'exige explicitement plutôt que de l'espérer.

### Ce que ça coûte

- **Le code écrit entre-temps l'est sans connaître la frontière.** Un paquet qui
  deviendra public gagne aujourd'hui des identifiants exportés qu'il faudra
  peut-être restreindre. Coût réel, mais borné : il se paie en une revue, pas en
  rupture de compatibilité.
- **`hexa new` en mode gabarit sera réécrit** le jour où le noyau s'importe : il
  cessera de recopier `core/` pour l'ajouter en dépendance. C'est un travail
  assumé, et il est plus petit que celui d'une frontière mal placée.
- **Le socle reste un boilerplate quelque temps.** Il faut le dire tel quel — et
  ne pas l'appeler framework dans les documents avant que ce soit vrai.

### Ce que ça rend impossible

- **Promettre une compatibilité d'API avant `v1.0`.** Rien n'étant importable,
  il n'y a pas d'API à promettre. C'est cohérent avec le découpage retenu :
  `v1.0` = frontière **gelée**.
- **Publier le dépôt tant que la frontière n'est pas posée.** Publier avec tout
  sous `internal/` inviterait à des forks plutôt qu'à des dépendances — et un
  fork ne reçoit aucun correctif.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| **Classer les 75 paquets maintenant**, au jugé | Aucune application n'a jamais importé ce socle : la classification serait supposée, dans un dépôt qui distingue *prouvé* de *écrit non prouvé*. Et l'erreur irait dans le sens coûteux — un paquet publié par excès ne se retire plus |
| **Tout publier**, laisser les consommateurs choisir | Rend publique la totalité de la surface, y compris les pilotes et l'infrastructure. Chaque identifiant exporté devient une promesse. C'est la décision la plus facile aujourd'hui et la plus chère ensuite |
| **Rester en gabarit définitivement** | Un correctif du socle ne se propage à aucun projet déjà généré : il faudrait le reporter à la main autant de fois qu'il y a de projets. C'est le boilerplate que le projet a décidé de quitter, et le coût croît avec le nombre de projets — donc avec le succès |
| **Faire le monorepo #16 d'abord** | Trois modules et trois versions résolvent un problème — des rythmes d'évolution différents — dont rien ne prouve encore l'existence. Décorer une décision non prise |

## Garde

- **`go list ./... | grep -v /internal/`** est la mesure de la frontière, et elle
  est vérifiable en une commande. Le jour où elle rend plus que les binaires,
  c'est que la frontière a bougé — volontairement ou non.
- **Modalité 4 outillée** : toute PR de déplacement doit livrer un témoin qui
  fait rougir `arch-go`. Sans lui, on ne saurait pas distinguer « l'hexagone
  tient » de « les règles ne désignent plus rien ».
- **[humain]** Rien ne vérifie qu'un identifiant exporté d'un paquet destiné à
  devenir public est *délibérément* public. Tant que tout est sous `internal/`,
  l'export n'engage rien ; le jour où la frontière bouge, il engagera tout. C'est
  un aveu de faiblesse, pas une tolérance — l'ADR de frontière devra le fermer.
