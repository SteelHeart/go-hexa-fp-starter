// Package outbox implémente l'outbox transactionnel : la seule sortie autorisée
// vers le monde extérieur (documentation/adr/006).
//
// Écriture et publication ne peuvent pas être atomiques entre deux systèmes.
// On rend donc atomique ce qui peut l'être — l'écriture métier et l'intention de
// publier — puis on publie séparément, en acceptant le « au moins une fois ».
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// traceParentFrom extrait le contexte de trace au format W3C.
//
// Sans lui, la consommation asynchrone est aveugle : on ne peut plus rattacher
// l'envoi d'un courriel à la requête qui l'a déclenché.
func traceParentFrom(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ContextWithTrace restaure le contexte de trace d'un message pour que le
// traitement asynchrone apparaisse dans la même trace que la requête d'origine.
func ContextWithTrace(ctx context.Context, msg Message) context.Context {
	if msg.TraceParent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{"traceparent": msg.TraceParent}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// Message est une intention de publication persistée.
type Message struct {
	ID          uuid.UUID
	Type        string
	AggregateID string
	Payload     []byte
	TraceParent string
	Attempts    int
	CreatedAt   time.Time
}

// Handler traite un message. Il DOIT être idempotent : le dépilage est « au
// moins une fois », donc un message sera rejoué au moins une fois dans la vie
// du système.
type Handler = func(ctx context.Context, msg Message) error

// ErrNoHandler signale un type d'événement sans consommateur enregistré.
var ErrNoHandler = errors.New("aucun consommateur pour ce type d'événement")

// Enqueue insère un message. À appeler DANS la transaction métier : c'est tout
// l'intérêt du patron, et database.Querier s'en charge automatiquement.
func Enqueue(ctx context.Context, q database.Querier, eventType, aggregateID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sérialisation de l'événement %s: %w", eventType, err)
	}
	const query = `
		INSERT INTO outbox_messages (id, event_type, aggregate_id, payload, trace_parent)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := q.Exec(ctx, query,
		uuid.New(), eventType, aggregateID, raw, traceParentFrom(ctx),
	); err != nil {
		return fmt.Errorf("insertion dans l'outbox: %w", err)
	}
	return nil
}

// Dispatcher dépile et publie.
type Dispatcher struct {
	querier     database.Querier
	handlers    map[string]Handler
	logger      *slog.Logger
	batchSize   int
	maxAttempts int
	baseBackoff time.Duration
}

// Options porte la configuration du dépileur.
type Options struct {
	BatchSize   int
	MaxAttempts int
	BaseBackoff time.Duration
}

// NewDispatcher construit un dépileur.
func NewDispatcher(
	q database.Querier,
	handlers map[string]Handler,
	logger *slog.Logger,
	opts Options,
) *Dispatcher {
	return &Dispatcher{
		querier:     q,
		handlers:    handlers,
		logger:      logger,
		batchSize:   opts.BatchSize,
		maxAttempts: opts.MaxAttempts,
		baseBackoff: opts.BaseBackoff,
	}
}

// claimQuery verrouille un lot de messages dus.
//
// FOR UPDATE SKIP LOCKED permet à N répliques de dépiler sans coordination :
// chacune saute les lignes que les autres tiennent déjà.
const claimQuery = `
	SELECT id, event_type, aggregate_id, payload, trace_parent, attempts, created_at
	FROM outbox_messages
	WHERE processed_at IS NULL
	  AND status = 'pending'
	  AND available_at <= now()
	ORDER BY available_at
	LIMIT $1
	FOR UPDATE SKIP LOCKED`

// RunOnce traite un lot et retourne le nombre de messages traités.
func (d *Dispatcher) RunOnce(ctx context.Context) (int, error) {
	rows, err := d.querier.Query(ctx, claimQuery, d.batchSize)
	if err != nil {
		return 0, fmt.Errorf("lecture de l'outbox: %w", err)
	}
	messages := make([]Message, 0, d.batchSize)
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.Type, &m.AggregateID, &m.Payload, &m.TraceParent, &m.Attempts, &m.CreatedAt,
		); err != nil {
			rows.Close()
			return 0, fmt.Errorf("lecture d'un message: %w", err)
		}
		messages = append(messages, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("parcours de l'outbox: %w", err)
	}

	for _, msg := range messages {
		d.process(ctx, msg)
	}
	return len(messages), nil
}

// process traite un message et enregistre son issue. Il ne retourne pas
// d'erreur : l'échec d'un message ne doit pas interrompre le lot.
func (d *Dispatcher) process(ctx context.Context, msg Message) {
	handler, found := d.handlers[msg.Type]
	if !found {
		d.fail(ctx, msg, ErrNoHandler)
		return
	}
	if err := handler(ctx, msg); err != nil {
		d.fail(ctx, msg, err)
		return
	}
	const query = `UPDATE outbox_messages SET processed_at = now(), status = 'done' WHERE id = $1`
	if _, err := d.querier.Exec(ctx, query, msg.ID); err != nil {
		d.logger.ErrorContext(ctx, "message publié mais non marqué traité",
			slog.String("outbox_id", msg.ID.String()),
			slog.String("event_type", msg.Type),
			slog.Any("error", err),
		)
	}
}

// fail programme un réessai avec recul exponentiel, ou abandonne définitivement.
//
// Un message abandonné est marqué 'failed', jamais supprimé : c'est la seule
// trace de ce qui n'a pas été publié.
func (d *Dispatcher) fail(ctx context.Context, msg Message, cause error) {
	attempts := msg.Attempts + 1
	status := "pending"
	if attempts >= d.maxAttempts {
		status = "failed"
	}
	backoff := d.baseBackoff * time.Duration(1<<min(attempts, 10))

	const query = `
		UPDATE outbox_messages
		SET attempts = $2, status = $3, available_at = now() + $4::interval, last_error = $5
		WHERE id = $1`
	if _, err := d.querier.Exec(ctx, query,
		msg.ID, attempts, status, backoff.String(), cause.Error(),
	); err != nil {
		d.logger.ErrorContext(ctx, "échec d'enregistrement de l'échec d'un message",
			slog.String("outbox_id", msg.ID.String()), slog.Any("error", err))
	}

	level := slog.LevelWarn
	if status == "failed" {
		// Abandon définitif : une action humaine est requise, donc Error.
		level = slog.LevelError
	}
	d.logger.Log(ctx, level, "publication d'un événement en échec",
		slog.String("outbox_id", msg.ID.String()),
		slog.String("event_type", msg.Type),
		slog.Int("attempts", attempts),
		slog.String("status", status),
		slog.Any("error", cause),
	)
}

// PendingCount compte les messages en attente.
//
// C'est la métrique la plus importante du système : elle croît quand le worker
// est mort, et c'est le seul symptôme visible d'une chaîne asynchrone en panne.
func PendingCount(ctx context.Context, q database.Querier) (int64, error) {
	const query = `SELECT count(*) FROM outbox_messages WHERE processed_at IS NULL AND status = 'pending'`
	var count int64
	if err := q.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("comptage de l'outbox: %w", err)
	}
	return count, nil
}
