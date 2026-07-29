# ADR 019 — L'anatomie nomme ses adaptateurs, et le générateur en pose un

- **Statut** : acceptée
- **Date** : 2026-07-29
- **Issue** : [#116](https://github.com/SteelHeart/go-hexa-fp-starter/issues/116)
- **Remplace** : la section « Une seule anatomie » de l'[ADR 012](012-anatomie-d-un-module-et-pilotes.md).
  Tout le reste de l'ADR 012 — pilotes, zéro prérequis, vocabulaire imposé — reste **en vigueur**.

## Contexte

L'audit [#107](https://github.com/SteelHeart/go-hexa-fp-starter/issues/107), écart É-02, a trouvé
une contradiction entre un ADR et le code qu'il gouverne.

L'ADR 012 grave cette anatomie :

```
├── surfaces/              adaptateurs primaires optionnels : http · cli · events
```

Le dépôt n'a **aucun** dossier `surfaces/`. Il a trois `adapters/primary/`, et la carte du dépôt
documente `adapters/primary/` **et** `adapters/secondary/`.

La hiérarchie des sources est explicite : *en cas de contradiction, l'ADR gagne*. **Le code violait
donc une décision d'architecture, et la carte documentait la violation** — depuis des mois, sans que
rien ne le dise.

### Ce que la vérification a ajouté à l'écart

`hexa make:feature` n'engendre **ni `surfaces/`, ni `adapters/`**. Ses gabarits couvrent `domain/`,
`ports/`, `application/`, `drivers/memory/`, `tests/`, `module.go`, `catalog.go` — et s'arrêtent là.

Il ne reconduisait donc pas le mauvais choix : il **esquivait la question**. Conséquence bien plus
gênante que l'écart de nom : **un module engendré n'est joignable par aucune surface.** La personne
qui vient de le créer doit deviner où poser son premier adaptateur, sans que la structure ne le lui
dise.

## Décision

### 1. Les adaptateurs se nomment `adapters/primary/` et `adapters/secondary/`

```
internal/{core,modules}/{nom}/
├── domain/                règles pures
├── ports/                 contrats — types fonction uniquement
├── application/           cas d'usage + décorateurs — ABSENT si le module n'a
│                          aucune politique à orchestrer (voir §3)
├── drivers/{nom}/         le point d'extension, une implémentation par dossier
├── adapters/
│   ├── primary/{nom}/     http · cli · events — une surface par dossier
│   └── secondary/{nom}/   ce dont le cœur a besoin du monde
├── tests/                 boîte noire
├── catalog.go             les pilotes DÉCLARABLES (ADR 014)
└── module.go              composition root local
```

**C'est le code qui gagne, pas l'ADR.** Trois raisons, par ordre de poids.

**L'ADR 012 n'avait nommé qu'une moitié.** `surfaces/` ne dit rien de l'endroit où vivent les
adaptateurs secondaires — et `user_registration` en a deux, `hashing` et `outboxpub`. Un vocabulaire
qui couvre la moitié des cas force une seconde convention non écrite, ce qui est pire que pas de
convention du tout.

**`adapters/{primary,secondary}` se lit sans glossaire.** C'est le vocabulaire hexagonal canonique.
Sur un dépôt public destiné à être adopté, cela compte davantage que la concision.

**Rien n'est perdu.** Le mot **surface** reste dans le vocabulaire imposé de l'ADR 012 : c'est le nom
du *concept*. `adapters/primary/{nom}/` est l'endroit où il vit. **Un concept et un répertoire n'ont
pas à porter le même nom** — et les confondre est précisément ce qui a produit cet écart.

### 2. `hexa make:feature` pose un adaptateur HTTP RÉEL

Pas un dossier vide, pas un LISEZ-MOI : un adaptateur qui compile, qui répond, et qui a son test.

**Pourquoi un dossier vide ne suffit pas.** `user_registration` est la tranche de référence : sa
forme sera copiée pour écrire les modules suivants. **Tout dossier manquant y est reproduit comme
« pas nécessaire »** — c'est écrit dans le relevé d'état à propos de cette même tranche, et c'est
exactement ce qui se produisait ici, un cran plus tôt : le dossier n'était pas absent d'un module, il
était absent du **moule**.

**Pourquoi un adaptateur réel plutôt qu'un gabarit commenté.** La propriété n°2 du socle — *le nombre
de frontends est un non-sujet* — se démontre en ajoutant une seconde surface sans toucher au cœur.
Un module engendré sans aucune surface ne permet pas cette démonstration : il oblige à écrire la
première avant de pouvoir écrire la seconde, et c'est la première qui coûte.

Le module engendré est donc **joignable dès sa création**, et l'ajout d'une CLI ou d'un consommateur
d'événements devient l'exercice qu'il doit être : copier un frère.

### 3. `application/` est facultatif, et c'est écrit

L'ADR 012 listait `application/` sans dire qu'il l'était, à la différence de `surfaces/` qu'il
marquait « optionnels ». Quatre modules noyau n'en ont pas : `audit`, `dynconf`, `idempotency`,
`storage`.

**Dérogation retenue, avec son motif** — audit #107, écart É-09 : ces quatre-là n'ont **aucun cas
d'usage à orchestrer**. Un port d'`idempotency` est `Reserve`/`Complete`/`Release` : il n'y a ni
pipeline, ni décorateur, ni séquence. Créer un `application/` qui ne ferait que réexporter le pilote
produirait une couche de transfert pure — précisément ce que l'architecture hexagonale reproche aux
couches de service.

**La preuve que ce n'est pas de la complaisance** : les quatre modules qui ont *vraiment* une
politique — `outbox` (recul exponentiel, abandon après N essais), `scheduler` (élection),
`notification` (envoi), `auth` (authentification, autorisation) — ont **tous** leur `application/`.
Le partage suit le besoin, pas la commodité.

## Conséquences

### Outillage

`tools/verifie-veracite-doc.sh` § contrôle 3 exigeait `domain/`, `ports/`, `tests/`, `module.go`,
`catalog.go` — et **s'arrêtait là, délibérément, tant que cet ADR n'était pas écrit** : trancher dans
un garde reviendrait à décider par l'outillage ce qu'un ADR doit décider.

Cette décision étant prise, le garde peut désormais l'appliquer.

### Ce qui ne bouge pas

- Aucun dossier n'est déplacé. Le code était déjà dans la forme retenue.
- Les douze règles de dépendance d'`arch-go` indexées sur `internal.` restent intactes.
- L'ADR 012 n'est pas réécrit : il est **immuable**. Sa section « anatomie » est remplacée par
  celle-ci, son vocabulaire et ses pilotes restent en vigueur.

### Ce que ça coûte

`hexa make:feature` produit désormais plus de fichiers, donc plus à lire pour qui découvre un module
engendré. C'est le prix assumé de la propriété qu'il démontre — et il reste inférieur au coût
d'écrire la première surface à la main, sans modèle, en devinant où la poser.

## Ce qui a été écarté

| Option | Pourquoi non |
|---|---|
| Déplacer le code vers `surfaces/` | Ne nomme toujours pas les adaptateurs secondaires. Touche trois dossiers, le générateur et la carte, pour aggraver le vrai défaut de l'ADR 012 |
| Accepter les deux noms | Deux noms pour une chose — ce que `rules/references.md` § vocabulaire imposé interdit, et pour cette raison exacte : les recherches dans le code cessent d'aboutir |
| Un dossier `adapters/primary/` vide avec un LISEZ-MOI | Un dossier vide est reproduit comme « pas nécessaire ». Il déplace le problème d'un cran sans le résoudre |
| Laisser le générateur muet et le documenter | C'est l'état actuel. Il transforme une décision d'architecture en exercice de lecture de documentation |
