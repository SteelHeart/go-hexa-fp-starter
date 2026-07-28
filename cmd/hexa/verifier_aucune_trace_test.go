package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifierAucuneTraceSaitEchouer est le témoin du garde de génération
// (ADR 013).
//
// Sans lui, on ne distinguerait pas « la réécriture a tout couvert » de « la
// vérification ne regarde plus rien ». Et le mode de défaillance est le pire
// possible : le projet COMPILE, parce que Go résout l'import vers le socle
// d'origine tant qu'il est accessible. Le symptôme arrive des semaines plus
// tard, sur une autre machine, sous la forme d'un paquet introuvable.
func TestVerifierAucuneTraceSaitEchouer(t *testing.T) {
	t.Parallel()

	const moduleSocle = "github.com/exemple/socle"
	destination := t.TempDir()
	p := plan{destination: destination, moduleSocle: moduleSocle, moduleCible: "github.com/impactone/x"}

	// Sens succès : rien ne porte le chemin du socle.
	ecrire(t, filepath.Join(destination, "propre.go"), "package propre\n", 0o600)
	if err := verifierAucuneTrace(p); err != nil {
		t.Fatalf("un projet propre doit passer: %v", err)
	}

	// Sens échec : un fichier ordinaire garde le chemin du socle.
	ecrire(t, filepath.Join(destination, "oublie.go"),
		"package oublie\n\nimport \""+moduleSocle+"/internal/pkg/fp\"\n", 0o600)

	err := verifierAucuneTrace(p)
	if err == nil {
		t.Fatal("un chemin de socle oublié doit être refusé : sinon le projet dépendrait " +
			"en silence d'un autre dépôt, et il compilerait")
	}
	if !strings.Contains(err.Error(), "1 fichier") {
		t.Errorf("le message doit compter les fichiers fautifs: %v", err)
	}

	// Un fichier de la liste déclarée ne compte pas, même s'il porte le chemin.
	if err := os.Remove(filepath.Join(destination, "oublie.go")); err != nil {
		t.Fatalf("nettoyage: %v", err)
	}
	ecrire(t, filepath.Join(destination, "CLAUDE.md"), "voir https://"+moduleSocle+"/pull/1\n", 0o600)
	if err := verifierAucuneTrace(p); err != nil {
		t.Errorf("un fichier déclaré dans citeLeSocleParHistoire ne doit pas être compté: %v", err)
	}
}
