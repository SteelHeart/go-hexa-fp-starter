// Package application dépile l'outbox : réserver, publier, enregistrer le sort.
//
// Ce paquet ne connaît AUCUN pilote et ne fait AUCUNE I/O : il reçoit le magasin,
// le publieur, le compte rendu et l'horloge sous forme de types fonction. C'est ce
// qui permet de prouver la politique de dépilage — recul exponentiel, abandon
// après N essais, survie à un message empoisonné — avec des closures, sans base et
// sans attendre.
//
// Conformément à `rules/README.md` § « le cœur est pur » : ni `time.Now()`, ni
// logger, ni `panic` levé ici.
//
// # Pourquoi ce dépileur n'utilise pas le module scheduler
//
// L'ordonnanceur existe pour élire UNE réplique parmi N. Le dépileur n'en a pas
// besoin : le contrat de `ports.Claim` garantit déjà que deux appels concurrents
// ne retournent jamais le même message — `FOR UPDATE SKIP LOCKED` côté postgres, un
// verrou local côté memory. Plusieurs dépileurs travaillent donc EN PARALLÈLE sans
// se coordonner, ce qui est mieux qu'un seul élu : le débit croît avec les
// répliques au lieu d'être plafonné par la plus lente.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Ports porte les contrats dont le dépileur a besoin.
//
// Regroupés dans une structure plutôt qu'en paramètres : six arguments
// positionnels de même forme s'inversent au premier ajout, et le compilateur ne
// dirait rien — `MarkDone` et `MarkFailed` ont la même signature à un type près.
type Ports struct {
	Claim      ports.Claim
	MarkDone   ports.MarkDone
	MarkFailed ports.MarkFailed
	Handle     ports.Handler
	Report     ports.Report
	Now        ports.Now
}

// Policy porte la politique de dépilage.
type Policy struct {
	// BatchSize borne un lot. Trop grand, une réplique monopolise le travail et
	// tient ses verrous longtemps ; trop petit, le débit s'effondre en aller-retours.
	BatchSize int
	// Interval est la période de scrutation quand le dépileur tourne seul.
	Interval time.Duration
	// Retry appartient au DOMAINE : c'est lui qui sait ce qu'est un recul
	// exponentiel borné, et c'est lui qu'on teste pour le prouver. L'orchestration
	// ne fait que la transporter.
	Retry domain.RetryPolicy
}

// ErrMissingPort refuse un dépileur incomplet.
var ErrMissingPort = errors.New("port manquant pour le dépileur de l'outbox")

// ErrInvalidPolicy refuse une politique inexploitable.
var ErrInvalidPolicy = errors.New("politique de dépilage invalide")

// ErrHandlerPanicked signale un publieur qui a paniqué.
var ErrHandlerPanicked = errors.New("le publieur a paniqué")

// Dispatcher dépile l'outbox.
type Dispatcher struct {
	ports  Ports
	policy Policy
}

// NewDispatcher construit le dépileur.
//
// Refuse un port nil ou une politique absurde plutôt que de paniquer au premier
// tour : `time.NewTicker(0)` panique, et une panique dans la goroutine d'un worker
// emporte tout le processus.
func NewDispatcher(p Ports, policy Policy) (*Dispatcher, error) {
	if p.Claim == nil || p.MarkDone == nil || p.MarkFailed == nil ||
		p.Handle == nil || p.Report == nil || p.Now == nil {
		return nil, ErrMissingPort
	}
	if err := validate(policy); err != nil {
		return nil, err
	}
	return &Dispatcher{ports: p, policy: policy}, nil
}

// validate refuse une politique qui produirait un comportement absurde.
func validate(policy Policy) error {
	switch {
	case policy.BatchSize <= 0:
		return fmt.Errorf("%w: lot de %d message(s)", ErrInvalidPolicy, policy.BatchSize)
	case policy.Retry.MaxAttempts <= 0:
		// Zéro tentative signifierait « abandonner avant d'essayer » : tout message
		// passerait en `failed` sans qu'aucune publication soit tentée.
		return fmt.Errorf("%w: %d tentative(s) autorisée(s)", ErrInvalidPolicy, policy.Retry.MaxAttempts)
	case policy.Retry.BaseBackoff <= 0:
		// Un recul nul rejouerait un message en échec sans aucune pause, en boucle.
		return fmt.Errorf("%w: recul de base de %v", ErrInvalidPolicy, policy.Retry.BaseBackoff)
	case policy.Interval <= 0:
		return fmt.Errorf("%w: période de scrutation de %v", ErrInvalidPolicy, policy.Interval)
	}
	return nil
}

// Run dépile jusqu'à l'annulation du contexte.
func (d *Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.policy.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// L'annulation est la fin NORMALE d'un worker : un arrêt propre ne doit
			// ressembler à une panne ni dans le code de retour, ni dans les alertes.
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("dépileur de l'outbox interrompu: %w", err)
			}
			return nil
		case <-ticker.C:
			d.catchUp(ctx)
		}
	}
}

// maxRounds borne le rattrapage d'un tour de scrutation.
//
// Sans borne, un retard de cent mille messages monopoliserait la boucle : le
// `select` ne serait plus jamais atteint, et le worker ignorerait la demande
// d'arrêt jusqu'à avoir tout vidé — un déploiement se transformerait en attente.
const maxRounds = 10

// catchUp enchaîne les lots tant qu'ils reviennent pleins.
//
// Un lot plein signifie « il en reste probablement » : attendre le tick suivant
// ferait avancer le rattrapage d'un seul lot par période, et le retard ne se
// résorberait jamais si le débit d'entrée dépasse un lot par tour.
func (d *Dispatcher) catchUp(ctx context.Context) {
	for range maxRounds {
		if ctx.Err() != nil {
			return
		}
		count, err := d.DrainOnce(ctx)
		if err != nil || count < d.policy.BatchSize {
			return
		}
	}
}

// DrainOnce réserve un lot et le traite. Rend le nombre de messages traités.
//
// Exportée pour qu'un déclencheur externe — un ordonnanceur, un test, une commande
// d'exploitation — applique exactement la même politique sans la boucle.
func (d *Dispatcher) DrainOnce(ctx context.Context) (int, error) {
	messages, err := d.ports.Claim(ctx, d.policy.BatchSize)
	if err != nil {
		d.ports.Report(ctx, domain.Outcome{Event: domain.EventClaimFailed, Err: err})
		return 0, fmt.Errorf("réservation d'un lot de l'outbox: %w", err)
	}

	for _, msg := range messages {
		d.deliver(ctx, msg)
	}
	return len(messages), nil
}

// deliver publie un message et enregistre son sort.
func (d *Dispatcher) deliver(ctx context.Context, msg domain.Message) {
	started := d.ports.Now()
	err := d.publish(ctx, msg)
	elapsed := d.ports.Now().Sub(started)

	if err != nil {
		d.reschedule(ctx, msg, elapsed, err)
		return
	}

	if markErr := d.ports.MarkDone(ctx, msg.ID); markErr != nil {
		// Publié, mais non marqué : le message reste « en attente » et sera
		// republié. Le consommateur recevra un doublon — d'où l'obligation
		// d'idempotence côté consommateur (ports.Handler).
		d.report(ctx, msg, domain.Outcome{
			Event: domain.EventResolveFailed, Attempts: msg.Attempts, Duration: elapsed, Err: markErr,
		})
		return
	}

	d.report(ctx, msg, domain.Outcome{
		Event: domain.EventPublished, Attempts: msg.Attempts, Duration: elapsed,
	})
}

// publish appelle le publieur en le protégeant d'une panique.
//
// # Pourquoi un recover ici, alors que le cœur ne panique jamais
//
// La règle interdit de LEVER une panique dans le cœur. Le publieur, lui, est du
// code d'appelant : un accès à une carte nil dans un consommateur suffit. Sans ce
// filet, un seul message empoisonné tuerait le worker — et, avec le pilote memory,
// le laisserait réservé pour toujours. Traité comme un échec ordinaire, il suit la
// politique de recul puis finit en `failed`, ce qui est exactement le bon sort.
func (d *Dispatcher) publish(ctx context.Context, msg domain.Message) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w (%s): %v", ErrHandlerPanicked, msg.Type, recovered)
		}
	}()

	if handleErr := d.ports.Handle(ctx, msg); handleErr != nil {
		return fmt.Errorf("publication de %s (%s): %w", msg.ID, msg.Type, handleErr)
	}
	return nil
}

// reschedule applique la politique de réessai après un échec.
//
// Ne décide rien lui-même : le calcul du recul et la décision d'abandon viennent
// de domain.NextAttempt, qui est pur et testé séparément.
func (d *Dispatcher) reschedule(ctx context.Context, msg domain.Message, elapsed time.Duration, cause error) {
	attempt := domain.NextAttempt(msg, d.policy.Retry, d.ports.Now(), cause.Error())

	event := domain.EventRetryScheduled
	switch {
	case errors.Is(cause, ErrHandlerPanicked):
		event = domain.EventHandlerPanicked
	case attempt.Status == domain.StatusFailed:
		event = domain.EventExhausted
	}

	if markErr := d.ports.MarkFailed(ctx, attempt); markErr != nil {
		d.report(ctx, msg, domain.Outcome{
			Event: domain.EventResolveFailed, Attempts: attempt.Attempts,
			Duration: elapsed, Err: markErr,
		})
		return
	}

	d.report(ctx, msg, domain.Outcome{
		Event: event, Attempts: attempt.Attempts, Duration: elapsed, Err: cause,
	})
}

// report complète un compte rendu avec l'identité du message.
func (d *Dispatcher) report(ctx context.Context, msg domain.Message, outcome domain.Outcome) {
	outcome.ID = msg.ID
	outcome.Type = msg.Type
	d.ports.Report(ctx, outcome)
}
