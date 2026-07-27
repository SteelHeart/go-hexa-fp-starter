package tests

import "testing"

// TestGetForcesDiscrimination : `Get` est la seule sortie de la boîte, et elle
// rend un booléen.
//
// Ce booléen n'est pas une commodité, c'est une contrainte : il force le site
// d'appel à discriminer. Sans lui, on pourrait lire la valeur d'un Result en
// erreur et récupérer la valeur zéro de T — exactement le défaut que Result existe
// pour rendre impossible.
func TestGetForcesDiscrimination(t *testing.T) {
	t.Parallel()

	value, err, ok := okInt(7).Get()
	if !ok {
		t.Fatal("un Ok doit rendre ok=true")
	}
	if value != 7 {
		t.Errorf("valeur = %d, attendu 7", value)
	}
	if err != "" {
		t.Errorf("un Ok ne doit porter aucune erreur, reçu %q", err)
	}

	value, err, ok = errInt("refusé").Get()
	if ok {
		t.Fatal("un Err doit rendre ok=false")
	}
	if err != "refusé" {
		t.Errorf("erreur = %q, attendu « refusé »", err)
	}
	if value != 0 {
		t.Errorf("un Err ne doit porter aucune valeur, reçu %d", value)
	}
}
