package tests

import "testing"

// TestOkAndErrAreMutuallyExclusive : un Result porte un succès OU une erreur,
// jamais les deux, jamais ni l'un ni l'autre.
//
// C'est ce qui le distingue du couple `(T, error)` de Go, où rien n'empêche de
// retourner une valeur ET une erreur — et où une bonne part des défauts vient de
// ce que l'appelant lit la valeur sans regarder l'erreur.
func TestOkAndErrAreMutuallyExclusive(t *testing.T) {
	t.Parallel()

	succes := okInt(7)
	if !succes.IsOk() || succes.IsErr() {
		t.Error("un Ok doit être IsOk et non IsErr")
	}

	echec := errInt("refusé")
	if echec.IsOk() || !echec.IsErr() {
		t.Error("un Err doit être IsErr et non IsOk")
	}
}
