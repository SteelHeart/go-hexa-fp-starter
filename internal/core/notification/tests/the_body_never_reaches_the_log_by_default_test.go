package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
)

// TestTheBodyNeverReachesTheLogByDefault garde le défaut sûr.
//
// # Ce que ce test protège
//
// Un corps de notification transporte régulièrement un secret : lien de
// confirmation, jeton de réinitialisation, code à usage unique. Ce sont des
// identifiants AU PORTEUR — qui les lit peut s'en servir, sans mot de passe et
// sans laisser de trace.
//
// Le journal applicatif est précisément ce qu'on exporte le plus volontiers vers
// un collecteur tiers. Un corps journalisé par défaut transformerait donc chaque
// fuite de journaux en prise de contrôle de comptes.
func TestTheBodyNeverReachesTheLogByDefault(t *testing.T) {
	t.Parallel()

	mod, j := newModule(t, nil)
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("envoi: %v", err)
	}

	trace := j.texte()
	if trace == "" {
		t.Fatal("le pilote n'a rien écrit : le test n'observe rien")
	}
	if strings.Contains(trace, secretInBody) {
		t.Fatalf("le corps fuite dans le journal : %s", trace)
	}
	if strings.Contains(trace, recipient) {
		t.Fatalf("l'adresse en clair fuite dans le journal : %s", trace)
	}

	// Ce qui DOIT y être : de quoi diagnostiquer sans rien révéler.
	for _, attendu := range []string{"notification", subject, "body_bytes"} {
		if !strings.Contains(trace, attendu) {
			t.Errorf("le journal doit porter %q : %s", attendu, trace)
		}
	}
}

// TestTheBodyIsLoggedOnlyWhenAsked constate que l'option est LUE.
//
// Une option de configuration silencieusement ignorée est le défaut qui a coûté
// l'issue #93 : le serveur démarrait, montait le pilote, et ne disait rien.
//
// Le test vérifie aussi que l'avertissement voyage dans le MÊME enregistrement
// que le corps. Dans deux lignes séparées, la première se perd au tri — et celui
// qui lit le secret ne lit jamais la mise en garde.
func TestTheBodyIsLoggedOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	mod, j := newModule(t, map[string]any{notification.OptionBody: notification.BodyLogged})
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("envoi: %v", err)
	}

	trace := j.texte()
	if !strings.Contains(trace, secretInBody) {
		t.Fatalf("le corps devait être journalisé : %s", trace)
	}
	if !strings.Contains(trace, "développement uniquement") {
		t.Fatalf("l'avertissement doit accompagner le corps : %s", trace)
	}

	// L'adresse reste masquée MÊME quand le corps est écrit : les deux
	// décisions sont indépendantes, et confondre les deux ferait tomber la
	// seconde en activant la première.
	if strings.Contains(trace, recipient) {
		t.Fatalf("l'adresse en clair fuite alors que seul le corps était demandé : %s", trace)
	}
}

// TestAnUnknownBodyOptionRefusesStartup : deny par défaut jusque dans l'option.
//
// Le repli sur `masked` serait pourtant tentant — il est « sûr », puisqu'il tait
// le corps. Mais quelqu'un qui écrit `body: logué` croirait avoir demandé le
// corps, ne le verrait pas, et chercherait ailleurs pendant une heure.
func TestAnUnknownBodyOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	j := newJournal()
	for _, valeur := range []any{"logué", "true", "verbose", ""} {
		_, err := notification.New(
			configModule(map[string]any{notification.OptionBody: valeur}),
			notification.Deps{Logger: j.logger},
		)
		if err == nil {
			t.Errorf("body=%q : le démarrage doit être refusé", valeur)
		}
	}
}
