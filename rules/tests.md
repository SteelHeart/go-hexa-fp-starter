# Tests

## 1. Le test qui compte : le cœur sans I/O

> **Si un test de `domain/` ou `application/` a besoin de Docker, l'architecture est violée.**

C'est le seul bénéfice concret de tout ce cadre : les règles métier se testent en microsecondes,
sans base, sans réseau, sans conteneur. Les doubles sont des **closures**, pas des mocks générés.

```go
deps := application.Deps{
    EmailIsTaken: func(context.Context, domain.Email) result.Result[bool, domain.Error] {
        return result.Ok[bool, domain.Error](true)
    },
    Now:        func() time.Time { return fixedTime },
    GenerateID: func() domain.UserID { return fixedID },
}
```

`Now` et `GenerateID` sont des ports **précisément pour ça** : un test déterministe n'a ni horloge
ni aléa.

## 2. Pyramide et emplacements

| Niveau | Emplacement | Dépendances | Quand |
|---|---|---|---|
| **Domaine** | `internal/features/{f}/domain/*_test.go` | aucune | chaque règle, chaque cas limite |
| **Cas d'usage** | `internal/features/{f}/application/*_test.go` | closures | chaque chemin, nominal **et** d'erreur |
| **Primitives** | `internal/pkg/*/…_test.go` | aucune | lois algébriques incluses |
| **Adaptateur secondaire** | `…/secondary/{x}/*_test.go` (tag `integration`) | Postgres réel | traduction d'erreur, SQL |
| **Conformité de port** | `…/ports/conformance_test.go` (tag `integration`) | toutes les implémentations | contrat d'erreur du port |
| **Bout en bout** | `tests/e2e/` (tag `e2e`) | pile complète | un chemin critique par surface |
| **Charge** | `tests/perf/` (k6) | pile complète | avant une mise en production |

`go test ./...` **sans tag** ne doit exiger aucun service. C'est ce qui permet de travailler sans
Docker sur la machine ([`toolchain.md`](toolchain.md)).

## 3. Obligations

- **Un bug corrigé porte son test de non-régression**, écrit avant le correctif et échouant sans lui.
- **Toute règle métier a un test qui porte son nom** : la traçabilité règle ↔ test doit se lire sans
  ouvrir les fichiers.
- **Chaque `ErrorCode` est atteint par au moins un test.** Un code d'erreur jamais produit est du
  code mort qui inspire confiance.
- **Tests de table** pour les règles à variantes ; sous-tests nommés en langage métier
  (`"refuse un email sans domaine"`), pas en jargon technique.
- `-race` et `-shuffle=on` **toujours**. Un test qui ne passe qu'en ordre fixe cache une dépendance
  d'état.
- Un test **ne réimplémente jamais** la logique testée : il énonce le résultat attendu en dur.
- Aucun `time.Sleep` dans un test. Attendre une condition, ou injecter l'horloge.

## 4. Conformité de port — la substituabilité, testée

Un port secondaire déclare son contrat d'erreur ([`ports-et-contrats.md`](ports-et-contrats.md)).
Ce contrat est vérifié par **une seule suite**, exécutée contre **toutes** les implémentations :

```go
func TestSaveUserConformance(t *testing.T) {
    for name, build := range map[string]func(*testing.T) ports.SaveUser{
        "postgres": newPostgresSaveUser,
        "memory":   newMemorySaveUser,
    } { … }
}
```

Ajouter une implémentation sans l'inscrire dans cette table est un défaut de revue.

## 5. Cliquets de couverture

| Portée | Seuil | Garde |
|---|---|---|
| Global | **70 %** | CI, job `test` |
| `domain/` + `application/` | **90 %** | CI, job `test` |

Les seuils **montent, ne descendent jamais**. Les abaisser exige un ADR — et c'est le genre de
décision qu'on n'écrit pas volontiers, ce qui est exactement l'effet recherché.

La couverture mesure ce qui est **exécuté**, pas ce qui est **vérifié** : 90 % sur le cœur est un
plancher, pas un objectif atteint.

## 6. Le piège du faux vert

Une commande qui n'a pas tourné (binaire absent, tag oublié, chemin vide) rend une sortie vide, ce
qui ressemble à « propre ». **Vérifier le code de retour, pas seulement la sortie.**

`go test ./tests/e2e/...` sans `-tags=e2e` compile zéro test et retourne `ok` : c'est le faux vert
le plus courant de ce dépôt. La CI passe explicitement le tag.
