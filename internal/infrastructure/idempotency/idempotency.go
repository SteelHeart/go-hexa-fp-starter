// Package idempotency rend une écriture rejouable sans effet de bord.
//
// Pourquoi c'est obligatoire dès qu'il y a du mobile : sur réseau instable, le
// client renvoie sa requête sans savoir si la première a abouti. Sans clé
// d'idempotence, un POST rejoué crée deux ressources — et le doublon est
// découvert par l'utilisateur, pas par le service.
//
// Le mécanisme repose sur une contrainte d'unicité, donc sur la base : deux
// requêtes concurrentes portant la même clé ne peuvent pas passer toutes les deux.
package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// Key est une clé d'idempotence fournie par l'appelant.
type Key string

// ErrConflict signale une clé réutilisée avec une empreinte de requête
// différente. C'est un défaut du client : la même clé doit désigner la même
// requête, sinon la garantie ne veut rien dire.
var ErrConflict = errors.New("clé d'idempotence réutilisée avec une requête différente")

// ErrInFlight signale qu'une requête portant la même clé est en cours.
// L'appelant doit réessayer, pas dupliquer.
var ErrInFlight = errors.New("requête identique déjà en cours")

// Fingerprint calcule l'empreinte d'une requête, pour détecter la réutilisation
// abusive d'une clé.
func Fingerprint(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		// Une charge non sérialisable est un défaut de programmation, pas une
		// erreur d'exécution : on rend une empreinte qui ne collisionne pas.
		raw = []byte(fmt.Sprintf("%#v", payload))
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Guard protège une opération par une clé d'idempotence.
type Guard struct {
	querier database.Querier
	ttl     time.Duration
}

// NewGuard construit le garde. `ttl` est la durée pendant laquelle une clé
// consommée reste mémorisée — au-delà, un rejeu très tardif recréera la ressource.
func NewGuard(q database.Querier, ttl time.Duration) *Guard {
	return &Guard{querier: q, ttl: ttl}
}

// Reserve tente de réserver la clé.
//
// Trois issues, et elles sont toutes nécessaires :
//   - réponse mémorisée présente → la retourner telle quelle, ne pas rejouer ;
//   - réservation obtenue → exécuter l'opération, puis appeler Complete ;
//   - ErrInFlight / ErrConflict → l'appelant ne doit surtout pas exécuter.
func (g *Guard) Reserve(ctx context.Context, key Key, fingerprint string) (fp.Option[[]byte], error) {
	const query = `
		INSERT INTO idempotency_keys (key, fingerprint, status, expires_at)
		VALUES ($1, $2, 'in_flight', now() + $3::interval)
		ON CONFLICT (key) DO UPDATE
		  SET key = idempotency_keys.key
		RETURNING fingerprint, status, response`

	var (
		storedFingerprint string
		status            string
		response          []byte
	)
	err := g.querier.QueryRow(ctx, query, string(key), fingerprint, g.ttl.String()).
		Scan(&storedFingerprint, &status, &response)
	if err != nil {
		return fp.None[[]byte](), fmt.Errorf("réservation de la clé d'idempotence: %w", err)
	}

	if storedFingerprint != fingerprint {
		return fp.None[[]byte](), ErrConflict
	}
	switch status {
	case "done":
		return fp.Some(response), nil
	case "in_flight":
		// La ligne vient peut-être d'être créée par NOUS : on ne peut pas le
		// savoir depuis le RETURNING d'un upsert. Complete() tranchera.
		return fp.None[[]byte](), nil
	default:
		return fp.None[[]byte](), fmt.Errorf("statut d'idempotence inattendu: %q", status)
	}
}

// Complete mémorise la réponse pour les rejeux ultérieurs.
func (g *Guard) Complete(ctx context.Context, key Key, response []byte) error {
	const query = `
		UPDATE idempotency_keys
		SET status = 'done', response = $2, completed_at = now()
		WHERE key = $1`
	if _, err := g.querier.Exec(ctx, query, string(key), response); err != nil {
		return fmt.Errorf("mémorisation de la réponse idempotente: %w", err)
	}
	return nil
}

// Release libère une clé après un échec, pour que l'appelant puisse réessayer.
//
// Sans cette libération, une erreur transitoire rendrait l'opération
// définitivement impossible avec cette clé — le remède serait pire que le mal.
func (g *Guard) Release(ctx context.Context, key Key) error {
	const query = `DELETE FROM idempotency_keys WHERE key = $1 AND status = 'in_flight'`
	if _, err := g.querier.Exec(ctx, query, string(key)); err != nil {
		return fmt.Errorf("libération de la clé d'idempotence: %w", err)
	}
	return nil
}

// Purge supprime les clés expirées. À appeler périodiquement par l'ordonnanceur.
func Purge(ctx context.Context, q database.Querier) (int64, error) {
	const query = `DELETE FROM idempotency_keys WHERE expires_at < now()`
	tag, err := q.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("purge des clés d'idempotence: %w", err)
	}
	return tag.RowsAffected(), nil
}
