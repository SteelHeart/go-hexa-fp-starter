// Package tests contient les tests en BOÎTE NOIRE du générateur : ils
// n'utilisent que son API publique, exactement comme `cmd/hexa`.
//
// # Pourquoi ces tests existaient ailleurs, et mal
//
// Ils vivaient à la racine de `cmd/hexa`, en `package main`. C'était le seul
// emplacement possible — Go interdit d'importer un paquet `main` — mais ce
// n'était ni la boîte noire ni `internal_test.go`, les deux seuls que
// `rules/tests.md` prévoit. Sortir la logique dans `internal/generator` a rendu
// cet emplacement-ci possible (#96).
//
// Un fichier par test, nommé d'après lui, aides partagées ici et nulle part
// ailleurs.
package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// archGoWithOneRule est le contenu minimal qu'attend `PlanFeature`.
//
// Il porte des commentaires DE PART ET D'AUTRE du point d'insertion : c'est ce
// qui permet de vérifier que l'insertion ne les efface pas.
const archGoWithOneRule = `dependenciesRules:
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
`

// fakeProject builds the minimal structure a feature plan requires.
//
// Building it here rather than generating a real project keeps these tests in
// milliseconds: a real generation runs `go build` and `go test`.
func fakeProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write(t, filepath.Join(root, "go.mod"), "module github.com/example/project\n\ngo 1.25.12\n", 0o600)
	write(t, filepath.Join(root, "arch-go.yml"), archGoWithOneRule, 0o600)
	return root
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(content)
}
