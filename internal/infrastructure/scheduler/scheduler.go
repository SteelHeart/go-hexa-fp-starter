// Package scheduler exécute des tâches périodiques, une seule fois quel que
// soit le nombre de répliques.
//
// Le problème réel n'est pas la périodicité — un ticker suffit — mais le fait
// que N répliques exécuteraient la tâche N fois. La solution est un verrou
// consultatif Postgres : pas de composant supplémentaire à opérer, et le verrou
// est libéré automatiquement si la réplique meurt.
package scheduler

import (
	"context"
	"errors"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Task est une tâche périodique.
type Task struct {
	Name    string
	Every   time.Duration
	Run     func(context.Context, database.Querier) error
	Timeout time.Duration
}

// Scheduler exécute les tâches enregistrées.
type Scheduler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	tasks  []Task
}

// New construit l'ordonnanceur.
func New(pool *pgxpool.Pool, logger *slog.Logger, tasks ...Task) *Scheduler {
	return &Scheduler{pool: pool, logger: logger, tasks: tasks}
}

// Run exécute toutes les tâches jusqu'à l'annulation du contexte.
func (s *Scheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, task := range s.tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			s.loop(ctx, t)
		}(task)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err //nolint:wrapcheck // erreur de contexte, déjà explicite
	}
	return nil
}

func (s *Scheduler) loop(ctx context.Context, task Task) {
	ticker := time.NewTicker(task.Every)
	defer ticker.Stop()
	s.logger.InfoContext(ctx, "tâche planifiée démarrée",
		slog.String("task", task.Name), slog.Duration("every", task.Every))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx, task)
		}
	}
}

// runOnce tente d'obtenir le verrou puis exécute. Ne pas obtenir le verrou est
// le cas nominal sur les répliques non élues : ce n'est pas une erreur.
func (s *Scheduler) runOnce(ctx context.Context, task Task) {
	timeout := task.Timeout
	if timeout == 0 {
		timeout = task.Every
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Le verrou est pris sur une connexion dédiée : un verrou de session pris
	// depuis le pool serait libéré au retour de la connexion, donc inutile.
	conn, err := s.pool.Acquire(runCtx)
	if err != nil {
		s.logger.WarnContext(runCtx, "connexion indisponible pour la tâche",
			slog.String("task", task.Name), slog.Any("error", err))
		return
	}
	defer conn.Release()

	key := lockKey(task.Name)
	acquired, err := database.TryAdvisoryLock(runCtx, conn, key)
	if err != nil {
		s.logger.WarnContext(runCtx, "prise de verrou impossible",
			slog.String("task", task.Name), slog.Any("error", err))
		return
	}
	if !acquired {
		return
	}
	defer func() {
		if err := database.ReleaseAdvisoryLock(context.WithoutCancel(runCtx), conn, key); err != nil {
			s.logger.WarnContext(runCtx, "libération de verrou impossible",
				slog.String("task", task.Name), slog.Any("error", err))
		}
	}()

	started := time.Now()
	if err := task.Run(runCtx, conn); err != nil {
		s.logger.ErrorContext(runCtx, "tâche planifiée en échec",
			slog.String("task", task.Name),
			slog.Duration("duration", time.Since(started)),
			slog.Any("error", err))
		return
	}
	s.logger.DebugContext(runCtx, "tâche planifiée terminée",
		slog.String("task", task.Name), slog.Duration("duration", time.Since(started)))
}

// lockKey dérive un entier stable du nom de la tâche.
//
// Une collision ferait s'exclure mutuellement deux tâches distinctes : c'est
// improbable sur 63 bits, et le pire cas reste une exécution sérialisée, pas
// une double exécution.
func lockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("scheduler:" + name))
	return int64(h.Sum64() >> 1)
}
