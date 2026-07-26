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
| **Domaine** | `internal/modules/{f}/domain/*_test.go` | aucune | chaque règle, chaque cas limite |
| **Cas d'usage** | `internal/modules/{f}/application/*_test.go` | closures | chaque chemin, nominal **et** d'erreur |
| **Primitives** | `internal/pkg/*/…_test.go` | aucune | lois algébriques incluses |
| **Adaptateur secondaire** | `…/secondary/{x}/*_test.go` (tag `integration`) | Postgres réel | traduction d'erreur, SQL |
| **Conformité de port** | `…/ports/conformance_test.go` (tag `integration`) | toutes les implémentations | contrat d'erreur du port |
| **Bout en bout** | `tests/e2e/` (tag `e2e`) | pile complète | un chemin critique par surface |
| **Charge** | `tests/perf/` (k6) | pile complète | avant une mise en production |

`go test ./...` **sans tag** ne doit exiger aucun service. C'est ce qui permet de travailler sans
Docker sur la machine ([`toolchain.md`](toolchain.md)).

### Boîte noire, boîte blanche, et un fichier par test

| Ce qu'on teste | Où | Paquet |
|---|---|---|
| L'API **publique** d'un paquet | `{paquet}/tests/` | `tests` |
| Les identifiants **non exportés** | `{paquet}/internal_test.go` | celui du paquet |

Tester par l'API publique est le défaut, et ce n'est pas une préférence de style : un test qui
n'atteint que ce qu'un appelant atteint interdit de figer un détail d'implémentation. Le jour où un
pilote change, ces tests ne bougent pas — c'est précisément ce qui prouve la substituabilité.

**Un test par fichier**, nommé d'après le test en `snake_case` :

```
internal/core/dynconf/tests/
├── helpers_test.go                      aides partagées, et rien d'autre
├── unknown_flag_is_denied_test.go
├── unreadable_flag_is_denied_test.go
└── settings_are_read_as_text_test.go
```

Pourquoi : le nom du fichier dit ce qui est vérifié **sans l'ouvrir**. `git log` sur un fichier
raconte l'histoire d'une seule garantie ; un conflit de fusion porte sur un seul test ; et une
garantie supprimée se voit comme une suppression de fichier, pas comme trente lignes en moins dans
un fichier de six cents.

### La même règle s'applique au CODE, pas seulement aux tests

**Un fichier long se découpe en un fichier par fonction publique**, nommé d'après elle en
`snake_case`. Le paquet reste le même — c'est le découpage physique qui change, **pas l'API** :
aucun appelant ne bouge, aucun import ne change.

```
internal/pkg/middleware/
├── middleware.go        le type Middleware, Chain, et rien d'autre
├── request_id.go        RequestID et RequestIDFrom
├── recover.go           Recover
├── security_headers.go  SecurityHeaders, SecurityHeadersWithoutHSTS
├── cors.go              CORS
├── max_body.go          MaxBody
├── rate_limiter.go      RateLimiter
└── access_log.go        AccessLog
```

Les raisons sont les mêmes que pour les tests, et une de plus : **un fichier de six cents lignes
cache ses défauts**. Le limiteur de débit de ce dépôt n'a jamais limité quoi que ce soit — l'ordre
de deux instructions supprimait le visiteur à l'instant où il était créé. Personne ne l'a vu en
relisant, parce que personne ne relit une fonction perdue au milieu d'un fichier qu'on ouvre pour
autre chose.

Le seuil n'est pas un nombre de lignes fixe : dès qu'un fichier porte **plusieurs responsabilités
publiques indépendantes**, il se découpe.

#### Quatre paquets découpés, et ce que chacun a fait apparaître

| Paquet | Découpage | Ce que le découpage a rendu visible |
|---|---|---|
| `internal/pkg/middleware` | 1 → 8 fichiers | Chaque garde traversée par toute requête est désormais nommée dans l'arborescence |
| `internal/infrastructure/security` | 1 → 4 fichiers | `hasher.go` et `cipher.go` portent chacun une **constante qui EST une garantie** (bornes du condensé, `aesKeyLen`). Noyées dans 235 lignes, elles se relâchaient sans qu'on mesure quoi |
| `internal/config` | 1 → 9 fichiers | La séparation `validation.go` / `hardening.go` empêche d'ajouter par accident une exigence de production dans un vérificateur qui tourne aussi en local. Et un `contains` maison, doublon de `slices.Contains`, est mort au passage |
| `internal/infrastructure/messaging` | 2 → 7 fichiers | `kafka.go` et `rabbitmq.go` sont **des fichiers entiers marqués NON PROUVÉS**, au lieu d'un paragraphe d'avertissement au milieu de code qui, lui, tourne |

Le point commun : dans chaque cas, le découpage n'a pas seulement rangé — il a **déplacé une
frontière au niveau du fichier**, là où elle se voit sans lire.

#### Comment nommer le fichier résiduel

Le fichier qui garde le nom du paquet ne devient pas un dépotoir : il porte le **langage** du
paquet — les types, les constantes, ce que tout le reste utilise — et la documentation de paquet,
dont une **carte des fichiers**. `middleware.go` garde `Middleware` et `Chain` ; `messaging.go`
garde l'enveloppe et les types fonction ; `security.go` ne garde que la carte, et c'est correct.

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
