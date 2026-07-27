// Package domain porte le vocabulaire des tâches périodiques, sans dépendance.
//
// # Le vrai problème n'est pas la périodicité
//
// Un ticker suffit à répéter. Le problème est que N répliques exécuteraient la
// tâche N fois : un rappel envoyé trois fois, une facture émise trois fois. Toute
// la valeur du module est dans l'ÉLECTION, pas dans l'horloge — c'est pourquoi
// l'élection est un port et la boucle une simple orchestration.
package domain

import (
	"errors"
	"fmt"
	"hash/fnv"
	"time"
)

// TaskName identifie une tâche. C'est aussi la clé d'élection : deux tâches de
// même nom s'excluent mutuellement, y compris entre répliques.
type TaskName string

// String rend le nom sous forme brute.
func (n TaskName) String() string { return string(n) }

// ErrInvalidTask refuse une tâche inexploitable.
var ErrInvalidTask = errors.New("tâche planifiée invalide")

// Task décrit une tâche périodique. Volontairement sans comportement : le travail
// est un port (ports.Job), pas un champ de structure, pour que le domaine reste
// pur et que la description d'une tâche se compare et se journalise.
type Task struct {
	Name  TaskName
	Every time.Duration
	// Timeout borne une exécution. À zéro, la période sert de borne : une tâche qui
	// dépasse sa propre période est déjà en retard, la laisser courir davantage ne
	// fait qu'empiler les exécutions.
	Timeout time.Duration
}

// Validate refuse une tâche qui ne peut pas être exécutée sainement.
//
// Une période nulle ou négative ferait paniquer time.NewTicker : le refus au
// démarrage vaut mieux qu'un plantage à la première minute d'exécution.
func (t Task) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("%w: nom manquant", ErrInvalidTask)
	}
	if t.Every <= 0 {
		return fmt.Errorf("%w: %s a une période de %v", ErrInvalidTask, t.Name, t.Every)
	}
	if t.Timeout < 0 {
		return fmt.Errorf("%w: %s a un délai négatif", ErrInvalidTask, t.Name)
	}
	return nil
}

// Deadline rend la borne d'une exécution.
func (t Task) Deadline() time.Duration {
	if t.Timeout > 0 {
		return t.Timeout
	}
	return t.Every
}

// Event nomme ce qui est arrivé à une tâche.
//
// Déclaré dans le domaine parce que l'orchestration ne journalise PAS : elle rend
// compte (ports.Report). C'est ce qui la garde pure — aucun logger, aucune I/O —
// et ce qui permet à un test de vérifier une politique d'exécution en lisant des
// valeurs plutôt qu'en analysant du JSON.
type Event string

const (
	// EventSkipped : une autre réplique tient la tâche. Cas NOMINAL, pas un incident.
	EventSkipped Event = "skipped"
	// EventSucceeded : le travail s'est terminé sans erreur.
	EventSucceeded Event = "succeeded"
	// EventFailed : le travail a retourné une erreur.
	EventFailed Event = "failed"
	// EventElectionFailed : le mécanisme d'élection est en panne. La tâche n'a PAS
	// été exécutée — c'est le repli sûr.
	EventElectionFailed Event = "election_failed"
	// EventReleaseFailed : le travail est fait mais le verrou n'a pas été rendu. La
	// tâche ne tournera plus jusqu'à péremption du verrou : à surveiller.
	EventReleaseFailed Event = "release_failed"
)

// Outcome est le compte rendu d'une tentative d'exécution.
type Outcome struct {
	Task     TaskName
	Event    Event
	Duration time.Duration
	// Err porte la cause pour EventFailed, EventElectionFailed et
	// EventReleaseFailed. Nil partout ailleurs.
	Err error
}

// LockKey dérive un entier stable du nom de la tâche.
//
// Nécessaire parce qu'un verrou consultatif PostgreSQL s'exprime en `bigint`, pas
// en texte. Le décalage d'un bit garde le résultat positif : une clé négative est
// valide côté base mais illisible dans un journal et dans `pg_locks`.
//
// # Sur le risque de collision
//
// Une collision ferait s'exclure mutuellement deux tâches DISTINCTES. C'est
// improbable sur 63 bits, et surtout le pire cas est une exécution sérialisée —
// jamais une double exécution. Le repli va donc dans le bon sens.
func LockKey(name TaskName) int64 {
	digest := fnv.New64a()
	// fnv.Write ne retourne jamais d'erreur : la signature vient de io.Writer.
	_, _ = digest.Write([]byte("scheduler:" + name.String()))
	return int64(digest.Sum64() >> 1)
}
