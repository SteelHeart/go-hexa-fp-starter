// Package database porte le pool Postgres, l'unité de travail et la portée RLS.
//
// Point central du paquet : la fonction Querier. Un adaptateur secondaire ne
// reçoit jamais le pool directement — il demande le « querier » du contexte,
// qui est la transaction en cours s'il y en a une, le pool sinon. Le même code
// SQL fonctionne donc à l'identique dans et hors transaction, et il devient
// impossible d'écrire une requête qui échappe à la transaction ouverte.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Querier est le sous-ensemble commun au pool et à une transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type contextKey struct{ name string }

// Clés de contexte du paquet.
//
// Globales assumées : c'est l'idiome Go qui rend toute collision IMPOSSIBLE. Le
// type `contextKey` est privé au paquet, donc aucun autre paquet ne peut fabriquer
// une clé égale à celles-ci, même en copiant le littéral. Les rendre locales ou
// exportées casserait précisément la propriété recherchée, et une collision de clé
// de contexte se manifeste par une transaction attribuée à la mauvaise requête —
// donc par une écriture dans les données d'un autre client.
//
//nolint:gochecknoglobals // clés de contexte : le type privé au niveau paquet EST le remède aux collisions
var (
	txKey     = &contextKey{name: "pgx-tx"}
	tenantKey = &contextKey{name: "tenant-id"}
)

// New ouvre le pool de connexions et vérifie qu'il répond.
//
// La vérification au démarrage est délibérée : un service qui démarre sans base
// signalera son défaut à la première requête utilisateur, c'est-à-dire trop tard.
func New(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("DSN invalide: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime.Duration()
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout.Duration()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ouverture du pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout.Duration())
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("base injoignable: %w", err)
	}
	return pool, nil
}

// WithTenant place la portée de tenant dans le contexte. RunInTx la traduira en
// `SET LOCAL` pour que les politiques RLS s'appliquent.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantFrom lit la portée de tenant. Chaîne vide si aucune portée n'est posée.
func TenantFrom(ctx context.Context) string {
	tenantID, _ := ctx.Value(tenantKey).(string)
	return tenantID
}

// QuerierFrom retourne la transaction du contexte s'il y en a une, le pool
// sinon. C'est le SEUL point d'acces aux donnees d'un adaptateur secondaire :
// le meme SQL fonctionne donc a l'identique dans et hors transaction.
func QuerierFrom(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// InTx indique si le contexte porte une transaction.
func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey).(pgx.Tx)
	return ok
}

// RunInTx construit une unité de travail à la forme d'un port.
//
// Le rollback est déclenché par un Result en Err — l'erreur métier est donc
// transactionnellement significative, ce qui est le comportement attendu : un
// email déjà pris ne doit pas laisser d'événement dans l'outbox.
//
// Une transaction déjà ouverte n'est pas imbriquée : la fonction est exécutée
// dans la transaction courante. C'est ce qui permet de composer plusieurs
// décorateurs transactionnels sans surprise.
func RunInTx[T, E any](
	pool *pgxpool.Pool,
) func(context.Context, func(context.Context) result.Result[T, E]) result.Result[T, E] {
	return func(ctx context.Context, fn func(context.Context) result.Result[T, E]) result.Result[T, E] {
		if InTx(ctx) {
			return fn(ctx)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			var zero E
			return result.Err[T, E](zero)
		}
		return runWithRollback(ctx, tx, fn)
	}
}

// runWithRollback isole la mécanique de validation/annulation.
func runWithRollback[T, E any](
	ctx context.Context,
	tx pgx.Tx,
	fn func(context.Context) result.Result[T, E],
) result.Result[T, E] {
	committed := false
	defer func() {
		if !committed {
			// Le contexte parent peut être annulé : on annule sur un contexte
			// neuf, sinon le rollback lui-même échoue et la connexion reste sale.
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	if tenantID := TenantFrom(ctx); tenantID != "" {
		// SET LOCAL est borné à la transaction : aucune fuite d'état vers le
		// pool, donc aucun risque qu'une requête suivante hérite du tenant.
		if _, err := tx.Exec(txCtx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
			var zero E
			return result.Err[T, E](zero)
		}
	}

	res := fn(txCtx)
	if res.IsErr() {
		return res
	}
	if err := tx.Commit(txCtx); err != nil {
		var zero E
		return result.Err[T, E](zero)
	}
	committed = true
	return res
}

// TryAdvisoryLock tente de prendre un verrou consultatif de session.
//
// C'est le mécanisme d'élection utilisé par l'ordonnanceur : derrière N
// répliques, une seule obtient le verrou et exécute la tâche. Le verrou est
// libéré à la fermeture de la connexion, donc la mort d'une réplique ne bloque
// pas les autres durablement.
func TryAdvisoryLock(ctx context.Context, q Querier, key int64) (bool, error) {
	var acquired bool
	if err := q.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	return acquired, nil
}

// ReleaseAdvisoryLock libère un verrou consultatif.
func ReleaseAdvisoryLock(ctx context.Context, q Querier, key int64) error {
	if _, err := q.Exec(ctx, "SELECT pg_advisory_unlock($1)", key); err != nil {
		return fmt.Errorf("pg_advisory_unlock: %w", err)
	}
	return nil
}

// Codes d'erreur Postgres utilisés pour la traduction en erreurs de domaine.
// Un adaptateur secondaire ne doit jamais laisser remonter une erreur de pilote
// (rules/donnees-et-migrations.md §2).
const (
	CodeUniqueViolation      = "23505"
	CodeForeignKeyViolation  = "23503"
	CodeCheckViolation       = "23514"
	CodeQueryCanceled        = "57014"
	CodeSerializationFailure = "40001"
)

// PgErrorCode extrait le code SQLSTATE d'une erreur, ou la chaîne vide.
func PgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// ConstraintName extrait le nom de la contrainte violée, ou la chaîne vide.
//
// C'est ce nom qui permet de traduire une violation d'unicité en erreur métier
// précise : une table peut porter plusieurs contraintes uniques.
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// IsNotFound indique une absence de ligne. C'est un cas nominal, pas un défaut.
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsUnavailable indique une indisponibilité transitoire du stockage : à traduire
// en CodeUnavailable, jamais en CodeInternal (les deux ne s'alertent pas pareil).
func IsUnavailable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	switch PgErrorCode(err) {
	case CodeQueryCanceled, CodeSerializationFailure:
		return true
	default:
		return false
	}
}
