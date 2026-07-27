# Programmation fonctionnelle en Go

> Décision de référence : [ADR 002](../documentation/adr/002-result-et-limites-du-typage-go.md).

## 1. Ce que Go permet — et ce qu'il ne permet pas

**À lire avant d'écrire la première monade.** Go n'a ni types de rang supérieur, ni
**paramètres de type sur les méthodes**. Cette seconde limite est structurante :

```go
// ILLÉGAL — Go n'accepte pas de paramètre de type sur une méthode.
func (r Result[T, E]) Map[U any](f func(T) U) Result[U, E]

// LÉGAL — la transformation est une fonction libre.
func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E]
```

Conséquences, non négociables parce qu'imposées par le langage :

- **Toute transformation qui change le type est une fonction libre** du paquet `result` ou `fp`.
- **Pas de chaînage fluide** `r.Map(f).FlatMap(g)`. Le style cible ressemble à du Rust sans `?`,
  pas à du Haskell.
- Les seules **méthodes** admises sur `Result` sont celles qui n'introduisent pas de type :
  `IsOk`, `IsErr`, `Get`, `ValueOr`.
- Pour éviter la pyramide de `FlatMap` imbriqués, un cas d'usage s'écrit comme une **suite d'étapes
  de même type** (`func(state) Result[state, Error]`) composée par `result.Chain`. C'est le patron
  imposé — voir `internal/modules/user_registration/application/register_user.go`.

Prétendre l'inverse produit du code illisible qu'on finit par abandonner. Le cadre assume la
limite plutôt que de la combattre.

## 2. Pureté

Une fonction du cœur est **pure** : mêmes entrées → mêmes sorties, aucun effet observable.

Sont donc **interdits dans `domain/`, `ports/`, `application/`** :

| Effet | À la place |
|---|---|
| `time.Now()` | un port `ports.Now = func() time.Time`, injecté |
| `uuid.New()` | un port `ports.GenerateID = func() domain.UserID` |
| `rand`, `os.Getenv`, lecture de fichier | injecté depuis `cmd/` |
| `slog.Info(...)` | le cœur **retourne** son résultat ; le décorateur `WithLogging` journalise |
| une écriture en base | un port secondaire |

Le test le plus simple : **si un test du cœur a besoin de Docker, la règle est violée.**

## 3. Immuabilité

- Les *value objects* sont des `struct` à champs **non exportés**, construits par un constructeur
  intelligent qui retourne un `Result`. Une fois construits, ils sont valides pour toujours.
- **Récepteurs par valeur** partout (`func (e Email) String() string`). Un récepteur pointeur sur
  un *value object* est refusé par `revive: modifies-value-receiver`.
- Une « modification » retourne une **nouvelle** valeur : `func (u User) WithStatus(s Status) User`.
- Pas de `slice`/`map` exportée dans une struct de domaine sans copie défensive à la construction.

## 4. `Result[T, E]` — l'erreur est une valeur

```go
// Signature canonique d'un cas d'usage.
type RegisterUser = func(context.Context, domain.RegistrationCommand) result.Result[domain.User, domain.Error]
```

Règles :

1. **Un cas d'usage ne retourne jamais `error`.** Il retourne `Result[T, domain.Error]`.
   `domain.Error` est une **valeur** typée (`Code`, `Message`, `Field`), pas une interface.
2. `Result` n'est **jamais** `nil` : sa valeur zéro est un `Err` avec `E` à zéro. Un `Result`
   oublié échoue, il ne réussit pas silencieusement — *deny par défaut* jusque dans le typage.
3. La conversion `error` ↔ `domain.Error` se fait **aux frontières** : dans les adaptateurs
   secondaires (entrée) et primaires (sortie). Le cœur ne voit que `domain.Error`.
4. `Get() (T, E, bool)` est le seul moyen de sortir de la boîte, et le booléen force le site
   d'appel à traiter les deux branches.

`error` nu reste normal dans `infrastructure/` et `cmd/` : c'est du code technique, pas du métier.

## 5. Effets : injectés, jamais subis

Un cas d'usage reçoit ses effets sous forme de **fonctions**, dans une struct `Deps` :

```go
type Deps struct {
    EmailIsTaken ports.EmailIsTaken
    HashPassword ports.HashPassword
    SaveUser     ports.SaveUser
    PublishEvent ports.PublishEvent
    GenerateID   ports.GenerateID
    Now          ports.Now
}
```

Le double d'essai d'un test est une **closure**, jamais un mock généré :

```go
deps.SaveUser = func(_ context.Context, u domain.User) result.Result[domain.User, domain.Error] {
    return result.Ok[domain.User, domain.Error](u)
}
```

Aucune bibliothèque de mock n'est autorisée ([`dependances.md`](dependances.md)) : un type fonction
se substitue en une ligne, et une bibliothèque de mock ne sert qu'à contourner des interfaces
trop larges.

## 6. Décorateurs plutôt qu'héritage

Toute préoccupation transverse est un `func(P) P` :

```go
type Decorator = func(ports.RegisterUser) ports.RegisterUser

register = application.Apply(register,
    application.WithTransaction(runInTx),  // le plus externe
    application.WithTracing(tracer),
    application.WithLogging(logger),
)                                          // le plus interne appelle le cas d'usage
```

**L'ordre est significatif et se lit de l'extérieur vers l'intérieur.** La transaction enveloppe la
trace, sinon un rollback n'apparaît pas dans le span. Toute modification de cet ordre se justifie
en revue.

## 7. Ce qui reste impératif — et c'est très bien

Go n'est pas un langage fonctionnel. Sont **normaux et attendus** :

- une boucle `for` dans une fonction de 5 lignes ;
- `if err != nil` dans `infrastructure/` et les adaptateurs ;
- `context.Context` en premier paramètre, partout où il y a un effet.

Le dogmatisme coûte plus cher qu'il ne rapporte. La ligne est claire : **le cœur est pur, les bords
sont impératifs**, et la frontière est vérifiée par `arch-go`.
