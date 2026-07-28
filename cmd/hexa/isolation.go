package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Le durcissement que personne n'aurait vu manquer
// ─────────────────────────────────────────────────────────────────────────────
//
// `arch-go.yml` interdit à un module métier d'en importer un autre — l'étanchéité
// entre contextes (ADR 011). Mais la règle NOMME le module en dur :
//
//	- package: "**.internal.modules.user_registration.**"
//	  shouldNotDependsOn:
//	    internal:
//	      - "**.internal.modules.!(user_registration).**"
//
// Un module créé sans y toucher n'est donc couvert par AUCUNE règle
// d'étanchéité. Il pourrait importer n'importe quel autre module, et `arch-go`
// afficherait « 100 % de conformité » — parce qu'il n'a rien à dire sur un module
// dont personne ne lui a parlé.
//
// C'est très exactement la forme de défaut que ce dépôt a payée onze fois : un
// garde muet est indiscernable d'un garde satisfait. `hexa make:feature` écrit
// donc la règle en même temps que le module, et REFUSE si elle ne peut pas
// l'écrire — plutôt que de créer un module hors garde en silence.

// ancrageIsolation repère où insérer la règle d'étanchéité.
//
// Un type plutôt que trois retours : la règle des deux retours vaut ici aussi, et
// `arch-go` la fait respecter sur `cmd/**`.
type ancrageIsolation struct {
	chemin  string
	contenu string
	// apres est l'index de FIN de la dernière règle d'étanchéité existante.
	apres int
}

// motifIsolation reconnaît la dernière ligne d'une règle d'étanchéité.
//
// L'ancrage porte sur la FORME de la règle, pas sur un commentaire ni sur un
// numéro de section : ceux-là bougent au premier remaniement du fichier, la forme
// non. Si elle bouge quand même, la commande refuse — elle ne devine pas.
var motifIsolation = regexp.MustCompile(`(?m)^\s+- "\*\*\.internal\.modules\.!\([a-z0-9_]+\)\.\*\*"$`)

// trouverAncrage lit `arch-go.yml` et localise le point d'insertion.
func trouverAncrage(racine, dir string) (ancrageIsolation, error) {
	chemin := filepath.Join(racine, "arch-go.yml")
	// Racine désignée par l'appelant, nom de fichier fixe.
	//nolint:gosec // racine fournie par l'utilisateur, nom de fichier constant
	brut, err := os.ReadFile(chemin)
	if err != nil {
		return ancrageIsolation{}, fmt.Errorf(
			"%s illisible — est-ce bien la racine d'un projet issu du socle ? %w", chemin, err)
	}
	contenu := string(brut)

	if strings.Contains(contenu, "internal.modules."+dir+".**") {
		return ancrageIsolation{}, fmt.Errorf(
			"arch-go.yml déclare déjà une règle pour %q — le module existe-t-il ailleurs ?", dir)
	}

	positions := motifIsolation.FindAllStringIndex(contenu, -1)
	if len(positions) == 0 {
		return ancrageIsolation{}, fmt.Errorf(
			"aucune règle d'étanchéité dans %s : impossible d'y ajouter celle de %q.\n"+
				"       Refus délibéré — créer un module qu'AUCUNE règle ne garde serait pire\n"+
				"       que ne pas le créer. Ajouter à la main, sous `dependenciesRules:` :\n\n%s",
			chemin, dir, regleIsolation(dir))
	}

	derniere := positions[len(positions)-1]
	return ancrageIsolation{chemin: chemin, contenu: contenu, apres: derniere[1]}, nil
}

// regleIsolation rend le bloc YAML d'étanchéité d'un module.
func regleIsolation(dir string) string {
	return fmt.Sprintf(`  - package: "**.internal.modules.%s.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.!(%s).**"
`, dir, dir)
}

// declarerIsolation insère la règle dans `arch-go.yml`.
//
// L'écriture préserve le reste du fichier à l'octet près : `arch-go.yml` est
// presque entièrement fait de commentaires qui expliquent POURQUOI chaque règle
// existe, et un aller-retour YAML les effacerait tous.
func declarerIsolation(a ancrageIsolation, dir string) error {
	fusion := a.contenu[:a.apres] + "\n\n" + strings.TrimRight(regleIsolation(dir), "\n") + a.contenu[a.apres:]

	if err := os.WriteFile(a.chemin, []byte(fusion), 0o600); err != nil {
		return fmt.Errorf("écriture de %s: %w", a.chemin, err)
	}
	return nil
}
