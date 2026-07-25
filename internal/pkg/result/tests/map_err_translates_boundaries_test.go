package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapErrTranslatesBoundaries : MapErr est la fonction de traduction des
// frontières.
//
// Un adaptateur secondaire s'en sert pour convertir une erreur technique — une
// violation de contrainte SQL — en erreur de domaine — « cette adresse est déjà
// prise ». Sans elle, soit le domaine connaîtrait les codes SQLSTATE, soit
// l'appelant recevrait une erreur qu'il ne peut pas interpréter.
//
// Symétrie exigée : elle ne doit PAS toucher un succès.
func TestMapErrTranslatesBoundaries(t *testing.T) {
	t.Parallel()

	traduit := func(e erreur) string { return "domaine: " + string(e) }

	echec := result.MapErr(errInt("23505"), traduit)
	if echec.IsOk() {
		t.Fatal("MapErr sur une erreur doit rendre une erreur")
	}
	if cause(echec) != "domaine: 23505" {
		t.Errorf("erreur traduite = %q", cause(echec))
	}

	appelee := false
	succes := result.MapErr(okInt(7), func(e erreur) string {
		appelee = true
		return string(e)
	})
	if appelee {
		t.Error("la traduction ne doit PAS être appelée sur un succès")
	}
	if valeur(succes) != 7 {
		t.Errorf("la valeur doit traverser inchangée, reçu %d", valeur(succes))
	}
}
