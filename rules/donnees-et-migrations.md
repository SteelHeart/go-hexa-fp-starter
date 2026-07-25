# Données, persistance et migrations

> Décision de référence : [ADR 006](../documentation/adr/006-outbox-transactionnel.md).

## 1. SQL explicite, zéro ORM

`gorm`, `ent` et `bun` sont **interdits** (`depguard` échoue la CI). Motif : leurs modèles, tags et
crochets fuient dans le domaine et transforment un *value object* en ligne de table.

- `pgx/v5` en accès direct, requêtes écrites à la main, dans `adapters/secondary/postgres/`
  **uniquement**.
- Pas de `SELECT *` : les colonnes sont nommées, sinon un `ALTER TABLE` casse le scan silencieusement.
- Toute requête prend un `context.Context` (`noctx` échoue la CI sinon).
- Les paramètres sont **toujours** liés (`$1`, `$2`). Aucune concaténation de chaîne dans du SQL.

## 2. Le cœur ne voit jamais une erreur de driver

L'adaptateur secondaire traduit, et c'est sa responsabilité principale :

| Cause Postgres | `domain.ErrorCode` |
|---|---|
| `23505` violation d'unicité | `CodeEmailAlreadyExists` (selon la contrainte) |
| `23503` violation de clé étrangère | `CodeNotFound` ou code métier |
| `context.DeadlineExceeded`, `57014` | `CodeUnavailable` |
| tout le reste | `CodeInternal`, avec la cause en `cause` (journalisée, jamais retournée) |

Une erreur non traduite qui remonte au cas d'usage est un défaut, pas un cas limite.

## 3. Transactions

L'unité de travail est un **décorateur**, pas un appel manuel :

```go
register = application.Apply(register, application.WithTransaction(runInTx))
```

- `runInTx` place la transaction dans le `context`. Les adaptateurs secondaires lisent le
  *querier* depuis le contexte (`database.Querier(ctx, pool)`) : ils fonctionnent à l'identique
  dans et hors transaction.
- Le rollback est déclenché par un `Result` en `Err`, ou par un `panic` (re-propagé après rollback).
- Un cas d'usage **n'ouvre jamais** de transaction lui-même : il ne sait pas qu'il en existe.

## 4. Outbox transactionnel — la seule sortie vers le monde

Publier un événement depuis un cas d'usage, c'est accepter qu'il soit publié alors que la
transaction a échoué, ou perdu alors qu'elle a réussi. Les deux sont inacceptables.

```
transaction : INSERT users …  +  INSERT outbox_messages …   ← atomique
   worker    : SELECT … FOR UPDATE SKIP LOCKED → publie → marque traité
```

Règles :

- Un cas d'usage écrit dans `outbox_messages` via le port `PublishEvent`. Il ne connaît aucun broker.
- Le dépilage est **au moins une fois** : tout consommateur est donc **idempotent**, et cette
  idempotence est testée.
- Un message en échec est réessayé avec recul exponentiel jusqu'à `WORKER_MAX_ATTEMPTS`, puis
  marqué `failed` — jamais supprimé.
- `FOR UPDATE SKIP LOCKED` : plusieurs workers tournent sans coordination.

## 5. Migrations — rétro-compatibilité obligatoire

Le déploiement applique les migrations **avant** le nouveau code, et un rollback de code ne défait
pas une migration. Donc :

> **Toute migration doit fonctionner avec la version N-1 du code.**

| Changement | Interdit en une fois | Procédure |
|---|---|---|
| Renommer une colonne | ✅ interdit | ajouter · écrire dans les deux · migrer · lire la nouvelle · supprimer |
| Supprimer une colonne | ✅ interdit | cesser d'écrire · déployer · supprimer au cycle suivant |
| Ajouter `NOT NULL` | ✅ interdit | ajouter avec défaut · remplir · contraindre |
| Ajouter un index sur une grosse table | ✅ interdit | `CREATE INDEX CONCURRENTLY`, hors transaction |
| Ajouter une colonne nullable | autorisé | — |

Chaque migration a un `-- +goose Down` **réellement testé**. Un `Down` faux est pire que pas de
`Down` : il donne une fausse assurance en incident.

Nommage : `{YYYYMMDDHHMMSS}_{slug_snake}.sql`, une migration par PR, jamais modifiée après merge.

## 6. Le rôle SQL des migrations n'est pas celui du runtime

Le rôle applicatif **ne possède pas** le schéma : il n'a ni `CREATE`, ni `ALTER`, ni `DROP`. C'est
ce qui empêche une injection SQL réussie de modifier la structure ou de désactiver une politique.

Deux DSN distincts : `DB_DSN` (runtime) et `DB_MIGRATION_DSN` (migrations). En développement local
ils peuvent coïncider ; en UAT et en production, jamais.

## 7. Types

- **Argent** : entier dans la plus petite unité. `float` interdit, sans exception.
- **Temps** : `timestamptz`, toujours en UTC. Jamais de `timestamp` nu.
- **Identifiants** : UUID v7 (ordonnés dans le temps → index performants), stockés en `uuid`.
- **Énumérations** : `text` + contrainte `CHECK`, pas de type `ENUM` Postgres (dont l'`ALTER` est
  pénible et non transactionnel selon les versions).
- **Suppression** : logique (`deleted_at`) quand une trace est requise, physique sinon. Le choix est
  documenté dans la migration.
