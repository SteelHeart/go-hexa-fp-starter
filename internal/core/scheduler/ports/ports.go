// Package ports déclare les contrats des tâches périodiques.
//
// Ce paquet ne contient QUE des déclarations de types.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Acquire tente de devenir l'exécutant d'une tâche.
//
// # Contrat de conformité
//
//   - `true` signifie « personne d'autre ne l'exécute, vas-y ». L'appelant DOIT
//     appeler Release ensuite, même en cas d'échec du travail.
//   - `false` n'est PAS une erreur : c'est le cas nominal sur toute réplique non
//     élue, et il se produit à chaque tick sur N-1 répliques.
//   - Une erreur signale une panne du mécanisme d'élection. Dans le doute, un
//     pilote rend `false` : ne pas exécuter est bénin, exécuter deux fois ne l'est
//     pas — un rappel envoyé deux fois se voit chez le client.
type Acquire = func(ctx context.Context, task domain.TaskName) (bool, error)

// Release rend l'exécution à la suivante.
//
// Doit être appelée avec un contexte NON annulé : libérer avec un contexte déjà
// mort laisserait le verrou en place jusqu'à sa péremption, et la tâche ne
// tournerait plus pendant ce temps.
type Release = func(ctx context.Context, task domain.TaskName) error

// Report rend compte d'une tentative d'exécution.
//
// L'orchestration ne journalise pas : elle rend compte. C'est ce qui la garde
// pure — `rules/README.md` interdit tout logger dans `application/` — et c'est ce
// qui permet à un test de vérifier une politique d'exécution en lisant des valeurs.
//
// Ne retourne rien : un compte rendu qui échouerait ne doit jamais faire échouer
// la tâche dont il rend compte.
type Report = func(ctx context.Context, outcome domain.Outcome)

// Now rend l'instant courant.
//
// L'orchestration ne lit pas l'horloge du système : elle reçoit ce port. Sans
// cela, aucune mesure de durée ne serait vérifiable dans un test.
type Now = func() time.Time

// Job est le travail à exécuter.
//
// Fourni par l'appelant, jamais par le module : le socle sait quand exécuter, il
// n'a aucune idée de quoi exécuter.
//
// Un Job DOIT être idempotent. Même avec une élection correcte, une réplique tuée
// entre son travail et sa libération laisse une exécution partielle qu'une autre
// réplique reprendra.
type Job = func(ctx context.Context) error
