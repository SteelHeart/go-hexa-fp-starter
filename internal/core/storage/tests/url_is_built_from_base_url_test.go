package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestURLIsBuiltFromBaseURL : l'URL rendue préfixe la clé par l'adresse de base
// configurée, sans jamais doubler la barre oblique.
//
// L'appelant stocke cette URL en base. Une barre en trop la rend valide un jour et
// cassée le lendemain selon le serveur qui la sert, et les URL déjà persistées ne
// se corrigent pas d'un déploiement.
func TestURLIsBuiltFromBaseURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"sans barre finale": "/files",
		"avec barre finale": "/files/",
		"adresse absolue":   "https://cdn.example.test/files/",
	}

	for name, base := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mod, err := storage.New(config.Module{
				Enabled: true,
				Driver:  "disk",
				Options: map[string]any{"base_dir": t.TempDir(), "base_url": base},
			}, storage.Deps{})
			if err != nil {
				t.Fatalf("construction: %v", err)
			}

			located, err := mod.Put(context.Background(), object("doc.pdf", "x"))
			if err != nil {
				t.Fatalf("Put: %v", err)
			}

			want := strings.TrimSuffix(base, "/") + "/" + located.Key.String()
			if located.URL != want {
				t.Errorf("URL = %q, attendu %q", located.URL, want)
			}
			if strings.Contains(strings.TrimPrefix(located.URL, "https://"), "//") {
				t.Errorf("URL = %q contient une barre doublée", located.URL)
			}
		})
	}
}
