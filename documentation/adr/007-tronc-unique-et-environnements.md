# ADR 007 — Tronc unique ; un environnement n'est pas une branche

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Le modèle par branches d'environnement (`develop` → `uat` → `release/*` → `main`) est répandu et
paraît lisible. En pratique il produit toujours la même dérive :

- les branches divergent, et plus elles divergent, plus leur merge coûte cher, donc plus on le
  repousse — la boucle ne se stabilise que par l'abandon ;
- la CI ne sait plus quoi vérifier : `develop` et `main` ne contiennent pas le même code, et
  « vert sur `develop` » ne dit rien de la production ;
- on ne peut pas répondre à « qu'est-ce qui tourne exactement en production ? » — le sommet d'une
  branche bouge.

## Décision

**Une seule branche longue : `main`.** Toute autre branche vit moins de deux jours et meurt à son
merge.

- Branches : `{type}/{issue}-{slug-kebab}`. Jamais de branche nommée d'après une personne.
- Merge par **squash**, avec suppression de branche. Le titre de PR devient le message du commit.
- Pas de push direct sur `main`, pour personne.
- Le travail incomplet passe derrière un **drapeau de fonctionnalité**, jamais derrière une
  branche longue.

**Un environnement est un artefact déployé, pas une branche :**

| Environnement | Déclencheur | Approbation |
|---|---|---|
| UAT | merge sur `main` | automatique |
| Production | tag `v*` | **manuelle** (environnement GitHub `production`) |

Le déploiement se fait par **digest** `sha256:`, jamais par tag de registre : un tag est
réinscriptible côté registre, un digest ne l'est pas.

**Correctif urgent** : brancher depuis le **tag**, livrer, tagger, puis reporter sur `main` dans la
foulée. La CI de production refuse un tag qui n'est pas ancêtre de `main` — un correctif qui ne
revient pas sur le tronc réapparaît à la release suivante.

## Conséquences

### Ce que ça achète

- La question « qu'est-ce qui tourne ? » a une réponse exacte : un tag, et un digest.
- Un seul état à vérifier par la CI.
- Les conflits de merge deviennent rares, parce que les branches sont courtes.

### Ce que ça coûte

- **Le tronc doit rester déployable en permanence.** C'est une discipline réelle, et le prix des
  drapeaux de fonctionnalité (code mort temporaire, chemins doubles).
- Découper le travail en incréments livrables demande un effort de conception à chaque fois.
- Pas de « zone tampon » où laisser mûrir du travail incertain.

### Ce que ça rend impossible

- Accumuler trois semaines de travail hors du tronc.
- Déployer un état qui n'a pas de tag.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| GitFlow (`develop` + `release/*` + `hotfix/*`) | Conçu pour du logiciel versionné livré au client ; inadapté à un service déployé en continu |
| Branches par environnement (`uat`, `prod`) | Divergence garantie ; « qu'est-ce qui tourne » devient insoluble |
| GitHub Flow strict (déploiement au merge, sans tag) | Convient à l'UAT, mais la production a besoin d'un artefact nommé et immuable |

## Garde

Crochet `.githooks/pre-push` (filet local, contournable), **ruleset serveur** sur `main` (contrôle
réel : PR obligatoire, `CI` en statut requis, historique linéaire), environnement GitHub
`production` avec approbation, et la vérification d'ancêtre dans `deploy-production.yml`.
