package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// TestTheCLICallsTheSamePortAsHTTP est le TÉMOIN de l'issue #8.
//
// # Ce qu'il démontre
//
// Que la surface est un détail. Cette commande appelle `mod.Register` — le même
// port que `POST /v1/users` — et obtient le même résultat, produit par le même
// code de domaine. Le module ne sait pas laquelle des deux surfaces l'appelle,
// et il n'a rien fallu lui ajouter pour qu'il serve les deux.
//
// C'est la propriété n°2 du socle, *le nombre de frontends est un non-sujet*.
// Avec une seule surface elle était énoncée ; avec deux, elle est mesurée.
//
// # Ce que le test vérifie, précisément
//
// L'identifiant, l'adresse NORMALISÉE et le statut sortent sur la sortie
// STANDARD, séparés par des tabulations — une ligne qu'un script peut couper. Et
// le code de retour est zéro.
func TestTheCLICallsTheSamePortAsHTTP(t *testing.T) {
	t.Parallel()

	commande, flux := nouvelleCommande(t, secret+"\n")
	code := commande.Register(context.Background(), []string{"--email", "  Alice@Example.COM "})

	if code != exit.OK {
		t.Fatalf("attendu %d, obtenu %d — err: %s", exit.OK, code, flux.err.String())
	}

	ligne := strings.TrimSpace(flux.out.String())
	champs := strings.Split(ligne, "\t")
	if len(champs) != 3 {
		t.Fatalf("la sortie doit porter 3 champs séparés par des tabulations, obtenu %q", ligne)
	}
	if champs[1] != email {
		t.Fatalf("l'adresse doit être NORMALISÉE par le domaine : %q au lieu de %q", champs[1], email)
	}
	if champs[2] != "pending" {
		t.Fatalf("le compte naît `pending`, obtenu %q", champs[2])
	}
}

// TestTheSecretNeverLeaksIntoTheOutput garde les deux flux.
//
// Un secret recopié dans la sortie finirait dans le journal de tout script qui
// capture la commande — et un script d'administration, on le capture toujours.
// Le condensé non plus ne doit pas sortir : il se casse hors ligne.
func TestTheSecretNeverLeaksIntoTheOutput(t *testing.T) {
	t.Parallel()

	commande, flux := nouvelleCommande(t, secret+"\n")
	if code := commande.Register(context.Background(), []string{"--email", email}); code != exit.OK {
		t.Fatalf("inscription: code %d — %s", code, flux.err.String())
	}

	for nom, contenu := range map[string]string{"stdout": flux.out.String(), "stderr": flux.err.String()} {
		if strings.Contains(contenu, secret) {
			t.Errorf("le secret EN CLAIR fuite sur %s : %q", nom, contenu)
		}
		if strings.Contains(contenu, "condensé") {
			t.Errorf("le condensé fuite sur %s : %q", nom, contenu)
		}
	}
}

// TestErrorsGoToStderrNeverToStdout : les deux flux ne se mélangent pas.
//
// # Pourquoi c'est une propriété et pas une préférence
//
// La sortie standard d'une CLI est une DONNÉE : un script en aval la découpe. Y
// écrire un message d'erreur casse ce découpage — et le casse le jour de
// l'incident, quand un compte est refusé, c'est-à-dire au pire moment.
//
// Le défaut est invisible tant qu'on regarde un terminal, où les deux flux
// s'affichent au même endroit. Il n'apparaît qu'en redirigeant l'un sans l'autre.
func TestErrorsGoToStderrNeverToStdout(t *testing.T) {
	t.Parallel()

	commande, flux := nouvelleCommande(t, "trop-court\n")
	code := commande.Register(context.Background(), []string{"--email", email})

	if code != exit.DataErr {
		t.Fatalf("un secret trop court est une donnée invalide : attendu %d, obtenu %d", exit.DataErr, code)
	}
	if flux.out.Len() != 0 {
		t.Fatalf("rien ne doit sortir sur stdout en cas d'échec, obtenu %q", flux.out.String())
	}
	if flux.err.Len() == 0 {
		t.Fatal("l'erreur doit être écrite sur stderr")
	}
	if !strings.Contains(flux.err.String(), "champ:") {
		t.Fatalf("le champ fautif doit être nommé : %q", flux.err.String())
	}
}
