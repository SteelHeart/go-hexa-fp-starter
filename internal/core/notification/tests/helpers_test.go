// Package tests contient les tests en BOÎTE NOIRE du module notification : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
)

const (
	// recipient est l'adresse utilisée par les tests.
	recipient = "alice@example.com"

	// subject et body sont un message valide minimal.
	subject = "Bienvenue"
	body    = "Confirmez votre compte : https://exemple.test/c/JETON-SECRET-123"

	// secretInBody est ce qu'on cherche dans les journaux. Un lien de
	// confirmation EST un identifiant au porteur.
	secretInBody = "JETON-SECRET-123"
)

// journal capte ce que le pilote écrit.
//
// Le texte BRUT est conservé : une fuite se cherche dans le texte, pas dans une
// carte de champs. Un secret transporté par un champ au nom anodin échapperait à
// toute inspection champ par champ.
type journal struct {
	buffer *bytes.Buffer
	logger *slog.Logger
}

// newJournal construit un journal inspectable.
func newJournal() *journal {
	buffer := &bytes.Buffer{}
	return &journal{
		buffer: buffer,
		logger: slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

// texte rend tout ce qui a été écrit.
func (j *journal) texte() string { return j.buffer.String() }

// newModule construit le module et son journal.
func newModule(t *testing.T, options map[string]any) (notification.Module, *journal) {
	t.Helper()

	j := newJournal()
	mod, err := notification.New(
		config.Module{Enabled: true, Driver: "log", Options: options},
		notification.Deps{Logger: j.logger},
	)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod, j
}

// configModule forge une configuration de module activée sur le pilote `log`.
func configModule(options map[string]any) config.Module {
	return config.Module{Enabled: true, Driver: "log", Options: options}
}

// message forge un message valide vers le destinataire des tests.
//
// Le destinataire n'est PAS un paramètre : il est constant dans tout ce paquet,
// et un paramètre qui ne varie jamais laisse croire qu'il varie. Les tests qui
// éprouvent d'autres adresses passent par `domain.NewRecipient` directement,
// là où l'adresse EST le sujet.
func message(t *testing.T) domain.Message {
	t.Helper()

	destinataire, err := domain.NewRecipient(recipient)
	if err != nil {
		t.Fatalf("destinataire %q: %v", recipient, err)
	}
	msg, err := domain.NewMessage(domain.ChannelEmail, destinataire, subject, body)
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	return msg
}
