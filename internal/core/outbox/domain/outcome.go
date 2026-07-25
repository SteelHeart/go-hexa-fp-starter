package domain

import "time"

// Event nomme ce qui est arrivé à un message pendant son dépilage.
//
// Déclaré dans le domaine parce que l'orchestration ne journalise PAS : elle rend
// compte (ports.Report). C'est ce qui la garde pure — aucun logger, aucune I/O —
// et ce qui permet à un test de vérifier une politique de dépilage en lisant des
// valeurs plutôt qu'en analysant du JSON.
type Event string

const (
	// EventPublished : le message est parti et son sort est enregistré. Seul cas
	// où la chaîne est réellement bouclée.
	EventPublished Event = "published"
	// EventRetryScheduled : la publication a échoué, un nouvel essai est programmé.
	EventRetryScheduled Event = "retry_scheduled"
	// EventExhausted : les tentatives sont épuisées. Le message passe en `failed`
	// et n'est plus rejoué — il reste en base comme SEULE trace de ce qui n'a pas
	// été publié. Quelqu'un doit regarder.
	EventExhausted Event = "exhausted"
	// EventClaimFailed : impossible de réserver un lot. Le magasin est en panne ;
	// rien n'a été publié et rien n'est perdu.
	EventClaimFailed Event = "claim_failed"
	// EventResolveFailed est le plus grave, et le moins évident.
	//
	// Le message a été publié, mais son sort n'a PAS pu être enregistré. Il reste
	// donc « en attente » et sera republié au prochain tour : le consommateur
	// recevra un doublon. C'est précisément pour ça que la livraison est « au moins
	// une fois » et que tout consommateur doit être idempotent.
	EventResolveFailed Event = "resolve_failed"
	// EventHandlerPanicked : le publieur a paniqué. Traité comme un échec ordinaire
	// pour que le dépileur survive à un message empoisonné.
	EventHandlerPanicked Event = "handler_panicked"
)

// Outcome est le compte rendu du traitement d'un message.
type Outcome struct {
	ID   MessageID
	Type string
	// Event dit ce qui s'est passé.
	Event Event
	// Attempts est le nombre de tentatives APRÈS celle-ci.
	Attempts int
	// Duration mesure l'appel au publieur seul, pas l'enregistrement du sort.
	Duration time.Duration
	// Err porte la cause pour tout événement autre que EventPublished.
	Err error
}
