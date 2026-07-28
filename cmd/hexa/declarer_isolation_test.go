package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDeclarerIsolationAjouteLaRegleSansToucherAuReste : la règle d'étanchéité
// est insérée, et le reste du fichier survit à l'octet près.
//
// `arch-go.yml` est fait presque entièrement de commentaires qui disent POURQUOI
// chaque règle existe. Un aller-retour YAML les effacerait tous — c'est la raison
// pour laquelle l'insertion est textuelle et non structurelle.
func TestDeclarerIsolationAjouteLaRegleSansToucherAuReste(t *testing.T) {
	t.Parallel()

	racine := projetFactice(t)
	chemin := filepath.Join(racine, "arch-go.yml")
	avant := lire(t, chemin)

	ancrage, err := trouverAncrage(racine, "billing")
	if err != nil {
		t.Fatalf("ancrage introuvable: %v", err)
	}
	if err := declarerIsolation(ancrage, "billing"); err != nil {
		t.Fatalf("insertion: %v", err)
	}

	apres := lire(t, chemin)

	if !strings.Contains(apres, `- package: "**.internal.modules.billing.**"`) {
		t.Error("la règle du module neuf est absente")
	}
	if !strings.Contains(apres, `- "**.internal.modules.!(billing).**"`) {
		t.Error("l'exclusion du module neuf est absente")
	}
	for _, garde := range []string{
		"# Un commentaire qui doit SURVIVRE à l'insertion.",
		"# Un second commentaire, après le point d'ancrage.",
		`- package: "**.internal.core.**"`,
	} {
		if !strings.Contains(apres, garde) {
			t.Errorf("l'insertion a perdu %q", garde)
		}
	}
	if len(apres) <= len(avant) {
		t.Error("le fichier devait grandir")
	}
}

// TestTrouverAncrageRefuseUnFichierSansRegle : sans point d'ancrage, la commande
// REFUSE au lieu de créer un module hors garde.
//
// # Pourquoi ce refus est le cœur du sujet
//
// La règle d'étanchéité NOMME chaque module en dur. Un module créé sans elle
// n'est couvert par AUCUNE règle : il pourrait importer n'importe quel autre
// module, et `arch-go` afficherait « 100 % de conformité » — parce qu'il n'a rien
// à dire sur un module dont personne ne lui a parlé.
//
// C'est la onzième fois que ce dépôt rencontre cette forme : un garde muet est
// indiscernable d'un garde satisfait. D'où le refus, plutôt qu'un avertissement
// que personne ne lirait.
func TestTrouverAncrageRefuseUnFichierSansRegle(t *testing.T) {
	t.Parallel()

	racine := projetFactice(t)
	ecrire(t, filepath.Join(racine, "arch-go.yml"), "dependenciesRules: []\n", 0o600)

	_, err := trouverAncrage(racine, "billing")
	if err == nil {
		t.Fatal("un arch-go.yml sans règle d'étanchéité devait faire refuser")
	}
	// Le message doit porter le bloc à ajouter : refuser sans dire quoi faire
	// transforme une garde en impasse.
	if !strings.Contains(err.Error(), `- package: "**.internal.modules.billing.**"`) {
		t.Errorf("le refus doit dicter la règle à écrire, obtenu:\n%v", err)
	}
}

// TestTrouverAncrageRefuseUnModuleDejaDeclare : on ne déclare pas deux fois le
// même module.
//
// Sans ce refus, relancer la commande empilerait des règles identiques, et
// `arch-go` finirait par lire un fichier que personne ne comprend.
func TestTrouverAncrageRefuseUnModuleDejaDeclare(t *testing.T) {
	t.Parallel()

	racine := projetFactice(t)
	if _, err := trouverAncrage(racine, "user_registration"); err == nil {
		t.Error("un module déjà déclaré devait faire refuser")
	}
}

// TestTrouverAncrageRefuseUnProjetSansArchGo : le fichier absent est une erreur,
// pas une permission.
func TestTrouverAncrageRefuseUnProjetSansArchGo(t *testing.T) {
	t.Parallel()

	if _, err := trouverAncrage(t.TempDir(), "billing"); err == nil {
		t.Error("un projet sans arch-go.yml devait faire refuser")
	}
}
