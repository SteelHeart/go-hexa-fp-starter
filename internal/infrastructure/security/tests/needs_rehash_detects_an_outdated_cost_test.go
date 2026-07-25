package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestNeedsRehashDetectsAnOutdatedCost : un condensé produit avec un coût inférieur
// au coût courant doit être refait.
//
// C'est ce qui permet de remonter le coût sans invalider les comptes existants :
// après une vérification réussie, on rehache avec les paramètres du jour. Sans
// cela, augmenter le coût ne protégerait QUE les nouveaux comptes — c'est-à-dire
// pas ceux qui existent depuis assez longtemps pour avoir fuité.
//
// Un condensé illisible rend `true` : mieux vaut le refaire que le conserver.
func TestNeedsRehashDetectsAnOutdatedCost(t *testing.T) {
	t.Parallel()

	ancien := hash(t, "correct cheval batterie agrafe")

	memeCout := newHasher()
	if memeCout.NeedsRehash(ancien) {
		t.Error("un condensé produit au coût courant ne doit pas être refait")
	}

	plusCher := security.NewHasher(security.Argon2Params{
		MemoryKiB: testParams().MemoryKiB * 4, Iterations: 2, Threads: 1,
	})
	if !plusCher.NeedsRehash(ancien) {
		t.Error("un condensé produit à un coût inférieur doit être refait")
	}

	if !memeCout.NeedsRehash("condensé illisible") {
		t.Error("un condensé illisible doit être refait, pas conservé")
	}
}
