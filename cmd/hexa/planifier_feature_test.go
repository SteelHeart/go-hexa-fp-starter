package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// projetFactice fabrique la structure minimale qu'attend `make:feature`.
//
// Un `go.mod` et un `arch-go.yml` portant au moins une règle d'étanchéité :
// c'est exactement ce que la commande exige, ni plus, ni moins. Le fabriquer ici
// plutôt que de générer un vrai projet garde ces tests en millisecondes.
func projetFactice(t *testing.T) string {
	t.Helper()
	racine := t.TempDir()

	ecrire(t, filepath.Join(racine, "go.mod"), "module github.com/exemple/projet\n\ngo 1.25.12\n", 0o600)
	ecrire(t, filepath.Join(racine, "arch-go.yml"), `dependenciesRules:
  # Un commentaire qui doit SURVIVRE à l'insertion.
  - package: "**.internal.modules.user_registration.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.!(user_registration).**"

  # Un second commentaire, après le point d'ancrage.
  - package: "**.internal.core.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.**"
`, 0o600)
	return racine
}

// TestPlanifierFeatureRefuseAvantDEcrire : chaque refus tombe pendant la
// PLANIFICATION, donc avant qu'un seul fichier soit écrit.
//
// C'est ce qui distingue un générateur d'un script : une commande qui échoue à
// mi-parcours laisse un module à moitié créé, et son auteur doit deviner ce
// qu'il faut nettoyer. Ici, soit tout est écrit, soit rien ne l'est.
func TestPlanifierFeatureRefuseAvantDEcrire(t *testing.T) {
	t.Parallel()

	racine := projetFactice(t)

	cas := map[string]string{
		"majuscules":       "OrderTracking",
		"tiret":            "order-tracking",
		"chiffre en tête":  "1billing",
		"tiret bas doublé": "order__tracking",
		"tiret bas final":  "order_",
		"vide":             "",
		"chemin déguisé":   "../evasion",
	}

	for nom, entree := range cas {
		t.Run(nom, func(t *testing.T) {
			t.Parallel()
			if _, err := planifierFeature(entree, racine); err == nil {
				t.Errorf("planifierFeature(%q) devait refuser", entree)
			}
		})
	}
}

// TestPlanifierFeatureAccepteEtDeriveLePaquet : un nom valide produit le
// répertoire et le nom de paquet attendus.
//
// La dérivation n'est pas cosmétique : `order_tracking` est correct comme nom de
// RÉPERTOIRE et comme clé de configuration, mais `revive` refuse un paquet Go
// portant un tiret bas. Les deux formes coexistent donc, et c'est le générateur
// qui les tient cohérentes.
func TestPlanifierFeatureAccepteEtDeriveLePaquet(t *testing.T) {
	t.Parallel()

	p, err := planifierFeature("order_tracking", projetFactice(t))
	if err != nil {
		t.Fatalf("un nom valide devait passer: %v", err)
	}

	if p.Dir != "order_tracking" {
		t.Errorf("Dir = %q, attendu le snake_case d'origine", p.Dir)
	}
	if p.Package != "ordertracking" {
		t.Errorf("Package = %q, attendu sans tiret bas", p.Package)
	}
	if p.Module != "github.com/exemple/projet" {
		t.Errorf("Module = %q, attendu celui du go.mod", p.Module)
	}
	if !strings.HasSuffix(p.destination, filepath.Join("internal", "modules", "order_tracking")) {
		t.Errorf("destination = %q, attendue sous internal/modules", p.destination)
	}
}

// TestPlanifierFeatureRefuseUnModuleDejaLa : un module existant n'est jamais
// écrasé.
//
// Deny par défaut : écraser est irréversible, et le générateur n'a aucun moyen
// de savoir ce que contenait le module précédent.
func TestPlanifierFeatureRefuseUnModuleDejaLa(t *testing.T) {
	t.Parallel()

	racine := projetFactice(t)
	occupe := filepath.Join(racine, "internal", "modules", "billing")
	if err := os.MkdirAll(occupe, 0o750); err != nil {
		t.Fatalf("préparation: %v", err)
	}
	ecrire(t, filepath.Join(occupe, "module.go"), "package billing\n", 0o600)

	if _, err := planifierFeature("billing", racine); err == nil {
		t.Error("un module existant devait faire refuser la commande")
	}
}
