package config

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// Ce fichier porte les tests des identifiants NON EXPORTÉS : `expand`,
// `deepMerge` et les `validate()`. Ils ne peuvent pas vivre dans
// `internal/config/tests/`, qui est un autre paquet.
//
// Convention du dépôt (rules/tests.md) :
//   - `{paquet}/tests/`            tests en boîte noire, API publique seulement
//   - `{paquet}/internal_test.go`  tests d'internes, à côté du code
//
// Un test d'interne est un aveu de couplage à l'implémentation : il doit rester
// minoritaire, et disparaître si la fonction devient exportée.

// TestDriverTablesAgree : les deux tables de référence doivent se répondre.
//
// Un module connu sans pilote par défaut rendrait la chaîne vide comme pilote, et
// la validation refuserait le démarrage d'un module pourtant correctement déclaré.
// Un défaut absent de la liste des pilotes admis produirait le même refus. Dans les
// deux cas, le message accuserait l'utilisateur d'une faute qui est la nôtre.
func TestDriverTablesAgree(t *testing.T) {
	t.Parallel()

	for module, allowed := range knownDrivers {
		fallback, found := defaultDrivers[module]
		if !found {
			t.Errorf("module %q sans pilote par défaut", module)
			continue
		}
		if !slices.Contains(allowed, fallback) {
			t.Errorf("pilote par défaut de %q = %q, absent de %v", module, fallback, allowed)
		}
	}
	for module := range defaultDrivers {
		if _, found := knownDrivers[module]; !found {
			t.Errorf("module %q a un pilote par défaut mais n'est pas déclaré connu", module)
		}
	}
}

// TestEveryDefaultDriverNeedsNoInfrastructure est le test qui tient la promesse
// centrale de l'ADR 012.
//
// Tous les modules activés d'un coup, tous sur leur pilote par défaut : le résultat
// doit n'exiger NI base, NI cache. Le jour où quelqu'un fait de `postgres` le défaut
// d'un module — par commodité, parce que c'est le pilote le plus complet — ce test
// échoue, et c'est le seul endroit qui s'en apercevra avant les utilisateurs.
func TestEveryDefaultDriverNeedsNoInfrastructure(t *testing.T) {
	t.Parallel()

	all := Modules{}
	for module := range knownDrivers {
		all[module] = Module{Enabled: true}
	}

	if all.RequiresSQL() {
		t.Error("les pilotes par défaut ne doivent exiger aucune base SQL")
	}
	if all.RequiresCache() {
		t.Error("les pilotes par défaut ne doivent exiger aucun cache")
	}
	if problems := all.validate(); len(problems) > 0 {
		t.Errorf("les pilotes par défaut doivent tous être admis: %v", problems)
	}
}

// TestExpandFailsOnMissingRequiredSecret : un secret manquant qui se résoudrait
// en chaîne vide produirait une connexion anonyme ou un chiffrement avec une
// clé vide. Il doit refuser le démarrage.
func TestExpandFailsOnMissingRequiredSecret(t *testing.T) {
	t.Parallel()

	_, err := expand("dsn: ${HEXA_TEST_ABSENT_VAR}")
	if err == nil {
		t.Fatal("une référence obligatoire non définie doit refuser le chargement")
	}
	if !strings.Contains(err.Error(), "HEXA_TEST_ABSENT_VAR") {
		t.Errorf("le message doit nommer la variable manquante: %v", err)
	}
}

func TestExpandUsesExplicitDefault(t *testing.T) {
	t.Parallel()

	out, err := expand("addr: ${HEXA_TEST_ABSENT_VAR:-localhost:6379}")
	if err != nil {
		t.Fatalf("un défaut explicite doit être accepté: %v", err)
	}
	if !strings.Contains(out, "localhost:6379") {
		t.Errorf("défaut non appliqué: %q", out)
	}
}

// TestExpandAcceptsEmptyExplicitDefault : `${VAR:-}` signale un réglage
// optionnel. C'est légitime, contrairement à une référence sans défaut.
func TestExpandAcceptsEmptyExplicitDefault(t *testing.T) {
	t.Parallel()

	if _, err := expand("password: ${HEXA_TEST_ABSENT_VAR:-}"); err != nil {
		t.Errorf("un défaut vide explicite doit être accepté: %v", err)
	}
}

// TestExpandFailsOnDefinedButEmptySecret : le secret déclaré dans une chaîne de
// déploiement mais jamais injecté arrive comme chaîne VIDE, pas comme variable
// absente. C'est la forme la plus fréquente du secret manquant, et celle qui
// passerait le plus facilement inaperçue.
//
// Ce test ne peut pas être parallèle : il manipule l'environnement du processus.
func TestExpandFailsOnDefinedButEmptySecret(t *testing.T) {
	t.Setenv("HEXA_TEST_EMPTY_VAR", "")

	_, err := expand("key: ${HEXA_TEST_EMPTY_VAR}")
	if err == nil {
		t.Fatal("une variable définie mais vide doit refuser le chargement")
	}
	var missing ErrMissingSecret
	if !errors.As(err, &missing) {
		t.Fatalf("attendu ErrMissingSecret, reçu %v", err)
	}
}

// TestExpandPrefersDefaultOverEmptyValue : sémantique POSIX de `${VAR:-défaut}`.
// Le `:` fait porter le repli sur le vide autant que sur l'absence — sinon une
// variable vidée par accident écraserait un défaut pourtant valide.
func TestExpandPrefersDefaultOverEmptyValue(t *testing.T) {
	t.Setenv("HEXA_TEST_EMPTY_VAR", "")

	out, err := expand("addr: ${HEXA_TEST_EMPTY_VAR:-localhost:6379}")
	if err != nil {
		t.Fatalf("un défaut explicite doit s'appliquer: %v", err)
	}
	if !strings.Contains(out, "localhost:6379") {
		t.Errorf("défaut non appliqué face à une variable vide: %q", out)
	}
}

func TestExpandReportsAllMissingVariablesAtOnce(t *testing.T) {
	t.Parallel()

	// Corriger sa configuration en six redémarrages est inacceptable : toutes
	// les variables manquantes sont signalées d'un coup.
	_, err := expand("a: ${HEXA_TEST_A}\nb: ${HEXA_TEST_B}")
	if err == nil {
		t.Fatal("attendu un échec")
	}
	for _, name := range []string{"HEXA_TEST_A", "HEXA_TEST_B"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("le message doit nommer %s: %v", name, err)
		}
	}
}

// TestDeepMergeOverridesLists : concaténer ajouterait silencieusement des
// origines CORS qu'on croyait avoir retirées.
func TestDeepMergeOverridesLists(t *testing.T) {
	t.Parallel()

	base := map[string]any{"http": map[string]any{"allowed_origins": []any{"http://a"}}}
	deepMerge(base, map[string]any{"http": map[string]any{"allowed_origins": []any{"https://b"}}})

	http, ok := base["http"].(map[string]any)
	if !ok {
		t.Fatal("structure perdue lors de la fusion")
	}
	origins, ok := http["allowed_origins"].([]any)
	if !ok {
		t.Fatal("liste perdue lors de la fusion")
	}
	if len(origins) != 1 || origins[0] != "https://b" {
		t.Errorf("la couche supérieure doit ÉCRASER la liste, obtenu: %v", origins)
	}
}

func TestDeepMergeMergesNestedTables(t *testing.T) {
	t.Parallel()

	base := map[string]any{"db": map[string]any{"max_conns": 10, "dsn": "a"}}
	deepMerge(base, map[string]any{"db": map[string]any{"max_conns": 25}})

	db, ok := base["db"].(map[string]any)
	if !ok {
		t.Fatal("structure perdue")
	}
	if db["max_conns"] != 25 {
		t.Errorf("max_conns non surchargé: %v", db["max_conns"])
	}
	if db["dsn"] != "a" {
		t.Errorf("dsn perdu lors de la fusion: %v", db["dsn"])
	}
}

// TestModulesValidateRefusesUnknownDriver : deny par défaut jusque dans la
// lecture de la configuration. Une faute de frappe ne se résout jamais en
// « le pilote le plus proche ».
func TestModulesValidateRefusesUnknownDriver(t *testing.T) {
	t.Parallel()

	problems := Modules{"outbox": {Enabled: true, Driver: "postgresql"}}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s) signalé(s), attendu 1", len(problems))
	}
	if !strings.Contains(problems[0].Error(), "postgresql") {
		t.Errorf("le message doit nommer le pilote fautif: %v", problems[0])
	}
}

func TestModulesValidateRefusesUnknownModule(t *testing.T) {
	t.Parallel()

	if problems := (Modules{"blockchain": {Enabled: true}}).validate(); len(problems) != 1 {
		t.Fatalf("%d problème(s), attendu 1", len(problems))
	}
}

// TestModulesValidateIgnoresDisabledModules : on ne bloque pas un démarrage sur
// la configuration d'une capacité qu'on n'utilise pas.
func TestModulesValidateIgnoresDisabledModules(t *testing.T) {
	t.Parallel()

	problems := Modules{"outbox": {Enabled: false, Driver: "n-importe-quoi"}}.validate()
	if len(problems) != 0 {
		t.Errorf("un module désactivé ne doit rien signaler: %v", problems)
	}
}

// TestInteropValidateRequiresBaseURLForHTTP : un transport http sans adresse
// échouerait au premier appel, en production. Il doit échouer au démarrage.
func TestInteropValidateRequiresBaseURLForHTTP(t *testing.T) {
	t.Parallel()

	problems := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "http"},
	}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s), attendu 1 (base_urls manquant)", len(problems))
	}
}

func TestInteropValidateRefusesUnknownTransport(t *testing.T) {
	t.Parallel()

	problems := Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "grpc"},
	}.validate()
	if len(problems) != 1 {
		t.Fatalf("%d problème(s), attendu 1", len(problems))
	}
}
