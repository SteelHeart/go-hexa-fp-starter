package tests

import (
	"strings"
	"testing"
)

// TestTheClearPasswordNeverReachesStorage est le test de sécurité du cas d'usage.
//
// Le mot de passe en clair ne doit atteindre QU'UN seul port : celui du hachage.
// Ni l'écriture, ni l'événement ne doivent le voir — parce que ce qui atteint le
// stockage finit dans les sauvegardes, et ce qui atteint l'événement finit dans le
// courtier, puis chez tous les consommateurs.
//
// Le type `RawPassword` protège des fuites par journalisation ; ce test protège de
// la fuite par TRANSPORT, que le type ne peut pas empêcher.
func TestTheClearPasswordNeverReachesStorage(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	user := utilisateurDe(t, inscrit(depsNominales(observe)))

	// Le port de hachage, lui, doit bien recevoir la valeur : c'est son rôle.
	if observe.hache != motDePasseFort {
		t.Error("le port de hachage doit recevoir le mot de passe en clair")
	}

	if observe.sauvegarde.PasswordHash.String() != condense {
		t.Errorf("l'utilisateur écrit porte %q, attendu le condensé",
			observe.sauvegarde.PasswordHash.String())
	}
	if strings.Contains(observe.sauvegarde.PasswordHash.String(), motDePasseFort) {
		t.Error("le mot de passe en clair a atteint le stockage")
	}
	if user.PasswordHash.String() == motDePasseFort {
		t.Error("l'utilisateur rendu porte le mot de passe en clair")
	}
}
