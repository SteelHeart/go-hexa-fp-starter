package config

import (
	"errors"
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

// TestAnEmptyCatalogRefusesEverything : le deny-par-défaut de l'ADR 014.
//
// Le catalogue vient du composition root. S'il est vide — parce que personne ne
// l'a construit, ou parce qu'on l'a oublié — la configuration ne doit RIEN
// admettre. L'oubli du catalogue doit être bruyant, jamais permissif : c'est ce
// qui distingue « aucun module déclaré » de « tous les modules acceptés ».
func TestAnEmptyCatalogRefusesEverything(t *testing.T) {
	t.Parallel()

	mods := Modules{"quelconque": {Enabled: true, Driver: "memory"}}

	problems := mods.validate(ModuleCatalog{})
	if len(problems) == 0 {
		t.Fatal("un catalogue vide doit tout refuser")
	}
	if !strings.Contains(problems[0].Error(), "quelconque") {
		t.Errorf("le message doit nommer le module refusé: %v", problems[0])
	}
}

// TestADriverAbsentFromTheCatalogIsRefused : une faute de frappe dans un nom de
// pilote ne se résout jamais en « le plus proche ».
func TestADriverAbsentFromTheCatalogIsRefused(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	mods := Modules{"facturation": {Enabled: true, Driver: "memry"}}

	problems := mods.validate(catalog)
	if len(problems) != 1 {
		t.Fatalf("un pilote inconnu doit produire exactement un refus, obtenu %d", len(problems))
	}
	if !strings.Contains(problems[0].Error(), "memry") {
		t.Errorf("le message doit citer le pilote fautif: %v", problems[0])
	}
}

// TestADisabledModuleIsNotValidated : on ne refuse pas le démarrage pour le
// pilote d'un module que personne n'a activé.
//
// Sans ça, une configuration d'exemple laissée en place — module désactivé,
// pilote `postgres` — empêcherait de démarrer sans base, ce qui contredirait
// frontalement l'ADR 012.
func TestADisabledModuleIsNotValidated(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	mods := Modules{"facturation": {Enabled: false, Driver: "un-pilote-qui-n-existe-pas"}}

	if problems := mods.validate(catalog); len(problems) != 0 {
		t.Errorf("un module désactivé ne doit pas être validé: %v", problems)
	}
}

// TestResolveDoesNotMutateItsInput : Resolve est une fonction PURE.
//
// Elle rend une copie où les défauts sont posés. Si elle modifiait son entrée,
// « appliquer les défauts » redeviendrait un effet caché — exactement ce que
// l'ADR 014 déplace hors des accesseurs.
func TestResolveDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()

	catalog := ModuleCatalog{
		"facturation": {Default: "memory", Drivers: map[string]Resources{"memory": {}}},
	}
	origine := Modules{"facturation": {Enabled: true}}

	resolved := origine.Resolve(catalog)

	if got := origine["facturation"].Driver; got != "" {
		t.Errorf("Resolve a modifié son entrée: pilote devenu %q", got)
	}
	if got := resolved["facturation"].Driver; got != "memory" {
		t.Errorf("le défaut du catalogue n'a pas été posé: %q", got)
	}
}

// TestMergeCatalogsRefusesACollision : deux modules ne peuvent pas porter le
// même nom.
//
// Le mode de défaillance évité est silencieux : l'un des deux se retrouverait
// configuré par les pilotes de l'autre, et le premier symptôme serait un pilote
// « inconnu » pour un module qui le déclare pourtant.
func TestMergeCatalogsRefusesACollision(t *testing.T) {
	t.Parallel()

	un := ModuleCatalog{"facturation": {Default: "memory"}}
	deux := ModuleCatalog{"facturation": {Default: "postgres"}}

	if _, err := MergeCatalogs(un, deux); err == nil {
		t.Fatal("un nom de module déclaré deux fois doit être refusé")
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
