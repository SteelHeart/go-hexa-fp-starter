package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryAbandonsImmediatelyOnShutdown : l'arrêt ne se négocie pas.
//
// # Le défaut que ce test attrape
//
// Un `time.Sleep` à la place du `select` sur `ctx.Done()` rendrait l'attente
// insensible à l'annulation. À l'arrêt du worker, chaque publication en cours
// tiendrait son recul jusqu'au bout ; l'orchestrateur, lui, n'attend pas — il
// envoie SIGKILL après son délai de grâce.
//
// Conséquence exacte : un message publié chez le broker mais JAMAIS marqué dans
// l'outbox. C'est le seul cas de ce socle qui produit un doublon chez le
// consommateur, et il naîtrait à chaque déploiement.
//
// Le test mesure la DURÉE : un recul de dix secondes doit rendre la main
// immédiatement. Il n'y a pas d'autre façon de distinguer les deux écritures.
func TestRetryAbandonsImmediatelyOnShutdown(t *testing.T) {
	t.Parallel()

	const backoff = 10 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// L'annulation part APRÈS le premier essai : c'est l'ATTENTE entre deux
	// essais qu'on vérifie, pas le refus d'entrer dans la boucle.
	//
	// sync.Once plutôt qu'une attente active sur un compteur : le publieur et
	// l'annuleur tourneraient sur des goroutines différentes, et le compteur
	// partagé serait lui-même la course que ce fichier prétend chasser.
	var once sync.Once
	always := func(context.Context, messaging.Envelope) error {
		once.Do(cancel)
		return errors.New("broker injoignable")
	}

	publish := messaging.WithRetry(always, 5, backoff)

	started := time.Now()
	err := publish(ctx, envelope("user.registered.v1"))
	elapsed := time.Since(started)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("erreur rendue = %v, attendu context.Canceled", err)
	}
	if elapsed >= backoff {
		t.Errorf("la publication a tenu son recul (%s) malgré l'annulation — "+
			"le worker serait tué au milieu, et un message publié resterait non marqué", elapsed)
	}
}
