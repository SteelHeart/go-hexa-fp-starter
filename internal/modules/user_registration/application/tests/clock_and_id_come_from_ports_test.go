package tests

import (
	"testing"
)

// TestClockAndIDComeFromPorts : ni l'heure ni l'identifiant ne sont produits par le
// cœur.
//
// C'est ce qui rend ce test déterministe, et c'est aussi ce qui rend le cas d'usage
// utilisable par un seeder : rejouer une inscription avec un identifiant et une
// date choisis permet de reconstituer un jeu de données reproductible.
//
// Un `time.Now()` ou un `uuid.New()` glissé dans le pipeline casserait les deux
// propriétés d'un coup, et le test échouerait — ce qui est exactement son rôle.
func TestClockAndIDComeFromPorts(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	user := utilisateurDe(t, inscrit(depsNominales(observe)))

	if !observe.aAppele("Now") {
		t.Error("le port Now doit être appelé : le cœur ne lit pas l'horloge")
	}
	if !observe.aAppele("GenerateID") {
		t.Error("le port GenerateID doit être appelé : le cœur ne produit pas d'aléa")
	}
	if !user.CreatedAt.Equal(instantFixe()) {
		t.Errorf("date = %v : elle ne vient pas du port", user.CreatedAt)
	}
}
