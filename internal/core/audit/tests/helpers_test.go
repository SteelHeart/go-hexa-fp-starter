// Package tests holds the BLACK BOX tests of the audit module: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// recordedAt is the injected instant. No test calls time.Now: an audit log must
// be reproducible to the second.
func recordedAt() time.Time {
	return time.Date(2026, time.July, 25, 12, 30, 0, 0, time.UTC)
}

// newLogModule builds the module on its default driver and returns the buffer
// the log is written to, so that it can be inspected.
func newLogModule(t *testing.T) (audit.Module, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	mod, err := audit.New(
		config.Module{Enabled: true, Driver: "log"},
		audit.Deps{Logger: logger, Now: recordedAt},
	)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod, &buf
}

// completeEntry is a valid entry, to be altered field by field in the tests.
func completeEntry() domain.Entry {
	return domain.Entry{
		Actor:      "user-42",
		Action:     "user.registered",
		EntityType: "user",
		EntityID:   "42",
		Metadata:   map[string]any{"surface": "http"},
	}
}

// decodeLogLine reads the line written by the log driver.
func decodeLogLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("unreadable log (%q): %v", buf.String(), err)
	}
	return line
}
