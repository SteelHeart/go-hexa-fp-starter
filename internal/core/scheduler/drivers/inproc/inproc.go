// Package inproc implémente l'élection à l'intérieur d'un seul processus.
//
// # NON-GARANTIES — à lire avant de l'utiliser
//
//   - **Aucune élection entre répliques.** C'est LA non-garantie, et elle annule
//     la raison d'être du module : deux instances exécuteront la tâche deux fois.
//     Un rappel envoyé deux fois se voit chez le client ; une facture émise deux
//     fois se voit en comptabilité.
//   - **Aucune persistance.** Un redémarrage pendant une exécution laisse la tâche
//     inachevée, sans trace.
//
// Ce qu'il garantit, en revanche, et qui n'est pas rien : **aucun recouvrement**.
// Une tâche encore en cours quand le tick suivant arrive n'est pas relancée. Sans
// ça, une tâche plus lente que sa période empilerait les exécutions jusqu'à
// saturer le processus.
//
// Convient en développement, en test, pour une CLI et pour tout déploiement à UNE
// seule réplique. Ne convient PAS dès qu'il y en a deux — passer alors au pilote
// `advisory-lock`, sans toucher au code appelant.
package inproc

import (
	"context"
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Elector accorde l'exécution dans le processus courant.
type Elector struct {
	mu      sync.Mutex
	running map[domain.TaskName]struct{}
}

// New construit l'électeur.
func New() *Elector {
	return &Elector{running: make(map[domain.TaskName]struct{})}
}

// Acquire implémente ports.Acquire.
//
// Rend `false` si la tâche est déjà en cours dans ce processus. Le verrou est tenu
// pendant toute la décision : le tester puis le poser en deux temps laisserait deux
// goroutines l'obtenir.
func (e *Elector) Acquire(_ context.Context, task domain.TaskName) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, busy := e.running[task]; busy {
		return false, nil
	}
	e.running[task] = struct{}{}
	return true, nil
}

// Release implémente ports.Release.
func (e *Elector) Release(_ context.Context, task domain.TaskName) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.running, task)
	return nil
}
