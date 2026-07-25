package config

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestDurationUnmarshalAcceptsHumanForms verrouille le défaut qui rendait toute
// la configuration inutilisable : yaml.v3 ne décode pas "5s" en time.Duration.
func TestDurationUnmarshalAcceptsHumanForms(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		yaml string
		want time.Duration
	}{
		"secondes":            {yaml: `d: 5s`, want: 5 * time.Second},
		"minutes et secondes": {yaml: `d: 1h30m`, want: 90 * time.Minute},
		"millisecondes":       {yaml: `d: 250ms`, want: 250 * time.Millisecond},
		"zero explicite":      {yaml: `d: 0s`, want: 0},
		"entier lu en secondes, jamais en nanosecondes": {yaml: `d: 30`, want: 30 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out struct{ D Duration }
			if err := yaml.Unmarshal([]byte(tc.yaml), &out); err != nil {
				t.Fatalf("décodage de %q: %v", tc.yaml, err)
			}
			if out.D.Duration() != tc.want {
				t.Errorf("%q → %v, attendu %v", tc.yaml, out.D.Duration(), tc.want)
			}
		})
	}
}

func TestDurationUnmarshalRefusesGarbage(t *testing.T) {
	t.Parallel()

	// Une durée illisible doit ÉCHOUER, jamais valoir zéro : un délai à zéro
	// n'est pas une valeur sûre, c'est une panne silencieuse.
	for _, raw := range []string{`d: cinq secondes`, `d: 5 secondes`, `d: [1,2]`} {
		var out struct{ D Duration }
		if err := yaml.Unmarshal([]byte(raw), &out); err == nil {
			t.Errorf("%q accepté alors qu'il devrait échouer (valeur obtenue: %v)", raw, out.D)
		}
	}
}

func TestModulesGetDefaultsToDisabledWithZeroDependencyDriver(t *testing.T) {
	t.Parallel()

	// Deny par défaut : un module absent de la configuration est DÉSACTIVÉ.
	// On n'active jamais une capacité que personne n'a demandée.
	modules := Modules{}
	got := modules.Get("outbox")
	if got.Enabled {
		t.Error("un module absent de la configuration doit être désactivé")
	}
	if got.Driver != "memory" {
		t.Errorf("pilote par défaut = %q, attendu \"memory\" (sans dépendance externe)", got.Driver)
	}
}

func TestModulesGetFillsMissingDriverWithZeroDependencyDefault(t *testing.T) {
	t.Parallel()

	modules := Modules{"audit": {Enabled: true}}
	if driver := modules.DriverOf("audit"); driver != "log" {
		t.Errorf("pilote de audit = %q, attendu \"log\"", driver)
	}
}

// TestModulesRequiresDatabase verrouille la promesse centrale de l'ADR 012 :
// avec les pilotes par défaut, aucune base n'est nécessaire.
func TestModulesRequiresDatabase(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		modules Modules
		want    bool
	}{
		"pilotes par defaut, aucune base requise": {
			modules: Modules{
				"outbox":      {Enabled: true},
				"idempotency": {Enabled: true},
				"audit":       {Enabled: true},
				"dynconf":     {Enabled: true},
				"storage":     {Enabled: true},
			},
			want: false,
		},
		"un seul pilote postgres suffit a exiger une base": {
			modules: Modules{
				"outbox": {Enabled: true, Driver: "postgres"},
			},
			want: true,
		},
		"un pilote postgres sur un module DESACTIVE n'exige rien": {
			modules: Modules{
				"outbox": {Enabled: false, Driver: "postgres"},
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.modules.RequiresDatabase(); got != tc.want {
				t.Errorf("RequiresDatabase() = %v, attendu %v", got, tc.want)
			}
		})
	}
}

func TestModulesValidateRefusesUnknownDriver(t *testing.T) {
	t.Parallel()

	// Deny par défaut jusque dans la lecture de la configuration : une faute de
	// frappe ne doit jamais se résoudre en « le pilote le plus proche ».
	problems := Modules{"outbox": {Enabled: true, Driver: "postgresql"}}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s) signalé(s), attendu 1", len(problems))
	}
	if !strings.Contains(problems[0].Error(), "postgresql") {
		t.Errorf("le message doit nommer le pilote fautif, obtenu: %v", problems[0])
	}
}

func TestModulesValidateRefusesUnknownModule(t *testing.T) {
	t.Parallel()

	problems := Modules{"blockchain": {Enabled: true}}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s) signalé(s), attendu 1", len(problems))
	}
}

func TestModulesValidateIgnoresDisabledModules(t *testing.T) {
	t.Parallel()

	// Un module désactivé n'a pas à désigner un pilote valide : on ne bloque pas
	// un démarrage sur la configuration d'une capacité qu'on n'utilise pas.
	if problems := (Modules{"outbox": {Enabled: false, Driver: "n-importe-quoi"}}).validate(); len(problems) != 0 {
		t.Errorf("un module désactivé ne doit rien signaler, obtenu: %v", problems)
	}
}

func TestInteropTransportForFallsBackToInproc(t *testing.T) {
	t.Parallel()

	if got := (Interop{}).TransportFor("user_registration"); got != "inproc" {
		t.Errorf("transport par défaut = %q, attendu \"inproc\"", got)
	}

	interop := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "http"},
	}
	if got := interop.TransportFor("billing"); got != "http" {
		t.Errorf("surcharge par module non appliquée: %q", got)
	}
	if got := interop.TransportFor("autre"); got != "inproc" {
		t.Errorf("module sans surcharge = %q, attendu \"inproc\"", got)
	}
}

func TestInteropValidateRequiresBaseURLForHTTP(t *testing.T) {
	t.Parallel()

	// Un transport http sans adresse échouerait au premier appel, en production.
	// Il doit échouer au démarrage.
	problems := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "http"},
	}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s), attendu 1 (base_urls manquant)", len(problems))
	}
}

func TestExpandFailsOnMissingRequiredSecret(t *testing.T) {
	t.Parallel()

	// Un secret manquant qui se résoudrait en chaîne vide produirait une
	// connexion anonyme ou un chiffrement avec une clé vide.
	if _, err := expand("dsn: ${HEXA_TEST_ABSENT_VAR}"); err == nil {
		t.Fatal("une référence obligatoire non definie doit refuser le chargement")
	}
}

func TestExpandUsesExplicitDefault(t *testing.T) {
	t.Parallel()

	out, err := expand("addr: ${HEXA_TEST_ABSENT_VAR:-localhost:6379}")
	if err != nil {
		t.Fatalf("un défaut explicite doit être accepté: %v", err)
	}
	if !strings.Contains(out, "localhost:6379") {
		t.Errorf("défaut non applique: %q", out)
	}
}

func TestExpandAcceptsEmptyExplicitDefault(t *testing.T) {
	t.Parallel()

	// `${VAR:-}` signale un réglage optionnel : c'est légitime, contrairement à
	// une référence sans défaut.
	if _, err := expand("password: ${HEXA_TEST_ABSENT_VAR:-}"); err != nil {
		t.Errorf("un défaut vide explicite doit être accepté: %v", err)
	}
}

func TestDeepMergeOverridesListsInsteadOfConcatenating(t *testing.T) {
	t.Parallel()

	// Concaténer ajouterait silencieusement des origines qu'on croyait retirées.
	base := map[string]any{"http": map[string]any{"allowed_origins": []any{"http://a"}}}
	deepMerge(base, map[string]any{"http": map[string]any{"allowed_origins": []any{"https://b"}}})

	origins := base["http"].(map[string]any)["allowed_origins"].([]any)
	if len(origins) != 1 || origins[0] != "https://b" {
		t.Errorf("la couche supérieure doit ÉCRASER la liste, obtenu: %v", origins)
	}
}

func TestDeepMergeMergesNestedTables(t *testing.T) {
	t.Parallel()

	base := map[string]any{"db": map[string]any{"max_conns": 10, "dsn": "a"}}
	deepMerge(base, map[string]any{"db": map[string]any{"max_conns": 25}})

	db := base["db"].(map[string]any)
	if db["max_conns"] != 25 {
		t.Errorf("max_conns non surchargé: %v", db["max_conns"])
	}
	if db["dsn"] != "a" {
		t.Errorf("dsn perdu lors de la fusion: %v", db["dsn"])
	}
}
