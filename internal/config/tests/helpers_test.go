// Package tests holds the BLACK BOX tests of the config package: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for the unexported identifiers. One file per
// test — the file name says what is verified, without having to open it.
package tests

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// shippedCatalog assembles the catalogue EXACTLY like the composition root.
//
// Loading the shipped configuration against a made-up catalogue would prove
// nothing: what is at stake is the agreement between the files of `config/` and
// the modules actually embedded (ADR 014). If those two diverge, the shipped
// configuration refuses to load — and that is what these tests must catch, not
// a binary at its first startup.
func shippedCatalog(t *testing.T) config.ModuleCatalog {
	t.Helper()
	coreCatalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("catalogue of the core: %v", err)
	}
	catalog, err := config.MergeCatalogs(coreCatalog, userregistration.Catalog())
	if err != nil {
		t.Fatalf("merge of the catalogues: %v", err)
	}
	return catalog
}

// shippedConfigDir points at the config/ directory that is ACTUALLY shipped.
//
// The other tests of this package validate structures built by hand. These ones
// validate the files of the repository: without them, a typo in
// config/modules.yaml would pass `task check` and would only appear at the
// first startup. That is exactly the false green that rules/README.md forbids.
func shippedConfigDir() string { return filepath.Join("..", "..", "..", "config") }

// testEncryptionKey builds a zeroed AES-256 key, encoded at runtime.
//
// No key is written hard-coded in the repository, not even a test one:
// rules/securite.md forbids any versioned secret, and a base64 string of 32
// bytes in a file is indistinguishable from a real leak, for gitleaks as for a
// reader.
func testEncryptionKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

// withShippedConfig points the loader at the shipped configuration, with the
// secret supplied.
func withShippedConfig(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvVarConfigDir, shippedConfigDir())
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", testEncryptionKey())
}

// withCatalogTestConfig prepares a loadable configuration where only the
// `modules:` section is the one of the test.
//
// It copies every shipped group EXCEPT `modules.yaml`, which it replaces.
// Copying rather than rewriting by hand is deliberate: a minimal configuration
// written here would diverge from the real one at the first mandatory setting
// added, and the test would fail for a reason unrelated to what it verifies.
func withCatalogTestConfig(t *testing.T, modules string) {
	t.Helper()

	dir := t.TempDir()
	// The environment layer counts too: it is `env/development.yaml` that gives
	// an explicit default to `DB_DSN`. Forgetting it would make the test fail on
	// a missing secret, that is to say for a reason entirely unrelated to what
	// it verifies.
	if err := os.MkdirAll(filepath.Join(dir, "env"), 0o750); err != nil {
		t.Fatalf("creation of env/: %v", err)
	}
	var copied int
	for _, pattern := range []string{"*.yaml", filepath.Join("env", "*.yaml")} {
		shipped, err := filepath.Glob(filepath.Join(shippedConfigDir(), pattern))
		if err != nil {
			t.Fatalf("reading of the shipped configuration: %v", err)
		}
		for _, path := range shipped {
			relative, err := filepath.Rel(shippedConfigDir(), path)
			if err != nil {
				t.Fatalf("relative path of %s: %v", path, err)
			}
			if filepath.Base(path) == "modules.yaml" || strings.HasPrefix(filepath.Base(path), "local") {
				continue
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading of %s: %v", relative, err)
			}
			if err := os.WriteFile(
				filepath.Join(dir, relative), withoutModules(t, relative, content), 0o600); err != nil {
				t.Fatalf("writing of %s: %v", relative, err)
			}
			copied++
		}
	}
	if copied == 0 {
		t.Fatal("no configuration file copied: the test would verify nothing")
	}
	if err := os.WriteFile(filepath.Join(dir, "modules.yaml"), []byte(modules), 0o600); err != nil {
		t.Fatalf("writing of modules.yaml: %v", err)
	}

	t.Setenv(config.EnvVarConfigDir, dir)
	t.Setenv(config.EnvVarAppEnv, "")
	t.Setenv("SECURITY_ENCRYPTION_KEY", testEncryptionKey())
}

// withoutModules removes the `modules:` section from a copied layer.
//
// # The defect this fixes
//
// The helper promises that "only the `modules:` section is the one of the
// test", and it only replaced `modules.yaml`. Now, an environment layer can
// declare modules too — `config/env/development.yaml` enables `auth`
// (ADR 017) — and that declaration then reached the test, which refused a
// module absent from ITS catalogue.
//
// The failure was right: the configuration named a module that the catalogue of
// this test does not know. It was the helper that lied about what it isolated.
func withoutModules(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var layer map[string]any
	if err := yaml.Unmarshal(content, &layer); err != nil {
		t.Fatalf("invalid YAML in %s: %v", name, err)
	}
	if _, present := layer["modules"]; !present {
		return content
	}

	delete(layer, "modules")
	cleaned, err := yaml.Marshal(layer)
	if err != nil {
		t.Fatalf("reassembling of %s: %v", name, err)
	}
	return cleaned
}

// applicationCatalog is the catalogue an APPLICATION would supply.
//
// `facturation` exists nowhere in the starter: no code, no driver, no line in
// `internal/config`. It only has a catalogue — and that is all ADR 014
// requires.
func applicationCatalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		"facturation": {
			Default: "memory",
			Drivers: map[string]config.Resources{
				"memory": {},
				"sqlite": {SQL: true},
			},
		},
	}
}
