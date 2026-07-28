package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCopierReecritSaufLHistoireEtPreserveLesPermissions couvre le cœur du
// générateur : ce qui est recopié, ce qui est réécrit, ce qui ne l'est pas.
//
// Trois propriétés, et chacune a déjà coûté quelque chose à ce dépôt :
//
//   - le chemin de module est réécrit dans le code. Une réécriture partielle
//     produit un projet qui COMPILE — Go résout l'import vers le socle d'origine
//     s'il est accessible — mais qui dépend en silence d'un autre dépôt ;
//   - `CLAUDE.md` ne l'est PAS : ses occurrences sont des liens vers des PR
//     historiques, qui pointeraient vers un dépôt ne les ayant jamais portées ;
//   - le bit exécutable est préservé. Ce dépôt a versionné ses deux crochets git
//     en 100644 : git les ignorait partout, sur toutes les machines, sans que
//     rien ne le signale.
func TestCopierReecritSaufLHistoireEtPreserveLesPermissions(t *testing.T) {
	t.Parallel()

	const moduleSocle = "github.com/exemple/socle"
	const moduleCible = "github.com/impactone/facturation"

	socle := preparerSocle(t, moduleSocle)
	destination := filepath.Join(t.TempDir(), "genere")

	p := plan{source: socle, destination: destination, moduleSocle: moduleSocle, moduleCible: moduleCible}

	fichiers, err := fichiersSuivis(context.Background(), socle)
	if err != nil {
		t.Fatalf("fichiersSuivis: %v", err)
	}
	if erreurCopie := copier(p, fichiers); erreurCopie != nil {
		t.Fatalf("copier: %v", erreurCopie)
	}

	// 1. Le code est réécrit.
	code := lire(t, filepath.Join(destination, "internal", "code.go"))
	if strings.Contains(code, moduleSocle) {
		t.Error("le chemin du socle subsiste dans le code : le projet dépendrait d'un autre dépôt")
	}
	if !strings.Contains(code, moduleCible) {
		t.Error("le chemin cible n'a pas été écrit")
	}

	// 2. CLAUDE.md ne l'est pas.
	if garde := lire(t, filepath.Join(destination, "CLAUDE.md")); !strings.Contains(garde, moduleSocle) {
		t.Error("CLAUDE.md a été réécrit : ses liens vers l'historique du socle sont perdus")
	}

	// 3. Le bit exécutable survit.
	info, err := os.Stat(filepath.Join(destination, "tools", "garde.sh"))
	if err != nil {
		t.Fatalf("relecture du script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("permissions = %v : un garde non exécutable ne garde rien", info.Mode().Perm())
	}

	// 4. La vérification finale doit être satisfaite.
	if err := verifierAucuneTrace(p); err != nil {
		t.Errorf("verifierAucuneTrace refuse un projet pourtant correct: %v", err)
	}
}

// preparerSocle fabrique un dépôt git minimal qui ressemble au socle.
func preparerSocle(t *testing.T, module string) string {
	t.Helper()

	racine := t.TempDir()
	ecrire(t, filepath.Join(racine, "go.mod"), "module "+module+"\n\ngo 1.25.12\n", 0o600)
	ecrire(t, filepath.Join(racine, "internal", "code.go"),
		"package internal\n\nimport \""+module+"/internal/pkg/fp\"\n\nvar _ = fp.None[int]\n", 0o600)
	ecrire(t, filepath.Join(racine, "CLAUDE.md"),
		"Voir https://"+module+"/pull/42 pour l'historique.\n", 0o600)
	ecrire(t, filepath.Join(racine, "tools", "garde.sh"), "#!/usr/bin/env sh\nexit 0\n", 0o750)

	for _, args := range [][]string{
		{"init", "--quiet"},
		{"add", "-A"},
	} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = racine
		if sortie, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, sortie)
		}
	}
	return racine
}

func ecrire(t *testing.T, chemin, contenu string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
		t.Fatalf("création de %s: %v", filepath.Dir(chemin), err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), mode); err != nil {
		t.Fatalf("écriture de %s: %v", chemin, err)
	}
}

func lire(t *testing.T, chemin string) string {
	t.Helper()
	contenu, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	return string(contenu)
}
