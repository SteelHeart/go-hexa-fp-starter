# Ports et contrats

> Décision de référence : [ADR 003](../documentation/adr/003-ports-comme-types-fonction.md).

## 1. Un port est un type fonction

```go
// ports/ports.go — ce paquet ne contient QUE des déclarations de types.
// Ni struct, ni fonction, ni interface (arch-go le vérifie).

// SaveUser persiste un utilisateur nouvellement enregistré.
//
// Contrat d'erreur — toute implémentation DOIT respecter :
//   - CodeEmailAlreadyExists  si l'unicité d'email est violée
//   - CodeUnavailable         si le stockage est injoignable
// Aucune erreur de driver ne doit remonter telle quelle.
type SaveUser = func(context.Context, domain.User) result.Result[domain.User, domain.Error]
```

Trois obligations :

1. **Un alias de type** (`=`), pas une définition. Un adaptateur peut ainsi retourner une closure
   sans conversion, et deux ports de même signature restent substituables.
2. **Un commentaire de contrat d'erreur** au-dessus de chaque port secondaire. C'est la seule forme
   que prend la substituabilité de Liskov ici, et elle est vérifiée par un **test de conformité**
   partagé entre les implémentations.
3. **`context.Context` en premier paramètre** dès qu'il y a un effet. Un port pur (`Now`,
   `GenerateID`) n'en prend pas.

Go 1.24+ autorise les **alias génériques** — utilisables si un port doit être paramétré, mais dans
le doute, un port concret est préférable : un port appartient à sa feature, pas au dépôt.

## 2. Ports primaires et secondaires

| | Primaire (cas d'usage) | Secondaire (besoin) |
|---|---|---|
| Qui l'appelle | un adaptateur primaire | le cas d'usage |
| Qui l'implémente | `application/` | un adaptateur secondaire |
| Retour | `Result[T, domain.Error]` | `Result[T, domain.Error]` |
| Exemple | `RegisterUser` | `SaveUser`, `PublishEvent` |

## 3. Value objects — la validation se fait une seule fois

Un identifiant n'est **jamais** une `string` nue dans une signature de domaine, et un montant
n'est **jamais** un `float`.

```go
// Champs non exportés : impossible de fabriquer un Email invalide.
type Email struct{ value string }

// NewEmail est le seul chemin de construction. Il normalise puis valide.
func NewEmail(raw string) result.Result[Email, Error] { … }

func (e Email) String() string { return e.value }
```

Conséquence directe : **le cœur ne valide plus rien**. Si une fonction reçoit un `Email`, il est
valide — c'est le type qui le garantit, pas une convention. Toute la validation vit dans les
constructeurs intelligents, appelés **à la frontière**.

Pour les montants : entiers en plus petite unité (centimes), jamais de flottant.

## 4. Validation aux frontières

```
requête HTTP / argument CLI / message d'événement
        │  ① l'adaptateur parse la forme (JSON, flag, payload)
        ▼
    Command (types primitifs)
        │  ② domain.ParseXxx construit les value objects → Result
        ▼
   ValidXxx (types du domaine)      ← à partir d'ici, plus aucune validation
```

- ① relève de l'**adaptateur** : forme, taille, encodage, types JSON. Avec `huma`, c'est déclaratif
  (`format`, `minLength`, `required`) et donc gratuit et documenté.
- ② relève du **domaine** : sémantique métier. Retourne `Result`, jamais de `panic`.

Une commande n'entre **jamais** dans `application/` sans être passée par ②.

## 5. Erreurs de domaine

```go
type ErrorCode string

const (
    CodeInvalidEmail       ErrorCode = "invalid_email"
    CodeWeakPassword       ErrorCode = "weak_password"
    CodeEmailAlreadyExists ErrorCode = "email_already_exists"
    CodeUnavailable        ErrorCode = "unavailable"
    CodeInternal           ErrorCode = "internal"
)
```

- `domain.Error` est une **valeur** (`Code`, `Message`, `Field`, `cause`), pas une interface.
- Le `Message` est **destiné à l'utilisateur** : il ne contient jamais de détail technique, ni de
  donnée sensible. Le détail technique va dans `cause`, journalisé mais jamais retourné.
- Le **mapping code → statut HTTP** vit dans `adapters/primary/http/`, pas dans le domaine. Un même
  code se traduit différemment selon le frontend : `409` en HTTP, `EX_DATAERR` en CLI.
- Tout `switch` sur `ErrorCode` doit être exhaustif (`exhaustive` échoue la CI sinon). C'est ce qui
  garantit qu'ajouter un code force à traiter sa traduction dans **tous** les adaptateurs.

## 6. Le contrat d'API est généré, jamais écrit

Les structs Go annotées sont la **source de vérité** ; `api/openapi.yaml` en est le produit
(`task openapi`). Écrire les deux, c'est garantir leur divergence.

- La CI échoue si `api/openapi.yaml` n'est pas à jour par rapport au code.
- Les SDK clients (TypeScript, Dart, Swift) se génèrent **depuis** ce fichier. Un frontend qui écrit
  ses types à la main réintroduit la divergence qu'on vient d'éliminer.
- Un changement cassant du contrat impose une **nouvelle version de route** (`/v2/...`), jamais une
  modification en place : plusieurs frontends déployés indépendamment consomment la v1.
