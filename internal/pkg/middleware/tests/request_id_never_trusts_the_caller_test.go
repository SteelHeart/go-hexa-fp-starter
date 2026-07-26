package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRequestIDNeverTrustsTheCaller : un identifiant de corrélation hostile est
// remplacé, jamais propagé.
//
// # Le chemin d'attaque
//
// Cet identifiant FINIT DANS LES JOURNAUX. Un appelant qui y glisse un retour
// chariot injecte donc des lignes entières dans le journal du serveur : fausses
// entrées, faux niveaux de gravité, effacement visuel d'une trace gênante. C'est
// une injection de journal, et elle vise précisément ce qui sert à enquêter après
// coup.
//
// Une valeur démesurée, elle, gonfle chaque ligne de journal d'une requête —
// donc le coût de stockage et le temps d'analyse.
//
// Le middleware réutilise l'identifiant fourni quand il est plausible, ce qui est
// utile pour suivre un appel à travers plusieurs services. « Plausible » doit donc
// être vérifié, pas supposé.
func TestRequestIDNeverTrustsTheCaller(t *testing.T) {
	t.Parallel()

	hostile := map[string]string{
		"retour chariot": "abc\r\nlevel=ERROR msg=\"faux incident\"",
		"saut de ligne":  "abc\ninjection",
		"démesuré":       strings.Repeat("x", 500),
		"vide":           "",
	}

	for name, value := range hostile {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set(middleware.RequestIDHeader, value)
			got := call(middleware.RequestID(), req, okHandler(nil)).Header().Get(middleware.RequestIDHeader)

			if got == value {
				t.Errorf("identifiant hostile propagé tel quel: %q", got)
			}
			if strings.ContainsAny(got, "\r\n") {
				t.Errorf("l'identifiant rendu contient un saut de ligne: %q", got)
			}
			if got == "" {
				t.Error("un identifiant doit toujours être rendu, même après refus de celui reçu")
			}
		})
	}
}
