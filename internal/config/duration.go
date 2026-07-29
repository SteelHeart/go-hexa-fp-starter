package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a duration readable in a configuration file.
//
// # Why this type exists
//
// `gopkg.in/yaml.v3` does NOT decode a string such as "5s" into a
// time.Duration: it only accepts an integer of nanoseconds. Without this type,
// the whole configuration would fail at loading — and the error message would
// not say why.
//
// The underlying type IS time.Duration, so the conversion at the point of use
// is free: `time.Duration(cfg.HTTP.ReadTimeout)`.
type Duration time.Duration

// UnmarshalYAML accepts the two useful forms:
//   - a string:  "5s", "1h30m", "250ms" — the form expected in conf
//   - an integer: interpreted in SECONDS, never in nanoseconds
//
// Nanoseconds are deliberately refused: `read_timeout: 5` must mean five
// seconds, not five nanoseconds. Interpreting a bare integer as nanoseconds
// would produce a timeout of zero in practice, hence a silent outage.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	// Discriminate on the TAG, not on the success of the decoding: yaml.v3
	// happily decodes an integer node into a string, so trying the string first
	// would make `30` fail instead of reading it as thirty seconds.
	if node.Tag == "!!int" {
		var asSeconds int64
		if err := node.Decode(&asSeconds); err != nil {
			return fmt.Errorf("unreadable integer duration at line %d: %w", node.Line, err)
		}
		*d = Duration(time.Duration(asSeconds) * time.Second)
		return nil
	}

	var asString string
	if err := node.Decode(&asString); err != nil {
		return fmt.Errorf(
			"unreadable duration at line %d (expected a string such as \"5s\"): %w", node.Line, err)
	}
	parsed, err := time.ParseDuration(asString)
	if err != nil {
		return fmt.Errorf("duration %q is unreadable (expected: 5s, 1h30m, 250ms): %w", asString, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalYAML rewrites the duration in its readable form, so that the effective
// configuration can be displayed exactly as it would be written.
func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

// Duration converts to the type of the standard library.
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// String returns the readable form.
func (d Duration) String() string { return time.Duration(d).String() }
