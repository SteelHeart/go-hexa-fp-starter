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

// Les DEUX seules variables d'environnement que le socle lit pour décider quoi
// charger. Tout le reste est dans les fichiers de config/ ; les secrets y sont
// référencés par ${VAR}.
const (
	EnvVarAppEnv    = "APP_ENV"
	EnvVarConfigDir = "CONFIG_DIR"
)

// DefaultConfigDir est le répertoire de configuration par défaut.
//
// Les fichiers sont lus sur DISQUE et non embarqués : l'exploitation doit
// pouvoir les lire, les comparer et les corriger sans reconstruire une image.
// Le Dockerfile les copie dans l'image ; un volume monté les surcharge.
const DefaultConfigDir = "config"

// placeholder capture ${VAR} et ${VAR:-défaut}.
var placeholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// ErrMissingSecret signale des références ${VAR} non résolues et sans défaut.
//
// Refuser le démarrage est délibéré : un secret manquant qui se résoudrait en
// chaîne vide produit une connexion anonyme ou un chiffrement avec une clé
// vide — un échec silencieux, donc le pire.
//
// « Non résolue » couvre la variable ABSENTE et la variable DÉFINIE MAIS VIDE :
// la seconde est le symptôme habituel d'un secret oublié dans une chaîne de
// déploiement, et elle doit refuser aussi fort que la première.
type ErrMissingSecret struct{ Variables []string }

func (e ErrMissingSecret) Error() string {
	return "variables d'environnement requises par la configuration et non définies: " +
		strings.Join(e.Variables, ", ")
}

// Dir résout le répertoire de configuration.
func Dir() string {
	if dir := os.Getenv(EnvVarConfigDir); dir != "" {
		return dir
	}
	return DefaultConfigDir
}

// Load lit, fusionne, résout et valide la configuration.
//
// Ordre de priorité croissant :
//
//  1. config/*.yaml           groupes — valeurs par défaut, versionnées
//  2. config/env/{env}.yaml   surcharges par environnement, versionnées
//  3. config/local.yaml       surcharges du développeur — NON versionné
//  4. ${VAR} dans les valeurs secrets, depuis l'environnement d'exécution
//
// Les secrets ne sont dans AUCUN fichier : seulement référencés.
// Load lit la configuration et la valide CONTRE LE CATALOGUE reçu.
//
// Le catalogue vient du composition root, qui fusionne celui de chaque module
// monté — noyau comme métier (ADR 014). Il n'y a aucune table de modules dans ce
// paquet : ce qui n'est pas monté n'est pas configurable.
//
// Passer un catalogue vide fait refuser toute déclaration de module. C'est le
// comportement voulu, et c'est ce qui rend l'oubli du catalogue bruyant plutôt
// que permissif.
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
		return Config{}, fmt.Errorf("réassemblage de la configuration: %w", err)
	}
	resolved, err := expand(string(raw))
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	// KnownFields refuse une clé inconnue : une faute de frappe dans un fichier
	// serait autrement silencieuse, et le réglage n'aurait simplement aucun effet.
	decoder := yaml.NewDecoder(strings.NewReader(resolved))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("décodage de la configuration (%s): %w", dir, err)
	}

	cfg.applyDefaults()
	// Les défauts de pilote sont résolus AVANT la validation, pour que le message
	// d'erreur nomme le pilote réellement retenu et non une chaîne vide.
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
		return nil, fmt.Errorf("lecture de %s: %w", dir, err)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf(
			"aucun fichier de configuration dans %q (définir %s ?)", dir, EnvVarConfigDir)
	}
	sort.Strings(groups)

	merged := map[string]any{}
	for _, name := range groups {
		// local.yaml est appliqué en dernier, après la couche d'environnement.
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
		return fmt.Errorf("lecture de %s: %w", name, err)
	}
	var layer map[string]any
	if err := yaml.Unmarshal(content, &layer); err != nil {
		return fmt.Errorf("YAML invalide dans %s: %w", name, err)
	}
	deepMerge(dst, layer)
	return nil
}

// deepMerge fusionne récursivement. Une valeur scalaire ou une liste écrase ;
// seules les tables fusionnent.
//
// Les listes écrasent VOLONTAIREMENT : concaténer `allowed_origins` entre
// couches ajouterait silencieusement des origines qu'on croyait avoir retirées.
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

// expand résout les références ${VAR} et ${VAR:-défaut}.
func expand(raw string) (string, error) {
	seen := map[string]struct{}{}
	out := placeholder.ReplaceAllStringFunc(raw, func(match string) string {
		groups := placeholder.FindStringSubmatch(match)
		name, fallback := groups[1], groups[2]
		optional := strings.Contains(match, ":-")

		// Une variable DÉFINIE MAIS VIDE vaut « absente ». C'est le cas le plus
		// courant en vrai : un secret déclaré dans la CI ou dans un orchestrateur
		// mais jamais injecté arrive comme chaîne vide, pas comme variable absente.
		// Le laisser passer produirait une connexion anonyme ou un chiffrement avec
		// une clé vide — l'échec silencieux que ErrMissingSecret existe pour
		// empêcher.
		//
		// C'est aussi la sémantique de `${VAR:-défaut}` dans un shell POSIX : le
		// `:` fait porter le repli sur le vide autant que sur l'absence.
		if value, found := os.LookupEnv(name); found && value != "" {
			return value
		}
		// Un défaut explicite, même vide (`${VAR:-}`), est légitime : il signale un
		// réglage optionnel. Une référence SANS défaut est obligatoire.
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
