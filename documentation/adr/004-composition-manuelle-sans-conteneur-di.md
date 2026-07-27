# ADR 004 — Composition manuelle, sans conteneur d'injection

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Un graphe de dépendances de quelques dizaines de nœuds pousse naturellement vers un conteneur :
`wire` (génération à la compilation), `fx` ou `dig` (résolution par réflexion au démarrage).

Mais le socle a fait de la **fonction** son unité de composition ([ADR 003](003-ports-comme-types-fonction.md)).
Un conteneur résout des **types**, pas des fonctions : deux ports de même signature — cas fréquent
et voulu — lui sont indiscernables.

## Décision

Les dépendances sont câblées **à la main**, par application partielle explicite, à deux niveaux :

1. `internal/modules/{f}/module.go` — assemble une feature et expose ses ports primaires.
2. `cmd/{server,worker,cli}/main.go` — lit la config, ouvre les connexions, construit les modules,
   monte les adaptateurs primaires, gère l'arrêt propre.

```go
register := application.NewRegisterUser(deps)
register  = application.Apply(register,
    application.WithTransaction(runInTx),
    application.WithTracing(tracer),
    application.WithLogging(logger),
)
```

Aucun conteneur d'injection n'est autorisé.

## Conséquences

### Ce que ça achète

- **Le compilateur vérifie le graphe.** Une dépendance manquante est une erreur de compilation, pas
  une panique au démarrage ni un message de résolution à l'exécution.
- La composition manuelle **est** de la programmation fonctionnelle : c'est littéralement de
  l'application partielle. Elle se lit de haut en bas, sans convention cachée.
- L'ordre des décorateurs est **explicite et relisible** — or cet ordre a un sens (la transaction
  doit envelopper la trace, sinon un rollback n'apparaît pas dans le span).
- Zéro dépendance, zéro étape de génération, zéro réflexion.

### Ce que ça coûte

- `main.go` grossit avec le nombre de features. C'est le seul fichier long du dépôt, et les
  linters l'exemptent explicitement (`funlen`, `cyclop`, `gochecknoglobals`).
- Ajouter une dépendance à un cas d'usage impose de toucher `module.go`. C'est voulu : ce coût est
  le signal qu'on ajoute un effet.

### Ce que ça rend impossible

- Découvrir une implémentation « automatiquement » par son type.
- Faire dépendre le démarrage d'un graphe résolu à l'exécution.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| `google/wire` | Résout par type : deux ports de même signature sont ambigus. Ajoute une étape de génération et un concept (Provider Sets) |
| `uber/fx` | Réflexion, erreurs au démarrage plutôt qu'à la compilation, cycle de vie concurrent de celui de `cmd/` |
| `uber/dig` | Mêmes défauts que `fx`, sans le cycle de vie |
| Singletons / variables de paquet | Interdits par `gochecknoglobals` ; rendent les tests dépendants de l'ordre |

## Garde

`gochecknoglobals`, `gochecknoinits` ; `rules/dependances.md` liste `fx`, `dig` et `wire` comme
interdits par nom. Revue humaine pour l'ordre des décorateurs.
