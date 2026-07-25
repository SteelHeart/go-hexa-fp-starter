package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainShortCircuitsAtFirstError : Chain s'arrête à la PREMIÈRE erreur et
// n'exécute aucune étape suivante.
//
// C'est le patron imposé pour écrire un cas d'usage. Le court-circuit n'est pas une
// optimisation : les étapes d'un pipeline supposent que les précédentes ont réussi.
// Une étape « valider l'adresse » qui tournerait après l'échec de « analyser la
// requête » travaillerait sur une valeur zéro, et produirait une seconde erreur qui
// masquerait la vraie.
func TestChainShortCircuitsAtFirstError(t *testing.T) {
	t.Parallel()

	var executees []string

	etape := func(nom string, sortie result.Result[int, erreur]) func(int) result.Result[int, erreur] {
		return func(int) result.Result[int, erreur] {
			executees = append(executees, nom)
			return sortie
		}
	}

	sortie := result.Chain(okInt(1),
		etape("première", okInt(2)),
		etape("deuxième", errInt("refusé")),
		etape("troisième", okInt(4)),
	)

	if sortie.IsOk() {
		t.Fatal("une étape en échec doit faire échouer toute la chaîne")
	}
	if cause(sortie) != "refusé" {
		t.Errorf("erreur = %q, attendu celle de l'étape fautive", cause(sortie))
	}
	if len(executees) != 2 {
		t.Errorf("étapes exécutées = %v, attendu les deux premières seulement", executees)
	}
}
