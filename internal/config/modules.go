package config

import (
	"fmt"
	"slices"
	"time"
)

// Modules carries the enabling and the driver choice of each core module.
//
// This is the file that makes "hexa new then go run, and it starts" true: the
// default drivers have no external dependency
// ([ADR 012](documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
type Modules map[string]Module

// Module carries the configuration of a core module.
type Module struct {
	// Enabled set to false disables the module: its ports return an explicit
	// error rather than falling back on an inert behaviour. A disabled module
	// that "works anyway" is a trap.
	Enabled bool `yaml:"enabled"`
	// Driver names the driver. An unknown driver refuses to start.
	Driver string `yaml:"driver"`
	// Options carries the settings specific to the driver.
	//
	// The VALUES stay untyped — the catalogue of drivers evolves faster than
	// the starter, and each driver interprets its own at its construction. The
	// KEYS, for their part, are declared by the module in its `catalog.go`, and
	// any unknown key refuses to start (#93).
	//
	// The nuance is the whole subject: without it, a typo returned the default
	// value, without a word.
	Options map[string]any `yaml:"options"`
}

// DurationOption reads a duration option of the driver.
//
// Options are not typed when the file is read — the catalogue of drivers
// evolves faster than the starter. This accessor is therefore the only place
// where an option duration is interpreted, so that every driver accepts exactly
// the same spelling: `"24h"` or an integer of seconds, like the Duration type
// of the typed fields.
//
// A value that is present but unreadable refuses to start: silently falling
// back on the default value would give a surprise TTL.
func (m Module) DurationOption(key string, fallback time.Duration) (time.Duration, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}

	var parsed time.Duration
	switch value := raw.(type) {
	case string:
		var err error
		if parsed, err = time.ParseDuration(value); err != nil {
			return 0, fmt.Errorf("options.%s=%q is not a duration (e.g. \"24h\"): %w", key, value, err)
		}
	case int:
		parsed = time.Duration(value) * time.Second
	case int64:
		parsed = time.Duration(value) * time.Second
	default:
		return 0, fmt.Errorf("options.%s must be a duration or an integer of seconds, got %T", key, raw)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("options.%s must be strictly positive, got %v", key, parsed)
	}
	return parsed, nil
}

// IntOption reads an integer option of the driver.
//
// Refuses zero and negative values: no integer option of the starter makes
// sense at zero — a batch of zero messages, or zero allowed attempts, describe
// a component that does nothing, silently.
func (m Module) IntOption(key string, fallback int) (int, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}

	var value int
	switch typed := raw.(type) {
	case int:
		value = typed
	case int64:
		value = int(typed)
	default:
		return 0, fmt.Errorf("options.%s must be an integer, got %T", key, raw)
	}

	if value <= 0 {
		return 0, fmt.Errorf("options.%s must be strictly positive, got %d", key, value)
	}
	return value, nil
}

// MapOption reads a nested group of options.
//
// An absent group returns an empty table and not an error: declaring nothing is
// a valid configuration. It is a value of the WRONG type that is refused,
// because it betrays a typo.
func (m Module) MapOption(key string) (map[string]any, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return map[string]any{}, nil
	}
	nested, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("options.%s must be a table of values, got %T", key, raw)
	}
	return nested, nil
}

// StringOption reads a textual option of the driver.
//
// A value that is present but empty is refused: it betrays an unsubstituted
// environment variable, not an intention.
func (m Module) StringOption(key, fallback string) (string, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("options.%s must be a string, got %T", key, raw)
	}
	if text == "" {
		return "", fmt.Errorf("options.%s is present but empty", key)
	}
	return text, nil
}

// Get returns the configuration of a module.
//
// A module absent from the configuration is considered DISABLED: one never
// enables a capability that nobody asked for.
//
// The driver returned is the one WRITTEN in the value. The default of the
// catalogue is placed there by Resolve, which Load calls — so that this
// accessor never lies about what it received.
func (m Modules) Get(name string) Module {
	return m[name]
}

// Resolve returns a copy where every active module carries an explicit driver.
//
// It is a PURE function: it does not modify its input. It exists so that
// "applying the defaults" is a visible step, instead of a hidden effect inside
// an accessor — the exact reproach one can make to a dependency injection
// container (ADR 004).
func (m Modules) Resolve(catalog ModuleCatalog) Modules {
	resolved := make(Modules, len(m))
	for name, mod := range m {
		if mod.Driver == "" {
			mod.Driver = catalog.DefaultDriver(name)
		}
		resolved[name] = mod
	}
	// The modules of the catalogue that the configuration does NOT mention also
	// receive their default driver, disabled.
	//
	// Without that, a module absent from the file would return an empty driver,
	// and the composition root — which mounts `outbox` unconditionally — would
	// build it with the empty string, and would therefore fail on "unknown
	// driver". In other words: a MINIMAL configuration would stop starting,
	// which would flatly contradict the promise of ADR 012.
	//
	// They stay disabled: carrying a driver enables nothing.
	for name, set := range catalog {
		if _, declared := resolved[name]; declared {
			continue
		}
		resolved[name] = Module{Enabled: false, Driver: set.Default}
	}
	return resolved
}

// IsEnabled tells whether a module is active.
func (m Modules) IsEnabled(name string) bool { return m.Get(name).Enabled }

// DriverOf returns the driver retained for a module.
func (m Modules) DriverOf(name string) string { return m.Get(name).Driver }

// RequiresSQL tells whether the configuration requires a database — without
// presuming the ENGINE.
//
// This is what allows a binary to open a connection only if it needs one, and
// therefore to start without a database when every active driver lives in
// memory or on file (ADR 012).
//
// The catalogue is a PARAMETER, not a hidden state: the answer depends on the
// mounted modules, and it is the composition root that knows them (ADR 014). A
// caller who forgets to pass it gets `false`, not an arbitrary default — no
// module mounted, no resource required.
func (m Modules) RequiresSQL(catalog ModuleCatalog) bool {
	return m.requires(catalog, func(r Resources) bool { return r.SQL })
}

// RequiresCache tells whether the configuration requires a network cache.
func (m Modules) RequiresCache(catalog ModuleCatalog) bool {
	return m.requires(catalog, func(r Resources) bool { return r.Cache })
}

// requires factors out the two questions above.
//
// It only iterates over the DECLARED modules: a module absent from the
// configuration is disabled (see Get), so it requires nothing.
func (m Modules) requires(catalog ModuleCatalog, wanted func(Resources) bool) bool {
	for name := range m {
		if !m.IsEnabled(name) {
			continue
		}
		if wanted(catalog.Requires(name, m.DriverOf(name))) {
			return true
		}
	}
	return false
}

// validate checks that every active module designates a driver known to the
// CATALOGUE.
//
// An empty catalogue refuses everything, and that is intended: what is not
// mounted is not configurable. One does not configure what one has not plugged
// in (ADR 014).
func (m Modules) validate(catalog ModuleCatalog) []error {
	var problems []error
	for name, mod := range m {
		allowed := catalog.AllowedDrivers(name)
		if allowed == nil {
			problems = append(problems, fmt.Errorf(
				"modules.%s: unknown module — no module of that name is mounted by the composition root", name))
			continue
		}
		if !mod.Enabled {
			continue
		}
		driver := mod.Driver
		if driver == "" {
			driver = catalog.DefaultDriver(name)
		}
		if !slices.Contains(allowed, driver) {
			problems = append(problems, fmt.Errorf(
				"modules.%s.driver=%q unknown (expected: %s)", name, driver, join(allowed)))
			continue
		}
		problems = append(problems, mod.unknownOptions(name, driver, catalog)...)
	}
	return problems
}

// unknownOptions refuses any option key that the driver does not read.
//
// # The hole this function closes
//
// Deny-by-default stopped at the name of the driver. Its OPTIONS went through
// unchecked, because the accessors — `IntOption`, `DurationOption`… — return
// the default value when the key is absent. Correct taken in isolation: an
// absent option IS valid. But nothing enumerated the known keys, so "absent"
// and "misspelt" were indistinguishable.
//
// Measured (#93): `bath_size` instead of `batch_size` let the server start,
// mount the module, and say nothing about it. The dispatcher ran with a setting
// nobody had asked for — and on `max_attempts` or `base_backoff`, the
// discrepancy would have stayed invisible and lasting.
//
// The admitted keys are declared by the module itself, in its `catalog.go`,
// next to the code that reads them and sharing its constants (ADR 014). No file
// of the framework names them.
func (m Module) unknownOptions(name, driver string, catalog ModuleCatalog) []error {
	allowed := catalog.AllowedOptions(name, driver)

	// Sort the offending keys: without that, the map order would make the
	// message vary from one run to the next, and two traces would stop being
	// comparable.
	var offending []string
	for key := range m.Options {
		if !slices.Contains(allowed, key) {
			offending = append(offending, key)
		}
	}
	slices.Sort(offending)

	// A driver with no admitted option at all deserves its own sentence:
	// "expected: " followed by nothing reads like a truncated message, and
	// sends people looking for a defect in the tool rather than for the typo.
	expected := "this driver accepts no option"
	if len(allowed) > 0 {
		expected = "expected: " + join(allowed)
	}

	problems := make([]error, 0, len(offending))
	for _, key := range offending {
		problems = append(problems, fmt.Errorf(
			"modules.%s.options.%s unknown to driver %q (%s)", name, key, driver, expected))
	}
	return problems
}

// Interop carries the communication modes BETWEEN modules.
//
// Distinct from Modules: here one decides how two modules talk to each other,
// not how a module implements itself. A module NEVER accesses the tables of
// another one (ADR 011).
type Interop struct {
	DefaultTransport string            `yaml:"default_transport"`
	CallTimeout      Duration          `yaml:"call_timeout"`
	Transports       map[string]string `yaml:"transports"`
	BaseURLs         map[string]string `yaml:"base_urls"`
}

// Communication modes between modules.
//
// Homonymy assumed with relayInproc: here "inproc" qualifies a direct CALL
// between two modules of the same binary, there an event relay.
const (
	transportInproc   = "inproc"
	transportHTTP     = "http"
	transportEvent    = "event"
	transportDisabled = "disabled"
)

// TransportFor resolves the mode applicable to a module.
func (i Interop) TransportFor(module string) string {
	if raw, found := i.Transports[module]; found && raw != "" {
		return raw
	}
	if i.DefaultTransport == "" {
		return transportInproc
	}
	return i.DefaultTransport
}

// validate checks the coherence of the communication modes.
func (i Interop) validate() []error {
	var problems []error
	allowed := []string{transportInproc, transportHTTP, transportEvent, transportDisabled}
	if !slices.Contains(allowed, i.TransportFor("")) {
		problems = append(problems, fmt.Errorf(
			"interop.default_transport=%q unknown (expected: %s)", i.DefaultTransport, join(allowed)))
	}
	for module, mode := range i.Transports {
		if !slices.Contains(allowed, mode) {
			problems = append(problems, fmt.Errorf(
				"interop.transports.%s=%q unknown (expected: %s)", module, mode, join(allowed)))
			continue
		}
		if mode == "http" && i.BaseURLs[module] == "" {
			problems = append(problems, fmt.Errorf(
				"interop.base_urls.%s is required when the transport is http", module))
		}
	}
	return problems
}
