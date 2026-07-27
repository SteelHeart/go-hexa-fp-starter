package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPlanifierAccepteUnPlanValide : le sens positif, sans lequel les refus ne
// prouveraient rien.
//
// Un `planifier` qui refuserait TOUT passerait tous les tests de refus. C'est la
// même dissymétrie que l'ADR 013 nomme pour les gardes : vérifier qu'une chose
// échoue ne dit rien tant qu'on n'a pas vérifié qu'elle réussit.
func TestPlanifierAccepteUnPlanValide(t *testing.T) {
	t.Parallel()

	socle := t.TempDir()
	if err := os.WriteFile(filepath.Join(socle, "go.mod"),
		[]byte("module github.com/exemple/socle\n\ngo 1.25.12\n"), 0o600); err != nil {
		t.Fatalf("préparation du socle: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "projet-neuf")

	p, err := planifier(destination, "github.com/impactone/facturation", socle)
	if err != nil {
		t.Fatalf("un plan valide doit être accepté: %v", err)
	}
	if p.moduleSocle != "github.com/exemple/socle" {
		t.Errorf("module du socle lu = %q", p.moduleSocle)
	}
	if p.moduleCible != "github.com/impactone/facturation" {
		t.Errorf("module cible = %q", p.moduleCible)
	}
	if !filepath.IsAbs(p.source) || !filepath.IsAbs(p.destination) {
		t.Error("les chemins doivent être absolus : un chemin relatif dépendrait du " +
			"répertoire courant au moment de la copie, pas de celui de l'appel")
	}
}
