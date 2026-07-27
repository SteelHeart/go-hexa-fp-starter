# Produit

> Le **pour qui**, dont découle le **quoi**. Ce dossier n'existait pas : le dépôt avait `adr/`,
> `process/`, `securite/` et `technique/`, et rien sur le produit. Cinq arbitrages en dépendaient
> sans que ce soit visible.

| Fichier | Portée |
|---|---|
| [`personas.md`](personas.md) | Les cinq personas, dont une **primaire** et une **refusée** · la grille de 15 questions remplie sur l'état réel · la **matrice persona × version** |

## Comment s'en servir

**Quand un arbitrage bloque**, on interroge ce dossier avant d'ouvrir un ADR. Une décision
d'architecture qui ne sert aucune persona n'a pas à être prise ; une décision qui en sert deux au
détriment de la primaire doit être justifiée par écrit.

**Quand une demande arrive**, deux questions dans cet ordre :

1. Quelle persona sert-elle ? Si la réponse est **P0**, c'est hors périmètre, sans débat.
2. Dans quelle version entre-t-elle ? Si la réponse est « celle en cours » alors qu'elle sert une
   persona de `v2.0`, elle attend.

## Ce que ce dossier ne fait pas

Il ne **constate** pas l'état du dépôt — c'est [`CLAUDE.md`](../../CLAUDE.md) § « État réel » qui
fait foi sur les faits. Il ne **compare** pas aux autres frameworks — c'est
[`parite-frameworks.md`](../technique/parite-frameworks.md). Il **définit le périmètre**, et il est
le seul à le faire.

En cas de contradiction avec un ADR, **l'ADR gagne** : une décision d'architecture tranchée prime
sur une intention de produit. Mais le désaccord doit alors être écrit dans les deux documents, dans
la même PR — c'est le signe que la décision a été prise contre la cible, et ça se relit.
