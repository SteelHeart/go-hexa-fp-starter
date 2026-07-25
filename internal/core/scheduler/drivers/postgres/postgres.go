// Package postgres implémente l'élection par verrou consultatif PostgreSQL.
//
// # GARANTIES
//
//   - **Une seule réplique exécute**, sans composant supplémentaire à opérer : pas
//     de Zookeeper, pas d'etcd, pas de bail à renouveler.
//   - **Libération automatique à la mort d'une réplique.** Le verrou est de
//     SESSION : si le processus disparaît, la base ferme la connexion et le verrou
//     tombe. Aucun bail expiré à nettoyer à la main — c'est ce qui rend ce
//     mécanisme préférable à une ligne `leader` dans une table.
//
// # NON-GARANTIES
//
//   - **Ne survit pas à un basculement de primaire.** Les verrous consultatifs ne
//     sont pas répliqués : pendant une bascule, deux répliques peuvent s'estimer
//     élues. Une tâche doit donc rester idempotente même avec ce pilote.
//   - **Une collision de clé sérialiserait deux tâches distinctes** (voir
//     domain.LockKey). Improbable sur 63 bits, et le pire cas va dans le bon sens.
//   - **Une tâche déjà en cours dans le processus n'est pas relancée**, comme avec
//     le pilote `inproc` : le recouvrement local est refusé avant même d'ouvrir une
//     connexion.
//
// # État
//
// Écrit, JAMAIS exécuté contre une base. Aucune migration n'est nécessaire — les
// verrous consultatifs ne créent pas de table — mais rien ne prouve encore ce code.
package postgres

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Elector accorde l'exécution à une seule réplique.
type Elector struct {
	pool *pgxpool.Pool

	// held retient la connexion qui PORTE le verrou.
	//
	// Indispensable : un verrou consultatif de session appartient à sa connexion.
	// Le prendre sur une connexion rendue au pool aussitôt après revient à ne rien
	// verrouiller — la connexion servirait à quelqu'un d'autre, verrou compris.
	mu   sync.Mutex
	held map[domain.TaskName]*pgxpool.Conn
}

// New construit l'électeur.
func New(pool *pgxpool.Pool) *Elector {
	return &Elector{pool: pool, held: make(map[domain.TaskName]*pgxpool.Conn)}
}

// Acquire implémente ports.Acquire.
func (e *Elector) Acquire(ctx context.Context, task domain.TaskName) (bool, error) {
	if e.busy(task) {
		return false, nil
	}

	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("connexion indisponible pour l'élection de %s: %w", task, err)
	}

	var elected bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, domain.LockKey(task)).Scan(&elected)
	if err != nil {
		conn.Release()
		return false, fmt.Errorf("prise du verrou d'élection de %s: %w", task, err)
	}
	if !elected {
		conn.Release()
		return false, nil
	}

	e.mu.Lock()
	e.held[task] = conn
	e.mu.Unlock()
	return true, nil
}

// busy indique si le processus courant tient déjà la tâche.
func (e *Elector) busy(task domain.TaskName) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, held := e.held[task]
	return held
}

// Release implémente ports.Release.
//
// # Pourquoi la connexion est DÉTRUITE quand le déverrouillage échoue
//
// Rendre au pool une connexion qui tient encore un verrou de session le rendrait
// permanent : personne ne le libérerait jamais, et la tâche ne tournerait plus
// jusqu'au redémarrage du processus. Fermer la connexion force la base à libérer
// tout ce que la session tenait. Perdre une connexion est le moindre mal.
func (e *Elector) Release(ctx context.Context, task domain.TaskName) error {
	e.mu.Lock()
	conn, held := e.held[task]
	delete(e.held, task)
	e.mu.Unlock()

	if !held {
		return nil
	}

	var released bool
	err := conn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, domain.LockKey(task)).Scan(&released)
	if err != nil || !released {
		if hijacked := conn.Hijack(); hijacked != nil {
			hijacked.Close(ctx)
		}
		if err != nil {
			return fmt.Errorf("libération du verrou d'élection de %s: %w", task, err)
		}
		return fmt.Errorf("verrou d'élection de %s non détenu par cette session", task)
	}

	conn.Release()
	return nil
}
