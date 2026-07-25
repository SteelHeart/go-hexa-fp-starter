# ADR 002 — `Result[T, E]` et limites réelles du typage Go

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Le style `(T, error)` de Go a deux défauts pour un cœur métier :

- **Rien n'oblige à traiter l'erreur.** `v, _ := f()` compile. Et `return nil, nil` aussi.
- **`error` est une interface**, donc le cœur dépend d'un contrat ouvert : n'importe quel paquet
  peut y glisser une erreur de driver, et le site d'appel ne sait pas ce qu'il peut recevoir.

L'idée de porter `Result` en Go est courante ; sa mise en œuvre naïve échoue toujours au même
endroit, et il faut l'écrire noir sur blanc avant que quelqu'un ne le redécouvre.

## Décision

Le cœur retourne `result.Result[T, E]`, avec `E = domain.Error` — une **valeur** typée, pas une
interface.

Et, point structurant : **Go n'autorise pas les paramètres de type sur les méthodes.**

```go
func (r Result[T, E]) Map[U any](f func(T) U) Result[U, E]   // ILLÉGAL
func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E]  // LÉGAL
```

Conséquences imposées par le langage, pas par un choix de style :

1. Toute transformation qui change le type est une **fonction libre** du paquet `result`.
2. **Pas de chaînage fluide.** Le style cible ressemble à du Rust sans `?`, pas à du Haskell.
3. Les seules méthodes admises sont celles qui n'introduisent pas de type : `IsOk`, `IsErr`,
   `Get`, `ValueOr`.
4. Pour éviter la pyramide de `FlatMap` imbriqués, un cas d'usage s'écrit comme une **suite
   d'étapes de même type** (`func(state) Result[state, Error]`), composée par `result.Chain`.

`Result` n'est **jamais** `nil` : sa valeur zéro est un `Err`. Un `Result` oublié échoue — *deny
par défaut* jusque dans le typage.

## Conséquences

### Ce que ça achète

- Impossible d'ignorer une erreur : `Get()` retourne un booléen que le compilateur oblige à lire.
- L'ensemble des erreurs possibles est **fermé et énumérable** (`ErrorCode`), donc les `switch` sont
  vérifiés exhaustifs, donc ajouter un cas d'erreur force à traiter sa traduction dans **toutes**
  les surfaces.
- Le cœur ne voit jamais une erreur de driver.

### Ce que ça coûte

- Une conversion `error` ↔ `domain.Error` à chaque frontière.
- Un style verbeux : `result.Map(r, f)` se lit moins bien que `r.map(f)`. C'est le prix du langage.
- Le patron `state` + `Chain` doit être appris ; il n'est pas idiomatique Go.

### Ce que ça rend impossible

- `v, _ := useCase(...)`
- Retourner `nil, nil`
- Remonter une erreur `pgx` à un adaptateur primaire

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| `(T, error)` idiomatique | N'oblige à rien ; l'ensemble des erreurs reste ouvert |
| Erreurs sentinelles + `errors.Is` | Meilleur, mais l'exhaustivité reste non vérifiable |
| Une bibliothèque `Result` tierce | Même limite de langage, plus une dépendance dans le cœur — interdit |
| Panique/récupération pour le flux d'erreur | Rend le flux invisible et les tests fragiles |

## Garde

`nilnil`, `nilerr`, `errcheck` (`check-blank`), `exhaustive` dans `.golangci.yml` ; `depguard`
interdit les paquets d'infrastructure dans le cœur ; revue pour le retour de type des cas d'usage.
