package main

import "testing"

// TestCiteLeSocleParHistoire : la liste d'exceptions est ÉNUMÉRÉE, jamais
// devinée.
//
// Ces fichiers portent le chemin du socle dans des LIENS vers ses PR et ses
// issues, pas dans des imports. Les réécrire ferait pointer l'historique du
// socle vers un dépôt qui ne l'a jamais porté.
//
// Le risque symétrique est plus grave : une exception trop large laisserait un
// vrai import non réécrit, et le projet dépendrait en silence d'un autre dépôt.
// D'où les cas négatifs ci-dessous, qui valent autant que les positifs.
func TestCiteLeSocleParHistoire(t *testing.T) {
	t.Parallel()

	cas := map[string]bool{
		"CLAUDE.md":                              true,
		"documentation/process/REPRISE.md":       true,
		"documentation/adr/013-un-garde.md":      true,
		"documentation/adr/README.md":            true,
		"README.md":                              false,
		"documentation/process/NOMENCLATURE.md":  false,
		"documentation/technique/pilotes.md":     false,
		"internal/config/config.go":              false,
		"cmd/server/main.go":                     false,
		"go.mod":                                 false,
		"documentation/adrien/note.md":           false,
		"quelquechose/documentation/adr/faux.md": false,
	}

	for chemin, voulu := range cas {
		if got := citeLeSocleParHistoire(chemin); got != voulu {
			t.Errorf("citeLeSocleParHistoire(%q) = %v, attendu %v", chemin, got, voulu)
		}
	}
}
