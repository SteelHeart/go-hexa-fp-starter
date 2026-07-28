package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// validFeatureName contraint le nom d'un module métier.
//
// `snake_case` strict, et c'est une contrainte de forme, pas de goût : le nom
// devient un nom de RÉPERTOIRE, une clé de `config/modules.yaml`, et — dépouillé
// de ses tirets bas — un nom de PAQUET Go. Accepter `Mon-Module` produirait un
// paquet `MonModule`, que `revive` refuse, dans un projet dont la barrière
// échouerait dès la première commande.
var validFeatureName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// FeaturePlan porte les valeurs vérifiées d'une création de module.
//
// Les champs exportés sont ceux que les gabarits interpolent : `Module`, `Dir`,
// `Package`. Les autres le sont pour que les tests puissent les constater sans
// atteindre l'intérieur du paquet.
type FeaturePlan struct {
	// Root est la racine du projet où le module est créé.
	Root string
	// Destination est le répertoire du module.
	Destination string
	// Module est le chemin de module Go du PROJET, pas du module métier.
	Module string
	// Dir est le nom du répertoire, en snake_case : `order_tracking`.
	Dir string
	// Package est le nom du paquet Go, sans tiret bas : `ordertracking`.
	Package string
	// anchor porte le point d'insertion de la règle d'étanchéité d'arch-go.
	//
	// Repéré pendant la PLANIFICATION, écrit pendant l'exécution : sans cela, un
	// arch-go.yml inattendu ferait échouer la commande après avoir déjà créé le
	// module, et laisserait un module hors garde sur le disque.
	anchor IsolationAnchor
}

// PlanFeature vérifie TOUT avant qu'un seul fichier soit écrit.
func PlanFeature(name, root string) (FeaturePlan, error) {
	if !validFeatureName.MatchString(name) {
		return FeaturePlan{}, fmt.Errorf(
			"nom de module %q invalide — attendu du snake_case : `billing`, `order_tracking`", name)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FeaturePlan{}, fmt.Errorf("chemin du projet: %w", err)
	}
	module, err := ModulePathOf(absRoot)
	if err != nil {
		return FeaturePlan{}, err
	}

	destination := filepath.Join(absRoot, "internal", "modules", name)
	if occupied := EmptyDestination(destination); occupied != nil {
		return FeaturePlan{}, fmt.Errorf("le module existe déjà: %w", occupied)
	}

	anchor, err := FindIsolationAnchor(absRoot, name)
	if err != nil {
		return FeaturePlan{}, err
	}

	return FeaturePlan{
		Root:        absRoot,
		Destination: destination,
		Module:      module,
		Dir:         name,
		Package:     strings.ReplaceAll(name, "_", ""),
		anchor:      anchor,
	}, nil
}
