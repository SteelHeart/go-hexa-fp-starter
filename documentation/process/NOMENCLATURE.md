# Nomenclature — issues, branches, commits, fichiers, tests

## Issue (obligatoire avant de coder)

**Titre**

```
[{scope}] {verbe} — {objet court}
```

| Scope | Usage |
|---|---|
| `{feature}` | Une feature métier précise (`user_registration`) |
| `core` | `internal/pkg/` — primitives transverses |
| `infra` | `internal/infrastructure/`, `config/` |
| `http` · `cli` · `events` | Une surface précise |
| `data` | Migrations, modèle de données, outbox |
| `ci` | Workflows, gardes, outillage |
| `docs` | Documentation, ADR, règlement |

Exemples : `[user_registration] Ajouter — vérification d'unicité d'email` ·
`[data] Corriger — index manquant sur outbox_messages`

**Corps minimal** : le besoin, les critères d'acceptation en cases à cocher, les surfaces
impactées, le contrat API touché (endpoint + schéma) ou « aucun ».

**Labels** : voir [`LABELS.md`](LABELS.md). Exactement un `type:`, au moins un `area:`.

---

## Branche

```
{type}/{issue}-{slug-kebab}
```

| Type | Usage |
|---|---|
| `feat` | Nouvelle fonctionnalité |
| `fix` | Correction |
| `sec` | Correction de sécurité (entrée `S*` du registre) |
| `refactor` | Restructuration sans changement de comportement |
| `test` | Tests uniquement |
| `docs` | Documentation, ADR, règlement |
| `perf` | Performance |
| `ci` | Workflows, gardes |
| `chore` | Outillage, dépendances |

Exemple : `feat/13-outbox-transactionnel`

Pas de préfixe projet : `#13` s'auto-lie dans GitHub. Durée de vie visée : **moins de deux jours**
([ADR 007](../adr/007-tronc-unique-et-environnements.md)).

---

## Commit

```
{type}({scope}): {description} (#{issue})
```

Exemple : `feat(user_registration): refuser un email déjà enregistré (#13)`

Impératif présent, minuscule, sans point final.

> 🔴 **Aucune mention d'outillage d'assistance** — pas de `Co-Authored-By`, pas de « Generated
> with », pas d'emoji robot. Voir [`rules/workflow-git.md`](../../rules/workflow-git.md) §3.
> Gardes : crochet `commit-msg`, job CI `inertia`.

---

## Fichiers

| Type | Chemin |
|---|---|
| Décision d'architecture | `documentation/adr/{NNN}-{slug-kebab}.md` |
| Doc technique | `documentation/technique/{slug-kebab}.md` |
| Faille | Entrée `S{NNN}` dans `documentation/securite/registre-securite.md` |
| Trou du socle | Entrée `F{NNN}` dans `documentation/process/JOURNAL_FRICTION.md` |
| Migration | `migrations/{YYYYMMDDHHMMSS}_{slug_snake}.sql` |

**Convention** : `kebab-case` pour les fichiers de documentation, `snake_case` pour les paquets Go
et les migrations, sauf identifiants (`ADR`, `S{NNN}`, `F{NNN}`) et fichiers racine conventionnels
(`README.md`, `documentation/AMORCAGE.md`, `SECURITY.md`).

**Paquets Go** : un nom court, en minuscules, sans underscore ni pluriel — sauf les dossiers de
features, en `snake_case`, qui reprennent le vocabulaire métier.

---

## Tests

| Niveau | Emplacement | Tag |
|---|---|---|
| Domaine (pur) | `internal/modules/{f}/domain/{sujet}_test.go` | aucun |
| Cas d'usage | `internal/modules/{f}/application/{usecase}_test.go` | aucun |
| Primitives | `internal/pkg/{paquet}/{sujet}_test.go` | aucun |
| Adaptateur secondaire | `internal/modules/{f}/adapters/secondary/{x}/{sujet}_test.go` | `integration` |
| Conformité de port | `internal/modules/{f}/ports/conformance_test.go` | `integration` |
| Bout en bout | `tests/e2e/{surface}_{parcours}_test.go` | `e2e` |
| Charge | `tests/perf/{parcours}.js` | — |

**Un test de règle métier porte le nom de la règle** : la traçabilité règle ↔ test doit se lire
sans ouvrir les fichiers. Les sous-tests sont nommés en langage métier
(`"refuse un email sans domaine"`), pas en jargon technique.

> ⚠️ `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et affiche `ok`. C'est le
> faux vert le plus courant du dépôt.
