package config

import (
	"errors"
	"strings"
	"testing"
)

// This file carries the tests of the UNEXPORTED identifiers: `expand`,
// `deepMerge` and the `validate()`. They cannot live in
// `internal/config/tests/`, which is another package.
//
// Repository convention (rules/tests.md):
//   - `{package}/tests/`            black box tests, public API only
//   - `{package}/internal_test.go`  tests of internals, next to the code
//
// A test of an internal is an admission of coupling to the implementation: it
// must stay a minority, and disappear if the function becomes exported.

// TestAnEmptyCatalogRefusesEverything: the deny-by-default of ADR 014.
//
// The catalogue comes from the composition root. If it is empty — because
// nobody built it, or because it was forgotten — the configuration must admit
// NOTHING. Forgetting the catalogue must be loud, never permissive: that is
// what distinguishes "no module declared" from "every module accepted".
func TestAnEmptyCatalogRefusesEverything(t *testing.T) {
	t.Parallel()

	mods := Modules{"quelconque": {Enabled: true, Driver: "memory"}}

	problems := mods.validate(ModuleCatalog{})
	if len(problems) == 0 {
		t.Fatal("an empty catalogue must refuse everything")
	}
	if !strings.Contains(problems[0].Error(), "quelconque") {
		t.Errorf("the message must name the refused module: %v", problems[0])
	}
}

// TestADriverAbsentFromTheCatalogIsRefused: a typo in a driver name never
// resolves to "the closest one".
func TestADriverAbsentFromTheCatalogIsRefused(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	mods := Modules{"facturation": {Enabled: true, Driver: "memry"}}

	problems := mods.validate(catalog)
	if len(problems) != 1 {
		t.Fatalf("an unknown driver must produce exactly one refusal, got %d", len(problems))
	}
	if !strings.Contains(problems[0].Error(), "memry") {
		t.Errorf("the message must quote the offending driver: %v", problems[0])
	}
}

// TestADisabledModuleIsNotValidated: one does not refuse to start because of
// the driver of a module nobody has enabled.
//
// Without that, an example configuration left in place — module disabled,
// `postgres` driver — would prevent starting without a database, which would
// flatly contradict ADR 012.
func TestADisabledModuleIsNotValidated(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	mods := Modules{"facturation": {Enabled: false, Driver: "a-driver-that-does-not-exist"}}

	if problems := mods.validate(catalog); len(problems) != 0 {
		t.Errorf("a disabled module must not be validated: %v", problems)
	}
}

// TestResolveDoesNotMutateItsInput: Resolve is a PURE function.
//
// It returns a copy where the defaults are placed. If it modified its input,
// "applying the defaults" would become a hidden effect again — exactly what
// ADR 014 moves out of the accessors.
func TestResolveDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	original := Modules{"facturation": {Enabled: true}}

	resolved := original.Resolve(catalog)

	if got := original["facturation"].Driver; got != "" {
		t.Errorf("Resolve modified its input: driver became %q", got)
	}
	if got := resolved["facturation"].Driver; got != "memory" {
		t.Errorf("the default of the catalogue was not placed: %q", got)
	}
}

// TestMergeCatalogsRefusesACollision: two modules cannot carry the same name.
//
// The failure mode avoided is silent: one of the two would end up configured by
// the drivers of the other, and the first symptom would be an "unknown" driver
// for a module that nonetheless declares it.
func TestMergeCatalogsRefusesACollision(t *testing.T) {
	t.Parallel()

	first := ModuleCatalog{"facturation": {Default: "memory"}}
	second := ModuleCatalog{"facturation": {Default: "postgres"}}

	if _, err := MergeCatalogs(first, second); err == nil {
		t.Fatal("a module name declared twice must be refused")
	}
}

// TestExpandFailsOnMissingRequiredSecret: a missing secret that resolved to the
// empty string would produce an anonymous connection or an encryption with an
// empty key. It must refuse to start.
func TestExpandFailsOnMissingRequiredSecret(t *testing.T) {
	t.Parallel()

	_, err := expand("dsn: ${HEXA_TEST_ABSENT_VAR}")
	if err == nil {
		t.Fatal("a mandatory reference that is not defined must refuse the loading")
	}
	if !strings.Contains(err.Error(), "HEXA_TEST_ABSENT_VAR") {
		t.Errorf("the message must name the missing variable: %v", err)
	}
}

func TestExpandUsesExplicitDefault(t *testing.T) {
	t.Parallel()

	out, err := expand("addr: ${HEXA_TEST_ABSENT_VAR:-localhost:6379}")
	if err != nil {
		t.Fatalf("an explicit default must be accepted: %v", err)
	}
	if !strings.Contains(out, "localhost:6379") {
		t.Errorf("default not applied: %q", out)
	}
}

// TestExpandAcceptsEmptyExplicitDefault: `${VAR:-}` signals an optional
// setting. That is legitimate, unlike a reference without a default.
func TestExpandAcceptsEmptyExplicitDefault(t *testing.T) {
	t.Parallel()

	if _, err := expand("password: ${HEXA_TEST_ABSENT_VAR:-}"); err != nil {
		t.Errorf("an explicit empty default must be accepted: %v", err)
	}
}

// TestExpandFailsOnDefinedButEmptySecret: the secret declared in a deployment
// chain but never injected arrives as an EMPTY string, not as an absent
// variable. That is the most frequent form of the missing secret, and the one
// that would most easily go unnoticed.
//
// This test cannot be parallel: it manipulates the environment of the process.
func TestExpandFailsOnDefinedButEmptySecret(t *testing.T) {
	t.Setenv("HEXA_TEST_EMPTY_VAR", "")

	_, err := expand("key: ${HEXA_TEST_EMPTY_VAR}")
	if err == nil {
		t.Fatal("a variable that is defined but empty must refuse the loading")
	}
	var missing ErrMissingSecret
	if !errors.As(err, &missing) {
		t.Fatalf("want ErrMissingSecret, got %v", err)
	}
}

// TestExpandPrefersDefaultOverEmptyValue: POSIX semantics of `${VAR:-default}`.
// The `:` makes the fallback apply to the empty value as much as to the absence
// — otherwise a variable emptied by accident would overwrite an otherwise valid
// default.
func TestExpandPrefersDefaultOverEmptyValue(t *testing.T) {
	t.Setenv("HEXA_TEST_EMPTY_VAR", "")

	out, err := expand("addr: ${HEXA_TEST_EMPTY_VAR:-localhost:6379}")
	if err != nil {
		t.Fatalf("an explicit default must apply: %v", err)
	}
	if !strings.Contains(out, "localhost:6379") {
		t.Errorf("default not applied in the face of an empty variable: %q", out)
	}
}

func TestExpandReportsAllMissingVariablesAtOnce(t *testing.T) {
	t.Parallel()

	// Fixing your configuration over six restarts is unacceptable: every
	// missing variable is reported at once.
	_, err := expand("a: ${HEXA_TEST_A}\nb: ${HEXA_TEST_B}")
	if err == nil {
		t.Fatal("want a failure")
	}
	for _, name := range []string{"HEXA_TEST_A", "HEXA_TEST_B"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the message must name %s: %v", name, err)
		}
	}
}

// TestDeepMergeOverridesLists: concatenating would silently add back CORS
// origins one believed had been removed.
func TestDeepMergeOverridesLists(t *testing.T) {
	t.Parallel()

	base := map[string]any{"http": map[string]any{"allowed_origins": []any{"http://a"}}}
	deepMerge(base, map[string]any{"http": map[string]any{"allowed_origins": []any{"https://b"}}})

	http, ok := base["http"].(map[string]any)
	if !ok {
		t.Fatal("structure lost during the merge")
	}
	origins, ok := http["allowed_origins"].([]any)
	if !ok {
		t.Fatal("list lost during the merge")
	}
	if len(origins) != 1 || origins[0] != "https://b" {
		t.Errorf("the upper layer must OVERWRITE the list, got: %v", origins)
	}
}

func TestDeepMergeMergesNestedTables(t *testing.T) {
	t.Parallel()

	base := map[string]any{"db": map[string]any{"max_conns": 10, "dsn": "a"}}
	deepMerge(base, map[string]any{"db": map[string]any{"max_conns": 25}})

	db, ok := base["db"].(map[string]any)
	if !ok {
		t.Fatal("structure lost")
	}
	if db["max_conns"] != 25 {
		t.Errorf("max_conns not overridden: %v", db["max_conns"])
	}
	if db["dsn"] != "a" {
		t.Errorf("dsn lost during the merge: %v", db["dsn"])
	}
}

// TestInteropValidateRequiresBaseURLForHTTP: an http transport without an
// address would fail on the first call, in production. It must fail at startup.
func TestInteropValidateRequiresBaseURLForHTTP(t *testing.T) {
	t.Parallel()

	problems := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "http"},
	}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problem(s), want 1 (base_urls missing)", len(problems))
	}
}

func TestInteropValidateRefusesUnknownTransport(t *testing.T) {
	t.Parallel()

	problems := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "grpc"},
	}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problem(s), want 1", len(problems))
	}
}
