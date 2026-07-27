package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptBackoffIsExponentialAndBounded(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	base := time.Second

	cases := map[string]struct {
		attempts int
		want     time.Duration
	}{
		"premier echec":  {attempts: 0, want: 2 * time.Second},
		"deuxieme echec": {attempts: 1, want: 4 * time.Second},
		"troisieme":      {attempts: 2, want: 8 * time.Second},
		// Le décalage est borné : sans borne, 1<<40 déborderait et produirait une
		// durée NÉGATIVE, donc un message rejoué en boucle immédiatement.
		"decalage borne a 2^10": {attempts: 40, want: 1024 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := domain.NextAttempt(
				domain.Message{Attempts: tc.attempts},
				domain.RetryPolicy{MaxAttempts: 100, BaseBackoff: base},
				now, "raison",
			)
			if delay := got.AvailableAt.Sub(now); delay != tc.want {
				t.Errorf("recul = %v, attendu %v", delay, tc.want)
			}
			if got.AvailableAt.Before(now) {
				t.Error("le prochain essai est dans le passé : débordement de durée")
			}
		})
	}
}
