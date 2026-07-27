# Process — comment on travaille

Ce dossier ne contient **aucune décision d'architecture** : elles vivent dans
[`documentation/adr/`](../adr/README.md). Il contient les conventions de travail, celles qu'on
applique dix fois par jour sans vouloir y repenser.

| Document | Ce qu'il tranche |
|---|---|
| [`REPRISE.md`](REPRISE.md) | **Poste neuf ou reprise** : amorçage vérifié, travail en cours, ordre de fusion, arbitrages en attente |
| [`NOMENCLATURE.md`](NOMENCLATURE.md) | Nommer une branche, un commit, une PR, un fichier, un test |
| [`LABELS.md`](LABELS.md) | Le jeu de labels GitHub, et ce que chacun engage |
| [`JOURNAL_FRICTION.md`](JOURNAL_FRICTION.md) | Ce qui freine, **assumé et daté**, plutôt que subi en silence |

## Le journal de friction mérite une explication

Une friction est un obstacle qu'on a **choisi de ne pas lever**, avec la raison et la date. Ce n'est
ni une liste de tâches, ni une excuse : c'est ce qui empêche de redécouvrir dans six mois pourquoi
les tests d'intégration ne tournent qu'en CI, et de « corriger » en cassant l'arbitrage qui l'a
décidé.

Une friction non écrite se paie deux fois : une fois quand on la subit, une fois quand quelqu'un
d'autre la re-débat.

## Le rapport avec le reste

- **`rules/`** dit ce qui est **obligatoire** et nomme l'outil qui refuse la violation.
- **`documentation/adr/`** dit **pourquoi** une décision structurante a été prise, et fait foi.
- **`documentation/process/`** dit **comment** on s'y prend au quotidien.
- **`documentation/technique/`** décrit des **cibles de conception** — modules noyau, pilotes,
  parité avec les frameworks mûrs. Ces documents distinguent toujours ce qui existe de ce qui est
  prévu.

En cas de contradiction, l'ordre de préséance est : ADR, puis `rules/`, puis le reste.
