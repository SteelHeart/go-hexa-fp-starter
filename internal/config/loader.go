package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// The ONLY TWO environment variables the starter reads to decide what to load.
// Everything else is in the files of config/; secrets are referenced there by
// ${VAR}.
const (
	EnvVarAppEnv    = "APP_ENV"
	EnvVarConfigDir = "CONFIG_DIR"
)

// DefaultConfigDir is the default configuration directory.
//
// The files are read from DISK and not embedded: operations must be able to
// read them, compare them and fix them without rebuilding an image. The
// Dockerfile copies them into the image; a mounted volume overrides them.
const DefaultConfigDir = "config"

// placeholder captures ${VAR} and ${VAR:-default}.
var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// ErrMissingSecret reports unresolved ${VAR} references that have no default.
//
// Refusing to start is deliberate: a missing secret that resolved to the empty
// string would produce an anonymous connection or an encryption with an empty
// key — a silent failure, hence the worst kind.
//
// "Unresolved" covers the ABSENT variable and the DEFINED BUT EMPTY variable:
// the second is the usual symptom of a secret forgotten in a deployment chain,
// and it must refuse just as hard as the first.
type ErrMissingSecret struct{ Variables []string }

func (e ErrMissingSecret) Error() string {
	return "environment variables required by the configuration and not defined: " +
		strings.Join(e.Variables, ", ")
}

// Dir resolves the configuration directory.
func Dir() string {
	if dir := os.Getenv(EnvVarConfigDir); dir != "" {
		return dir
	}
	return DefaultConfigDir
}

// Load reads, merges, resolves and validates the configuration.
//
// Order of increasing priority:
//
//  1. config/*.yaml           groups — default values, versioned
//  2. config/env/{env}.yaml   per-environment overrides, versioned
//  3. config/local.yaml       developer overrides — NOT versioned
//  4. ${VAR} in the values    secrets, from the runtime environment
//
// Secrets are in NO file: only referenced.
// Load reads the configuration and validates it AGAINST THE CATALOGUE it
// receives.
//
// The catalogue comes from the composition root, which merges the one of every
// mounted module — core as well as business (ADR 014). There is no module table
// in this package: what is not mounted is not configurable.
//
// Passing an empty catalogue makes every module declaration be refused. That is
// the intended behaviour, and it is what makes forgetting the catalogue loud
// rather than permissive.
func Load(catalog ModuleCatalog) (Config, error) {
	dir := Dir()
	env := os.Getenv(EnvVarAppEnv)
	if env == "" {
		env = string(EnvDevelopment)
	}

	merged, err := mergeLayers(dir, env)
	if err != nil {
		return Config{}, err
	}

	raw, err := yaml.Marshal(merged)
	if err != nil {
		return Config{}, fmt.Errorf("reassembling the configuration: %w", err)
	}
	resolved, err := expand(string(raw))
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	// KnownFields refuses an unknown key: a typo in a file would otherwise be
	// silent, and the setting would simply have no effect.
	decoder := yaml.NewDecoder(strings.NewReader(resolved))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decoding the configuration (%s): %w", dir, err)
	}

	cfg.applyDefaults()
	// Driver defaults are resolved BEFORE validation, so that the error message
	// names the driver actually retained and not an empty string.
	cfg.Modules = cfg.Modules.Resolve(catalog)
	if err := cfg.validate(catalog); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func mergeLayers(dir, env string) (map[string]any, error) {
	root := os.DirFS(dir)
	groups, err := fs.Glob(root, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf(
			"no configuration file in %q (set %s?)", dir, EnvVarConfigDir)
	}
	sort.Strings(groups)

	merged := map[string]any{}
	for _, name := range groups {
		// local.yaml is applied last, after the environment layer.
		if filepath.Base(name) == "local.yaml" {
			continue
		}
		if err := mergeFile(merged, root, name); err != nil {
			return nil, err
		}
	}
	for _, name := range []string{"env/" + env + ".yaml", "local.yaml"} {
		if err := mergeFile(merged, root, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return merged, nil
}

func mergeFile(dst map[string]any, root fs.FS, name string) error {
	content, err := fs.ReadFile(root, name)
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}
	var layer map[string]any
	if err := yaml.Unmarshal(content, &layer); err != nil {
		return fmt.Errorf("invalid YAML in %s: %w", name, err)
	}
	deepMerge(dst, layer)
	return nil
}

// deepMerge merges recursively. A scalar value or a list overwrites; only
// tables merge.
//
// Lists overwrite DELIBERATELY: concatenating `allowed_origins` between layers
// would silently add back origins one believed had been removed.
func deepMerge(dst, src map[string]any) {
	for key, value := range src {
		nested, isMap := value.(map[string]any)
		if !isMap {
			dst[key] = value
			continue
		}
		existing, _ := dst[key].(map[string]any)
		clone := map[string]any{}
		maps.Copy(clone, existing)
		deepMerge(clone, nested)
		dst[key] = clone
	}
}

// expand resolves the ${VAR} and ${VAR:-default} references.
func expand(raw string) (string, error) {
	seen := map[string]struct{}{}
	out := placeholder.ReplaceAllStringFunc(raw, func(match string) string {
		groups := placeholder.FindStringSubmatch(match)
		name, fallback := groups[1], groups[2]
		optional := strings.Contains(match, ":-")

		// A variable DEFINED BUT EMPTY counts as "absent". That is the most
		// common case in the real world: a secret declared in the CI or in an
		// orchestrator but never injected arrives as an empty string, not as an
		// absent variable. Letting it through would produce an anonymous
		// connection or an encryption with an empty key — the silent failure
		// ErrMissingSecret exists to prevent.
		//
		// It is also the semantics of `${VAR:-default}` in a POSIX shell: the
		// `:` makes the fallback apply to the empty value as much as to the
		// absence.
		if value, found := os.LookupEnv(name); found && value != "" {
			return value
		}
		// An explicit default, even an empty one (`${VAR:-}`), is legitimate: it
		// signals an optional setting. A reference WITHOUT a default is
		// mandatory.
		if optional {
			return fallback
		}
		seen[name] = struct{}{}
		return ""
	})
	if len(seen) > 0 {
		missing := make([]string, 0, len(seen))
		for name := range seen {
			missing = append(missing, name)
		}
		sort.Strings(missing)
		return "", ErrMissingSecret{Variables: missing}
	}
	return out, nil
}
