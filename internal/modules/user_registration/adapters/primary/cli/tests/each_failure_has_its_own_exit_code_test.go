package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// TestEachFailureHasItsOwnExitCode : le code de retour sert à décider du RÉESSAI.
//
// # Pourquoi ce n'est pas cosmétique
//
// Un binaire de ligne de commande est appelé par des scripts bien plus souvent
// que par des humains. Ces appelants ne lisent pas les messages ; ils lisent le
// code de retour.
//
// Avec `1` pour tout, un mot de passe trop court et une base injoignable sont
// indiscernables : le script réessaie l'un — inutilement, à l'infini — ou
// abandonne sur l'autre, alors qu'une seconde tentative aurait suffi.
//
// La table est celle de `sysexits.h`, vieille de 1980, pour qu'un opérateur qui
// voit `78` sache déjà qu'il s'agit d'une configuration sans lire notre
// documentation.
func TestEachFailureHasItsOwnExitCode(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args   []string
		entree string
		code   int
	}{
		"adresse invalide":  {[]string{"--email", "pas-une-adresse"}, secret + "\n", exit.DataErr},
		"secret trop court": {[]string{"--email", email}, "court\n", exit.DataErr},
		"adresse absente":   {[]string{}, secret + "\n", exit.Usage},
		"drapeau inconnu":   {[]string{"--courriel", email}, secret + "\n", exit.Usage},
		"secret vide":       {[]string{"--email", email}, "\n", exit.Usage},
		"entrée vide":       {[]string{"--email", email}, "", exit.Usage},
	}

	for nom, tc := range cases {
		commande, flux := nouvelleCommande(t, tc.entree)
		code := commande.Register(context.Background(), tc.args)
		if code != tc.code {
			t.Errorf("%s : attendu %d, obtenu %d — %s", nom, tc.code, code, flux.err.String())
		}
	}
}

// TestATakenAddressIsNotRetryable distingue l'état du serveur d'une panne.
//
// Une adresse déjà prise rend `DataErr` et non `Unavailable` : la requête est
// bien formée, mais réessayer ne changera rien tant que le compte existe. C'est
// exactement la distinction que la surface HTTP fait entre 409 et 503, rendue
// ici par un code de retour.
func TestATakenAddressIsNotRetryable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	commande, flux := nouvelleCommande(t, secret+"\n"+secret+"\n")

	if code := commande.Register(ctx, []string{"--email", email}); code != exit.OK {
		t.Fatalf("première inscription: code %d — %s", code, flux.err.String())
	}
	if code := commande.Register(ctx, []string{"--email", email}); code != exit.DataErr {
		t.Fatalf("adresse déjà prise : attendu %d, obtenu %d", exit.DataErr, code)
	}
	if strings.Contains(flux.err.String(), "condensé") {
		t.Fatalf("le condensé fuite dans le message de conflit : %q", flux.err.String())
	}
}

// TestSeedRefusesAnUnknownProfile : deny par défaut jusque dans un drapeau.
//
// Se rabattre sur le plus petit profil créerait trois comptes là où l'on en
// attendait vingt-cinq — et le défaut ne se verrait qu'au moment où une liste
// paginée cesserait de paginer, c'est-à-dire pendant une démonstration.
func TestSeedRefusesAnUnknownProfile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for _, profil := range []string{"perf", "prod", "DEV", ""} {
		commande, flux := nouvelleCommande(t, "")
		code := commande.Seed(ctx, []string{"--profile", profil})
		if code != exit.Usage {
			t.Errorf("profil %q : attendu %d, obtenu %d", profil, exit.Usage, code)
		}
		if flux.out.Len() != 0 {
			t.Errorf("profil %q : aucun compte ne doit être créé, obtenu %q", profil, flux.out.String())
		}
	}
}

// TestSeedGoesThroughTheUseCaseNotThroughSQL garde le chemin du peuplement.
//
// # La propriété qui compte
//
// Les comptes engendrés sont en tout point ceux qu'aurait créés la surface HTTP :
// adresse normalisée par le domaine, statut `pending`, mot de passe haché. Un
// jeu de données fabriqué en SQL produirait des comptes que le code n'aurait
// jamais laissé naître — et un test qui s'appuierait dessus validerait un état
// que la production ne peut pas atteindre.
//
// Le test le constate par le seul chemin observable depuis l'extérieur : la
// sortie du peuplement porte les mêmes trois champs que `register`, et le
// deuxième passage rend un CONFLIT — donc les comptes existent vraiment dans le
// magasin, ils n'ont pas été inventés.
func TestSeedGoesThroughTheUseCaseNotThroughSQL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	commande, flux := nouvelleCommande(t, "")

	if code := commande.Seed(ctx, []string{"--profile", "dev"}); code != exit.OK {
		t.Fatalf("peuplement: code %d — %s", code, flux.err.String())
	}

	lignes := strings.Split(strings.TrimSpace(flux.out.String()), "\n")
	if len(lignes) != 3 {
		t.Fatalf("le profil dev crée 3 comptes, obtenu %d ligne(s) : %q", len(lignes), flux.out.String())
	}
	for _, ligne := range lignes {
		if champs := strings.Split(ligne, "\t"); len(champs) != 3 || champs[2] != "pending" {
			t.Fatalf("un compte peuplé doit être identique à un compte inscrit : %q", ligne)
		}
	}

	// Rejouer le même profil doit CONFLIGER : la preuve que le premier passage a
	// bien écrit dans le magasin, par le cas d'usage.
	if code := commande.Seed(ctx, []string{"--profile", "dev"}); code != exit.DataErr {
		t.Fatalf("un second peuplement doit conflitger, obtenu %d", code)
	}
}
