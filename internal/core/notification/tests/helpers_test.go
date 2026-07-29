// Package tests contains the BLACK BOX tests of the notification module: they
// only use the public API, exactly like a caller.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is checked, without having to open it.
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
	// recipient is the address used by the tests.
	recipient = "alice@example.com"

	// subject and body are a minimal valid message.
	subject = "Bienvenue"
	body    = "Confirmez votre compte : https://exemple.test/c/JETON-SECRET-123"

	// secretInBody is what one looks for in the logs. A confirmation link IS a
	// bearer credential.
	secretInBody = "JETON-SECRET-123"
)

// logCapture captures what the driver writes.
//
// The RAW text is kept: a leak is looked for in the text, not in a map of
// fields. A secret carried by an innocuously named field would escape any
// field-by-field inspection.
type logCapture struct {
	buffer *bytes.Buffer
	logger *slog.Logger
}

// newLogCapture builds an inspectable log.
func newLogCapture() *logCapture {
	buffer := &bytes.Buffer{}
	return &logCapture{
		buffer: buffer,
		logger: slog.New(slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

// text returns everything that has been written.
func (c *logCapture) text() string { return c.buffer.String() }

// newModule builds the module and its log.
func newModule(t *testing.T, options map[string]any) (notification.Module, *logCapture) {
	t.Helper()

	logs := newLogCapture()
	mod, err := notification.New(
		config.Module{Enabled: true, Driver: "log", Options: options},
		notification.Deps{Logger: logs.logger},
	)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod, logs
}

// configModule forges a module configuration enabled on the `log` driver.
func configModule(options map[string]any) config.Module {
	return config.Module{Enabled: true, Driver: "log", Options: options}
}

// message forges a valid message towards the tests' recipient.
//
// The recipient is NOT a parameter: it is constant across this whole package,
// and a parameter that never varies makes one believe it does. Tests that
// exercise other addresses go through `domain.NewRecipient` directly, there
// where the address IS the subject.
func message(t *testing.T) domain.Message {
	t.Helper()

	to, err := domain.NewRecipient(recipient)
	if err != nil {
		t.Fatalf("recipient %q: %v", recipient, err)
	}
	msg, err := domain.NewMessage(domain.ChannelEmail, to, subject, body)
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	return msg
}
