# ADR 003 — Un port est un type fonction, jamais une interface

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

L'expression habituelle d'un port en Go est une interface : `type UserRepository interface { Save;
FindByID; FindByEmail; Delete; List }`. Trois problèmes se manifestent systématiquement :

- Un cas d'usage qui n'utilise que `Save` dépend quand même des cinq méthodes. C'est la violation
  d'*interface segregation* la plus courante, et elle est invisible.
- Chaque test exige un double implémentant les cinq méthodes — d'où l'arrivée d'un générateur de
  mocks, donc d'une dépendance et d'une étape de génération.
- Personne n'implémente jamais l'interface une seconde fois : elle documente sans contraindre.

## Décision

Un port est un **alias de type fonction** déclaré dans `ports/` :

```go
type SaveUser = func(context.Context, domain.User) result.Result[domain.User, domain.Error]
```

- Un port = **une** opération. C'est la plus petite interface possible.
- **Alias** (`=`), pas définition : un adaptateur retourne une closure sans conversion, et deux
  ports de même signature restent substituables.
- Le paquet `ports/` ne contient **ni struct, ni fonction, ni interface** — uniquement des
  déclarations de types (`arch-go` le vérifie).
- Chaque port secondaire porte, en commentaire, son **contrat d'erreur** : la liste des
  `ErrorCode` que toute implémentation doit produire. C'est la forme opérationnelle de la
  substituabilité de Liskov.

Une interface reste admise **uniquement** si un algorithme a besoin de plusieurs opérations
indissociables sur le même état. Elle est alors déclarée côté consommateur, jamais dans `domain/`
ni `ports/`.

## Conséquences

### Ce que ça achète

- Un double de test est une **closure de trois lignes**. Aucune bibliothèque de mock nécessaire —
  et donc interdite ([`rules/dependances.md`](../../rules/dependances.md)).
- La dépendance d'un cas d'usage est exactement ce qu'il utilise, visible dans sa struct `Deps`.
- La composition et la décoration deviennent naturelles : un décorateur est `func(P) P`.

### Ce que ça coûte

- Plus de déclarations : cinq types plutôt qu'une interface à cinq méthodes.
- Un adaptateur secondaire expose N constructeurs (`NewSaveUser`, `NewEmailIsTaken`) qui partagent
  souvent le même pool — une légère répétition assumée.
- Le contrat d'erreur est du **commentaire** : seul le test de conformité le rend contraignant.

### Ce que ça rend impossible

- Injecter « le repository » dans un cas d'usage sans savoir ce qu'il en utilise.
- Générer des mocks.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| Interface large côté implémentation | Couplage large, mocks générés, ISP violée |
| Interfaces mono-méthode | Équivalent fonctionnel, mais impose un type nommé et un receveur inutiles |
| Struct de fonctions comme port | Regroupe ce qui doit rester séparable ; ramène le problème de l'interface large |

## Garde

`arch-go` : `shouldNotContainInterfaces` sur `domain/`, `shouldNotContainFunctions` et
`shouldNotContainStructs` sur `ports/`. Test de conformité de port pour le contrat d'erreur.
