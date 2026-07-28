package generator

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectPlan porte les valeurs vérifiées d'une génération de projet.
//
// Un type plutôt que quatre paramètres : la règle des deux retours vaut aussi
// pour les entrées, et quatre chaînes de suite s'inversent silencieusement.
type ProjectPlan struct {
	// Source est la racine du socle recopié.
	Source string
	// Destination est le répertoire du projet créé.
	Destination string
	// SocleModule est le chemin de module du socle, celui qu'on remplace.
	SocleModule string
	// TargetModule est le chemin de module du projet créé.
	TargetModule string
}

// PlanProject vérifie TOUT avant qu'un seul fichier soit écrit.
//
// Chaque étape REFUSE plutôt que de réparer : un générateur qui rattrape
// silencieusement une entrée douteuse produit un projet dont personne ne sait
// dans quel état il est.
func PlanProject(destination, targetModule, source string) (ProjectPlan, error) {
	if targetModule == "" {
		return ProjectPlan{}, errors.New("--module est obligatoire : un projet sans chemin de module ne compile pas")
	}
	if !strings.Contains(targetModule, "/") {
		return ProjectPlan{}, fmt.Errorf(
			"--module=%q ne ressemble pas à un chemin de module (attendu : hôte/organisation/nom)", targetModule)
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		return ProjectPlan{}, fmt.Errorf("chemin du socle: %w", err)
	}
	socleModule, err := ModulePathOf(absSource)
	if err != nil {
		return ProjectPlan{}, err
	}
	if socleModule == targetModule {
		return ProjectPlan{}, fmt.Errorf(
			"--module=%q est déjà le module du socle : rien à réécrire", targetModule)
	}

	absDest, err := filepath.Abs(destination)
	if err != nil {
		return ProjectPlan{}, fmt.Errorf("chemin de destination: %w", err)
	}
	if occupied := EmptyDestination(absDest); occupied != nil {
		return ProjectPlan{}, occupied
	}

	return ProjectPlan{
		Source:       absSource,
		Destination:  absDest,
		SocleModule:  socleModule,
		TargetModule: targetModule,
	}, nil
}
