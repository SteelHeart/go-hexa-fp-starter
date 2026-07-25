// Package database porte le pool Postgres, l'unitÃ© de travail et la portÃ©e RLS.
//
// Point central du paquet : la fonction Querier. Un adaptateur secondaire ne
// reÃ§oit jamais le pool directement â€” il demande le Â« querier Â» du contexte,
// qui est la transaction en cours s'il y en a une, le pool sinon. Le mÃªme code
// SQL fonctionne donc Ã  l'identique dans et hors transaction, et il devient
// impossible d'Ã©crire une requÃªte qui Ã©chappe Ã  la transaction ouverte.
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

// Querier est le sous-ensemble commun au pool et Ã  une transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type contextKey struct{ name string }

var (
	txKey     = &contextKey{name: "pgx-tx"}
	tenantKey = &contextKey{name: "tenant-id"}
)

// New ouvre le pool de connexions et vÃ©rifie qu'il rÃ©pond.
//
// La vÃ©rification au dÃ©marrage est dÃ©libÃ©rÃ©e : un service qui dÃ©marre sans base
// signalera son dÃ©faut Ã  la premiÃ¨re requÃªte utilisateur, c'est-Ã -dire trop tard.
func New(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("DSN invalide: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ouverture du pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("base injoignable: %w", err)
	}
	return pool, nil
}

// WithTenant place la portÃ©e de tenant dans le contexte. RunInTx la traduira en
// `SET LOCAL` pour que les politiques RLS s'appliquent.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantFrom lit la portÃ©e de tenant. ChaÃ®ne vide si aucune portÃ©e n'est posÃ©e.
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

// RunInTx construit une unitÃ© de travail Ã  la forme d'un port.
//
// Le rollback est dÃ©clenchÃ© par un Result en Err â€” l'erreur mÃ©tier est donc
// transactionnellement significative, ce qui est le comportement attendu : un
// email dÃ©jÃ  pris ne doit pas laisser d'Ã©vÃ©nement dans l'outbox.
//
// Une transaction dÃ©jÃ  ouverte n'est pas imbriquÃ©e : la fonction est exÃ©cutÃ©e
// dans la transaction courante. C'est ce qui permet de composer plusieurs
// dÃ©corateurs transactionnels sans surprise.
func RunInTx[T any, E any](
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

// runWithRollback isole la mÃ©canique de validation/annulation.
func runWithRollback[T any, E any](
	ctx context.Context,
	tx pgx.Tx,
	fn func(context.Context) result.Result[T, E],
) result.Result[T, E] {
	committed := false
	defer func() {
		if !committed {
			// Le contexte parent peut Ãªtre annulÃ© : on annule sur un contexte
			// neuf, sinon le rollback lui-mÃªme Ã©choue et la connexion reste sale.
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	if tenantID := TenantFrom(ctx); tenantID != "" {
		// SET LOCAL est bornÃ© Ã  la transaction : aucune fuite d'Ã©tat vers le
		// pool, donc aucun risque qu'une requÃªte suivante hÃ©rite du tenant.
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
// C'est le mÃ©canisme d'Ã©lection utilisÃ© par l'ordonnanceur : derriÃ¨re N
// rÃ©pliques, une seule obtient le verrou et exÃ©cute la tÃ¢che. Le verrou est
// libÃ©rÃ© Ã  la fermeture de la connexion, donc la mort d'une rÃ©plique ne bloque
// pas les autres durablement.
func TryAdvisoryLock(ctx context.Context, q Querier, key int64) (bool, error) {
	var acquired bool
	if err := q.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	return acquired, nil
}

// ReleaseAdvisoryLock libÃ¨re un verrou consultatif.
func ReleaseAdvisoryLock(ctx context.Context, q Querier, key int64) error {
	if _, err := q.Exec(ctx, "SELECT pg_advisory_unlock($1)", key); err != nil {
		return fmt.Errorf("pg_advisory_unlock: %w", err)
	}
	return nil
}

// Codes d'erreur Postgres utilisÃ©s pour la traduction en erreurs de domaine.
// Un adaptateur secondaire ne doit jamais laisser remonter une erreur de pilote
// (rules/donnees-et-migrations.md Â§2).
const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
	CodeCheckViolation      = "23514"
	CodeQueryCanceled       = "57014"
	CodeSerializationFailure = "40001"
)

// PgErrorCode extrait le code SQLSTATE d'une erreur, ou la chaÃ®ne vide.
func PgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// ConstraintName extrait le nom de la contrainte violÃ©e, ou la chaÃ®ne vide.
//
// C'est ce nom qui permet de traduire une violation d'unicitÃ© en erreur mÃ©tier
// prÃ©cise : une table peut porter plusieurs contraintes uniques.
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// IsNotFound indique une absence de ligne. C'est un cas nominal, pas un dÃ©faut.
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsUnavailable indique une indisponibilitÃ© transitoire du stockage : Ã  traduire
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
