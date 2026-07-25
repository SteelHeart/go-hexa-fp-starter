// Package audit écrit un journal d'audit en ajout seul.
//
// « Ajout seul » n'est pas une intention, c'est une contrainte : la migration
// révoque UPDATE et DELETE sur la table pour le rôle applicatif. Un journal
// qu'on peut réécrire ne prouve rien.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Entry est un fait d'audit. Les champs sont volontairement pauvres : un journal
// d'audit répond à « qui a fait quoi, quand, sur quoi », pas à « pourquoi ».
type Entry struct {
	Actor      string
	Action     string
	EntityType string
	EntityID   string
	// Metadata ne doit contenir AUCUNE donnée personnelle en clair : le journal
	// est conservé longtemps et lu par des humains (rules/securite.md §5).
	Metadata map[string]any
	At       time.Time
}

// Record enregistre un fait d'audit.
type Record = func(ctx context.Context, entry Entry) error

// NewRecorder construit l'enregistreur.
//
// L'écriture se fait via database.Querier, donc DANS la transaction métier si
// elle existe : un fait annulé ne laisse pas de trace d'audit mensongère.
func NewRecorder(pool database.Querier) Record {
	return func(ctx context.Context, entry Entry) error {
		metadata, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("sérialisation des métadonnées d'audit: %w", err)
		}
		const query = `
			INSERT INTO audit_log (actor, action, entity_type, entity_id, metadata, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6)`
		at := entry.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		if _, err := pool.Exec(ctx, query,
			entry.Actor, entry.Action, entry.EntityType, entry.EntityID, metadata, at,
		); err != nil {
			return fmt.Errorf("écriture du journal d'audit: %w", err)
		}
		return nil
	}
}

// Discard est un enregistreur inerte, pour les tests et les binaires qui n'ont
// pas de base.
func Discard() Record {
	return func(context.Context, Entry) error { return nil }
}
