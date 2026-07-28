# ADR 016 — Le générateur est une bibliothèque, pas un composition root

- **Statut** : acceptée
- **Date** : 2026-07-28
- **Issue** : [#96](https://github.com/SteelHeart/go-hexa-fp-starter/issues/96)
- **Remplace** : rien. **Précise** l'ADR 004 (composition manuelle) et prépare l'ADR 015.

## Contexte

`hexa new` et `hexa make:feature` ont été écrits directement dans `cmd/hexa`, en `package main`.
C'était le réflexe naturel — un outil en ligne de commande, un binaire — et il a produit une faute
mesurable en revue.

**Go interdit d'importer un paquet `main`.** Conséquence directe : aucun test du générateur ne
pouvait respecter `rules/tests.md`, qui ne prévoit que deux emplacements —
`{paquet}/tests/` pour l'API publique, `{paquet}/internal_test.go` pour les identifiants non
exportés. Dix fichiers de test s'étaient accumulés à la racine de `cmd/hexa`, dans un troisième
emplacement qui n'existe pas dans le règlement, et avec des identifiants en français là où tout le
reste du dépôt est en anglais.

Deux effets moins visibles, et plus coûteux :

- **`covergate` exclut `cmd/` du périmètre unitaire**, à juste titre : un composition root s'exerce
  par les tests de bout en bout. Les 900 lignes du générateur y étaient donc invisibles pour la
  mesure de couverture, alors que ce sont des fonctions pures et parfaitement testables.
- **Le composition root avait cessé d'en être un.** L'ADR 004 le définit comme *le seul code
  autorisé à tout connaître* — pas comme l'endroit où l'on écrit ce qui n'a pas trouvé de place.

## Décision

**La logique du générateur vit dans `internal/generator`, avec une API publique. `cmd/hexa` est une
coquille mince : elle déclare des drapeaux, aiguille, et imprime.**

Trois conséquences directes :

1. Les tests deviennent possibles **par l'API publique**, dans `internal/generator/tests/`, un
   fichier par test, en anglais — les trois exigences de `rules/tests.md`.
2. `covergate` les **compte**.
3. Le générateur devient **extractible**. Le jour où le socle sera une bibliothèque importable
   (ADR 015), `internal/generator` est exactement le genre de paquet qui remontera dans la frontière
   publique — ou qui partira dans un module `cli/` séparé (issue #16).

### La règle qui tient la décision

```yaml
- package: "**.internal.generator.**"
  shouldNotDependsOn:
    internal:
      - "**.internal.modules.**"
      - "**.internal.core.**"
      - "**.internal.infrastructure.**"
      - "**.internal.config.**"
```

Elle n'est pas décorative. Un générateur qui importerait un module pour « lire son catalogue »
lierait l'outil au code qu'il génère : il faudrait alors le recompiler pour générer un projet dont
les modules diffèrent. Le générateur manipule des **fichiers**, pas des modules — et cette règle
l'empêche de l'oublier.

## Conséquences

### Ce que ça achète

- Un règlement de test qui s'applique enfin au générateur, au lieu d'une exception non écrite.
- Une couverture mesurée sur 900 lignes qui ne l'étaient pas.
- Un `cmd/hexa/main.go` de 190 lignes, lisible d'un trait, dont on voit qu'il ne décide rien.

### Ce que ça coûte

- **Une frontière publique de plus à tenir.** `PlanProject`, `CreateProject`, `PlanFeature`,
  `CreateFeature`, `RenderFeature`, `DeclareIsolation`, `SplitArguments`… sont désormais des noms
  exportés. Ils ne sont pas encore *publics* au sens de l'ADR 015 — tout vit sous `internal/` — mais
  ils le deviendront peut-être, et les nommer au jugé aujourd'hui coûterait plus tard.
- **Un paquet qui n'a pas d'équivalent dans l'anatomie.** `internal/generator` n'est ni un module,
  ni de l'infrastructure, ni une primitive. C'est de l'outillage livré avec le socle. Le nommer
  autrement — `internal/tooling`, `internal/cli` — ne changerait rien à cette singularité, et le
  déguiser en autre chose serait pire.

### Ce que ça rend impossible

- Écrire de la logique dans `cmd/hexa` sans que sa non-testabilité saute aux yeux : le fichier est
  désormais si mince que toute addition y détonne.
- Que le générateur dépende de l'application. `arch-go` le refuse.

## Alternatives écartées

| Alternative | Pourquoi non |
|---|---|
| **Déplacer les tests dans `cmd/hexa/tests/`** | Ne compile pas. Go interdit d'importer un paquet `main` — c'est le fait qui déclenche toute cette ADR |
| **Tout mettre dans `cmd/hexa/internal_test.go`** | Conforme à la lettre, et mauvais : un seul fichier pour dix tests, en boîte blanche, sur du code qui n'a aucune raison d'être privé. La règle prévoit `internal_test.go` pour les *identifiants non exportés*, pas comme voie de contournement |
| **Un module Go séparé `cli/` tout de suite** | C'est l'issue #16, et la décision de séquencement tient : **stabiliser avant de restructurer**. Rien n'oblige à payer un multi-module aujourd'hui pour obtenir un paquet testable |
| **Laisser en l'état et documenter l'exception** | Une exception non outillée se propage. Le prochain outil aurait été écrit de la même façon, avec le même argument |

## Garde

- **`arch-go`** vérifie que `internal/generator` ne dépend d'aucun paquet applicatif. La règle est
  aussi ce qui le rend **couvert** au sens de l'outil : un paquet neuf sans règle fait tomber la
  couverture sous 100 %, donc échouer `task check`. Déjà rencontré en ajoutant `tools/covergate`.
- **`task ci:generateur`** génère un projet, y crée un module, constate la règle d'étanchéité, et
  relance `task check` dans le projet généré. Il a d'ailleurs attrapé cette PR en cours de route :
  les fichiers déplacés n'étaient pas encore suivis par git, donc `hexa new` — qui copie les
  fichiers **suivis** — produisait un projet auquel `internal/generator` manquait.
- **[humain]** Rien n'empêche mécaniquement d'ajouter de la logique dans `cmd/hexa`. La minceur du
  fichier est une incitation, pas une garde. Un seuil de lignes sur `cmd/**` existe déjà dans
  `arch-go` — il a d'ailleurs attrapé une régression le même jour — mais il borne les fonctions, pas
  les fichiers.
