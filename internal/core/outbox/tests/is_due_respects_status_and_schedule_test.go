package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestIsDueRespectsStatusAndSchedule(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	cases := map[string]struct {
		msg  domain.Message
		want bool
	}{
		"pending et echu":             {msg: domain.Message{Status: domain.StatusPending, AvailableAt: now}, want: true},
		"pending mais futur":          {msg: domain.Message{Status: domain.StatusPending, AvailableAt: now.Add(time.Minute)}, want: false},
		"deja traite":                 {msg: domain.Message{Status: domain.StatusDone, AvailableAt: now}, want: false},
		"abandonne, jamais rejouable": {msg: domain.Message{Status: domain.StatusFailed, AvailableAt: now}, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.msg.IsDue(now); got != tc.want {
				t.Errorf("IsDue = %v, attendu %v", got, tc.want)
			}
		})
	}
}
