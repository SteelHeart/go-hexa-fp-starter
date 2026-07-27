# SOLID et DRY, traduits en Go fonctionnel

SOLID a été formulé pour des objets. Appliqué mot à mot en Go, il produit des interfaces inutiles
et des couches vides. Voici la traduction qui fait foi dans ce dépôt.

## SOLID

### S — Responsabilité unique

**Une fonction, une raison de changer.** Outillé, pas déclaratif : `funlen` (50 lignes dans un module,
40 en `pkg`), `cyclop` (complexité 10), `arch-go` (`maxLines`, `maxPublicFunctionPerFile: 6`).

Un fichier expose **au plus 6 fonctions publiques**. Au-delà, il porte deux sujets : le découper.

### O — Ouvert / fermé

**On étend par composition, jamais en modifiant l'existant.**

- Nouveau frontend → nouveau dossier sous `adapters/primary/`. Le cœur ne bouge pas.
- Nouvelle préoccupation transverse (cache, *retry*, *circuit breaker*) → un **décorateur**
  `func(P) P`. Le cas d'usage ne bouge pas.
- Nouveau stockage → nouvelle implémentation du port secondaire. Le cas d'usage ne bouge pas.

Si ajouter un comportement oblige à ouvrir `application/`, la conception est fausse.

### L — Substitution de Liskov

Sans hiérarchie de types, le principe se réduit à sa forme utile : **deux implémentations du même
port sont interchangeables sans surprise**. Concrètement, un port secondaire déclare son contrat
d'erreur, et toutes ses implémentations le respectent.

> `SaveUser` retourne `CodeEmailAlreadyExists` sur violation d'unicité — que l'implémentation soit
> Postgres, en mémoire ou un double de test. Une implémentation qui remonterait une erreur brute de
> driver casse le contrat.

Ce contrat est **documenté dans `ports/`, au-dessus de chaque type**, et vérifié par un test de
conformité partagé entre les implémentations.

### I — Ségrégation des interfaces

**Poussé à sa limite : un port = une fonction.** C'est la plus petite interface possible ; il n'y a
rien à ségréguer.

```go
// ❌ Interface fourre-tout : un consommateur qui ne veut que Save dépend de 5 méthodes.
type UserRepository interface { Save(...); FindByID(...); FindByEmail(...); Delete(...); List(...) }

// ✅ Chaque cas d'usage ne dépend que de ce qu'il utilise réellement.
type SaveUser        = func(context.Context, domain.User) result.Result[domain.User, domain.Error]
type FindUserByEmail = func(context.Context, domain.Email) result.Result[fp.Option[domain.User], domain.Error]
```

Une interface n'est admise **que** si un algorithme a besoin de plusieurs opérations
**indissociables sur le même état** — cas rare. Elle est alors déclarée **côté consommateur**,
jamais côté implémentation, et `arch-go` interdit de la mettre dans `domain/` ou `ports/`.

### D — Inversion des dépendances

Le cœur déclare ses besoins (`ports/`), les adaptateurs les satisfont, `cmd/` les relie. La flèche
d'import est vérifiée mécaniquement par `arch-go.yml` : c'est le seul principe SOLID de ce dépôt
qui est **impossible** à violer sans faire rougir la CI.

## DRY — et sa limite

### Ce qui doit être factorisé

- Les **primitives** : `result`, `fp`, middlewares HTTP.
- Le **technique répété** : ouverture de transaction, mapping d'erreur SQL → `domain.Error`,
  décorateurs, dépilage d'outbox.
- Les **contrats générés** : le schéma OpenAPI est produit à partir des types Go, jamais écrit deux
  fois. Idem pour les SDK clients.
- `dupl` (seuil 120 jetons) et `goconst` échouent la CI sur du copier-coller.

### Ce qui doit rester dupliqué

> **Le sur-DRY est le premier tueur d'architecture hexagonale.** Une abstraction partagée entre
> deux *bounded contexts* est un couplage déguisé en économie.

| Situation | Règle |
|---|---|
| Deux modules ont un `User` qui se ressemble | **Deux types distincts.** Ils divergeront. |
| Deux modules ont besoin de la même règle | La règle est **copiée**, ou l'une publie un événement |
| Web et mobile veulent presque la même réponse | **Deux présentateurs.** Un DTO partagé fige les deux frontends ensemble |
| Deux adaptateurs secondaires font un `INSERT` similaire | Duplication acceptée : le SQL suit la table, pas le code |

Le critère de décision n'est pas la ressemblance du code, c'est la **raison de changer** : deux
morceaux qui changent pour des raisons différentes ne sont pas des duplications, ce sont des
coïncidences.

### Quand une exception est nécessaire

Une abstraction transverse qui n'entre dans aucun cas ci-dessus se tranche en **ADR**, avec le
couplage qu'elle crée écrit noir sur blanc. Pas d'ADR, pas d'abstraction.
