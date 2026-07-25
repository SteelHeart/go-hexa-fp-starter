# Dépendances externes

Chaque dépendance est une dette : mises à jour, vulnérabilités, changements cassants, et un jour
un abandon de maintenance. Le socle en assume peu, délibérément.

## 1. Le test avant d'ajouter quoi que ce soit

Dans l'ordre, et on s'arrête à la première réponse positive :

1. **La bibliothèque standard le fait-elle ?** `net/http`, `crypto/*`, `log/slog`, `encoding/json`,
   `database/sql`, `testing` couvrent l'essentiel. Une dépendance qui remplace 40 lignes de stdlib
   est un mauvais échange.
2. **Peut-on l'écrire en moins de 100 lignes qu'on comprend entièrement ?** Alors on l'écrit.
3. **Est-elle confinable dans `infrastructure/` ou un adaptateur ?** Si elle doit apparaître dans
   `domain/` ou `ports/`, la réponse est **non**, définitivement.
4. **Survivra-t-elle cinq ans ?** Gouvernance, rythme de publication, nombre de mainteneurs,
   compatibilité avec `http.Handler` et `context.Context`.

Si les quatre passent : ADR, avec les alternatives écartées et le coût de sortie.

## 2. Liste blanche

Toute dépendance hors de cette liste exige un ADR. La liste elle-même est en zone à haute inertie
(garde CI `inertia`).

| Module | Rôle | Confiné dans |
|---|---|---|
| `github.com/go-chi/chi/v5` | routage HTTP, 100 % `http.Handler` | `infrastructure/http_server/` |
| `github.com/danielgtaylor/huma/v2` | validation + OpenAPI généré | `infrastructure/http_server/`, `adapters/primary/http/` |
| `github.com/jackc/pgx/v5` | pilote Postgres | `infrastructure/database/`, `adapters/secondary/postgres/` |
| `github.com/redis/go-redis/v9` | cache | `infrastructure/cache/` |
| `github.com/google/uuid` | UUID v7 | `infrastructure/`, adaptateurs |
| `github.com/caarlos0/env/v11` | lecture d'environnement | `config/` |
| `golang.org/x/crypto` | Argon2id | `infrastructure/security/` |
| `golang.org/x/time` | limitation de débit | `internal/pkg/middleware/` |
| `go.opentelemetry.io/*` | traces et métriques | `infrastructure/telemetry/` |
| `github.com/prometheus/client_golang` | exposition Prometheus | `infrastructure/telemetry/` |

## 3. Interdits par nom

| Interdit | Motif | Garde |
|---|---|---|
| `gorm.io/gorm`, `entgo.io/ent`, `github.com/uptrace/bun` | ORM : fuit dans le domaine | `depguard` |
| Toute bibliothèque de *mock* générée | un type fonction se substitue en une ligne | revue |
| Tout conteneur d'injection de dépendances (`fx`, `dig`, `wire`) | la composition manuelle est vérifiée par le compilateur ([ADR 004](../documentation/adr/004-composition-manuelle-sans-conteneur-di.md)) | revue |
| `github.com/gofiber/fiber` | hors `net/http` : exclut tout l'écosystème `http.Handler` | revue |
| `github.com/pkg/errors` | `errors.Join` et `%w` couvrent le besoin depuis Go 1.20 | revue |
| `github.com/sirupsen/logrus`, `go.uber.org/zap` | `log/slog` est dans la stdlib | revue |
| `github.com/spf13/viper` | lit 6 formats dont on n'utilise aucun ; l'environnement suffit | revue |

## 4. Discipline de mise à jour

- `dependabot` hebdomadaire, modules OpenTelemetry et `golang.org/x/*` **groupés** : des versions
  désynchronisées cassent la compilation.
- Une montée de version majeure se traite dans **sa propre PR**, jamais mélangée à du métier.
- `govulncheck` est bloquant. Une vulnérabilité sans correctif disponible devient une entrée `S*`
  du [registre de sécurité](../documentation/securite/registre-securite.md), avec sa mitigation.
- `go.mod` déclare la version de Go **réellement testée en CI** — la CI lit `go-version-file: go.mod`,
  donc les deux ne peuvent pas diverger.

## 5. Outils (hors compilation)

`golangci-lint`, `arch-go`, `goose`, `govulncheck`, `gitleaks`, `k6` ne sont pas des dépendances du
module : ils sont installés par `task tools` et pinnés en CI.

Ils sont en `latest` tant que le socle n'a pas publié sa première version, puis **figés** — un
linter qui change de comportement tout seul rend la CI non reproductible.
