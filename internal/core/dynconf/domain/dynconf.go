// Package domain carries the vocabulary of dynamic configuration, with no
// dependency.
//
// # The dividing line with the config package
//
// What requires a redeployment to change — DSN, ports, timeouts — is in
// `config`. What must be able to change AT RUN TIME is here: feature flags and
// business settings.
//
// Flags are not a luxury: trunk-based development (ADR 007) requires that
// incomplete work go behind a flag rather than behind a long-lived branch.
// Without this building block, the rule is untenable.
package domain

import (
	"errors"
	"fmt"
	"strconv"
)

// Kind separates the two natures of dynamic value.
//
// Separated and not merged into a single namespace: a flag is read as a boolean
// with a fallback on `false`, a setting is read as text with a possible
// absence. Confusing the two would make an empty setting look like an inactive
// flag.
type Kind string

const (
	// KindFlag denotes a feature flag.
	KindFlag Kind = "flag"
	// KindSetting denotes a business setting.
	KindSetting Kind = "setting"
)

// FlagKey identifies a feature flag.
type FlagKey string

// SettingKey identifies a business setting.
type SettingKey string

// Setting is the result of reading a setting.
//
// `Found` distinguishes "absent" from "present and empty" — two situations
// nothing must confuse: the first calls for a default value on the caller's
// side, the second is an operations decision.
type Setting struct {
	Value string
	Found bool
}

// ErrReadOnly signals a store that does not accept writes.
//
// The `file` driver reads VERSIONED values: rewriting them at run time would
// produce a divergence between the repository and what is running, invisible
// until the next deployment overwrote them.
var ErrReadOnly = errors.New("dynamic configuration store is read-only")

// ErrInvalidChange refuses a write that is incomplete or of an unknown nature.
var ErrInvalidChange = errors.New("invalid dynamic configuration change")

// Change is a write of a dynamic value.
type Change struct {
	Kind  Kind
	Key   string
	Value string
}

// IsValid says whether the change is usable.
//
// The value may be empty — that is a legitimate value. The key and the nature
// are not.
func (c Change) IsValid() bool {
	return c.Key != "" && (c.Kind == KindFlag || c.Kind == KindSetting)
}

// Qualify returns the full identifier of a value, nature included.
//
// Used as a cache key and as a lookup key by every driver: two drivers that did
// not qualify in the same way would not be substitutable.
func Qualify(kind Kind, key string) string { return string(kind) + "." + key }

// ParseFlag interprets a textual value as a boolean.
//
// # Deny by default
//
// An unreadable value gives `false`. A flag that switched itself on for a value
// we do not understand would be exactly the opposite of what we want: the
// incomplete feature it hides would go to production on a typo.
//
// This is the only place where this interpretation exists, so that every driver
// answers identically to `"1"`, `"true"` or `"TRUE"`.
func ParseFlag(raw string) bool {
	enabled, err := strconv.ParseBool(raw)
	return err == nil && enabled
}

// FormatFlag returns the canonical form of a flag, for writing.
func FormatFlag(enabled bool) string { return strconv.FormatBool(enabled) }

// Describe returns a change readable in an error message.
func (c Change) Describe() string { return fmt.Sprintf("%s.%s", c.Kind, c.Key) }
