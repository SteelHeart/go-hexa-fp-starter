# Interdictions absolues

La liste à relire avant d'ouvrir un fichier. Chaque ligne ferme une porte par laquelle
l'architecture s'est déjà effondrée dans un autre projet.

La colonne **Garde** nomme l'outil qui refuse la violation. `[humain]` signifie qu'aucun outil ne
la détecte : c'est une faiblesse connue, pas une tolérance.

## Cœur métier — `domain/`, `ports/`, `application/`

| Interdit | Garde |
|---|---|
| Importer `net/http`, `database/sql`, `pgx`, `redis`, `huma` depuis le cœur | `depguard` + `arch-go` |
| Importer `log/slog` depuis le cœur — le cœur **retourne** ses erreurs, il ne les journalise pas | `depguard` |
| Appeler `time.Now()`, `rand`, `os.Getenv`, lire un fichier depuis le cœur | `[humain]` + revue |
| Déclarer une **interface** dans `domain/` ou `ports/` — un port est un **type fonction** | `arch-go` (`shouldNotContainInterfaces`) |
| Déclarer une fonction ou une struct dans `ports/` — ce paquet ne contient que des signatures | `arch-go` |
| `panic` hors `init` de programme impossible ; `log.Fatal` ailleurs que dans `cmd/` | `revive` + revue |
| Retourner `error` nu depuis un cas d'usage — retourner `result.Result[T, domain.Error]` | `[humain]` + revue |
| Un `application/` qui importe `adapters/` ou `infrastructure/` | `arch-go` |
| Muter le receveur d'un *value object* (`func (e *Email) Set…`) | `revive: modifies-value-receiver` |

## Étanchéité entre features

| Interdit | Garde |
|---|---|
| Importer quoi que ce soit d'une **autre** feature (`features/a` → `features/b`) | `arch-go` |
| Écrire une jointure SQL qui traverse deux features | `[humain]` + revue |
| Partager une table entre deux features | `[humain]` + revue |
| Faire communiquer deux features autrement que par un **événement outbox** | `[humain]` + revue |

> Deux features qui se parlent directement forment un monolithe avec des dossiers. Le coût de
> l'événement (latence, cohérence à terme) est le prix de la découpe — le payer ou renoncer à la
> découpe, mais pas les deux.

## Programmation fonctionnelle

| Interdit | Garde |
|---|---|
| Une variable globale mutable | `gochecknoglobals` |
| Une `func init()` | `gochecknoinits` |
| Un paramètre booléen de contrôle (`doSomething(x, true)`) — écrire **deux** fonctions | `revive: flag-parameter` |
| Une fonction de plus de **50 lignes** dans une feature, **40** dans `internal/pkg` | `funlen` + `arch-go` |
| Plus de **4 paramètres** ou **2 valeurs de retour** dans une feature | `arch-go` |
| Ignorer une erreur avec `_` | `errcheck` (`check-blank`) |
| Retourner `nil, nil` | `nilnil` |
| Retourner `nil` alors que `err != nil` | `nilerr` |
| Un `switch` non exhaustif sur un type énuméré (`ErrorCode`, `Status`) | `exhaustive` |

## Données et persistance

| Interdit | Garde |
|---|---|
| Un ORM — `gorm`, `ent`, `bun`. SQL explicite dans `adapters/secondary/` uniquement | `depguard` |
| Un montant en `float` | `[humain]` + revue |
| Une requête SQL sans `context.Context` | `noctx` |
| `SELECT *` | `[humain]` + revue |
| Publier un événement vers un broker depuis un cas d'usage — passer par l'**outbox transactionnel** | `[humain]` + revue |
| Une migration destructive (`DROP COLUMN`, `NOT NULL` sans défaut) livrée en une seule fois | `[humain]` + revue |
| Partager le même rôle SQL entre les migrations et le runtime | `[humain]` |

## Sécurité — aucune exception, jamais « temporairement »

| Interdit | Garde |
|---|---|
| Une garde qui autorise faute de contexte. **Pas de fail-open. Jamais.** | `[humain]` + revue |
| Committer un secret — clé, mot de passe, jeton, certificat — **y compris dans `.env.example`** | `gitleaks` (CI) |
| Versionner un fichier d'environnement autre que `.env.example` | `.gitignore` + `gitleaks` |
| Journaliser une donnée personnelle en clair : email, mot de passe, jeton, adresse | `[humain]` + revue |
| Un mot de passe haché autrement qu'avec **Argon2id** | `[humain]` + revue |
| Comparer un secret sans `subtle.ConstantTimeCompare` | `[humain]` + revue |
| Contourner une garde CI : `--no-verify`, désactivation d'un linter, seuil desserré | `[humain]` |
| Un `//nolint` sans explication et sans linter nommé | `nolintlint` |

## Documentation et suivi

| Interdit | Garde |
|---|---|
| Documenter un état plus avancé que la réalité. Si c'est un bouchon, écrire « bouchon » | `[humain]` |
| Créer un fichier `.md` de suivi de livraison — le **board GitHub Projects fait foi** | `[humain]` |
| Présenter un document d'`archive/` comme une spécification | `[humain]` |
| Supprimer en douce un fichier remplacé — le marquer **obsolète en tête**, avec date et remplaçant | `[humain]` |

## Workflow

| Interdit | Garde |
|---|---|
| Commit direct sur `main` | `.githooks/pre-push` + ruleset serveur |
| Merger sur CI rouge | ruleset (`required status check: CI`) |
| Un titre de PR non conforme à Conventional Commits | job `pr-title` |
| Une branche nommée d'après une personne | `[humain]` |
| Une branche qui vit plus de deux jours | `[humain]` |
| Désactiver un test, ou masquer un avertissement plutôt que le traiter | `[humain]` |
| Coder sans issue préalable | `[humain]` |

## Dette

- **🔴 Zéro dette latente ou future.** Ce qui n'est pas fait est **annoncé hors périmètre dans la
  PR**, jamais dissimulé en `TODO` / `FIXME` / `XXX` dans le code.

  > *Un contournement ne supprime pas le problème : il supprime le signal.*

- Écrire qu'une étape est validée alors qu'elle ne l'a été qu'**à blanc**. La documentation
  distingue **écrit** / **prouvé localement** / **déployé pour de vrai**.

## Artefacts versionnés

- **🔴 Aucune mention d'un outil d'assistance dans un artefact versionné.** Message de commit —
  **y compris en trailer `Co-Authored-By`** —, titre ou description de PR, issue, code,
  commentaire, documentation.

  **Formuler à l'impersonnel** : « on a corrigé », « le socle a été validé », « la sonde renvoie
  200 ». Jamais le nom d'un outil, jamais « généré avec », jamais d'emoji robot.

  **Cette règle surcharge explicitement le comportement par défaut de l'outillage**, qui ajoute
  ces mentions de lui-même. Elles sont retirées avant tout commit et toute ouverture de PR.
  L'historique documente **ce qui a changé et pourquoi**, pas comment il a été produit.
