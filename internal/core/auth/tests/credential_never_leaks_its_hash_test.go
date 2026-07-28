package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestCredentialNeverLeaksItsHash couvre les DEUX verbes de formatage.
//
// # Pourquoi deux, et pas un
//
// `%v` passe par `String()`, `%#v` par `GoString()`. Couvrir l'un laisse l'autre
// fuiter, et `%#v` est précisément ce qu'on écrit dans un journal de débogage —
// donc le jour d'un incident, donc le jour où les journaux partent chez un tiers.
//
// Un condensé Argon2id n'est pas un mot de passe, mais il se casse hors ligne :
// le publier transforme une fuite de journaux en fuite de comptes.
//
// ⚠️ Le test formate DÉLIBÉRÉMENT avec `%v` et `%#v` plutôt que d'appeler
// `String()` : c'est le chemin de fuite réel qui est éprouvé. Appeler la méthode
// laisserait le test vert si quelqu'un retirait le `Stringer`.
func TestCredentialNeverLeaksItsHash(t *testing.T) {
	t.Parallel()

	const hash = "$argon2id$v=19$m=65536,t=3,p=4$c2VsLWZhY3RpY2U$Y29uZGVuc2UtZmFjdGljZQ"

	subj, err := domain.NewSubject(subject)
	if err != nil {
		t.Fatalf("sujet: %v", err)
	}
	identity, err := domain.NewIdentity("id-1", subj, nil, time.Now())
	if err != nil {
		t.Fatalf("identité: %v", err)
	}
	credential, err := domain.NewCredential(identity, hash)
	if err != nil {
		t.Fatalf("créance: %v", err)
	}

	for _, rendered := range []string{
		fmt.Sprintf("%v", credential),  //nolint:gocritic // c'est le chemin de fuite testé
		fmt.Sprintf("%#v", credential), // par GoString — l'autre moitié du masque
		fmt.Sprintf("%s", credential),  //nolint:gocritic,staticcheck // idem, par String
		fmt.Sprint(credential),         //nolint:gocritic // idem, sans verbe
	} {
		if strings.Contains(rendered, hash) {
			t.Fatalf("le condensé fuite dans %q", rendered)
		}
		if !strings.Contains(rendered, "***") {
			t.Fatalf("le masque doit être visible dans %q", rendered)
		}
	}

	// Le condensé reste ACCESSIBLE, par un accesseur nommé : un accès se voit
	// alors en relecture, et se cherche en une commande.
	if credential.SecretHash() != hash {
		t.Fatal("le condensé doit rester accessible pour comparaison")
	}
}

// TestCredentialRefusesAnIncompleteAssembly garde les bornes du type.
//
// Deux `string` de suite — « le sujet » et « le condensé » — s'inversent
// silencieusement, et l'inversion produirait une comparaison qui réussit toujours.
// Le type existe pour rendre l'inversion impossible ; le refus garde ses bornes.
func TestCredentialRefusesAnIncompleteAssembly(t *testing.T) {
	t.Parallel()

	subj, err := domain.NewSubject(subject)
	if err != nil {
		t.Fatalf("sujet: %v", err)
	}
	identity, err := domain.NewIdentity("id-1", subj, nil, time.Now())
	if err != nil {
		t.Fatalf("identité: %v", err)
	}

	if _, err := domain.NewCredential(domain.Identity{}, "condensé"); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("sans identité : attendu ErrIncomplete, obtenu %v", err)
	}
	if _, err := domain.NewCredential(identity, ""); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("sans condensé : attendu ErrIncomplete, obtenu %v", err)
	}
}

// TestTokenComparesInConstantTime éprouve la propriété par son CONTRAT.
//
// La constance du temps ne se mesure pas honnêtement dans un test unitaire — un
// ordonnanceur, un ramasse-miettes ou une machine partagée y ajoutent plus de
// bruit que le canal recherché n'en produit de signal. Ce que le test garde, c'est
// la CORRECTION de `Equals` : le jour où quelqu'un remplacerait
// `subtle.ConstantTimeCompare` par `==` en pensant simplifier, la revue est le
// seul filet — et un test qui exige `Equals` d'exister empêche au moins que la
// méthode disparaisse au profit d'une comparaison directe chez l'appelant.
func TestTokenComparesInConstantTime(t *testing.T) {
	t.Parallel()

	const raw = "0123456789012345678901234567890123456789012"

	token, err := domain.NewToken(raw)
	if err != nil {
		t.Fatalf("jeton: %v", err)
	}
	same, err := domain.NewToken(raw)
	if err != nil {
		t.Fatalf("jeton: %v", err)
	}
	other, err := domain.NewToken(strings.Repeat("a", 43))
	if err != nil {
		t.Fatalf("jeton: %v", err)
	}

	if !token.Equals(same) {
		t.Error("deux jetons identiques doivent être égaux")
	}
	if token.Equals(other) {
		t.Error("deux jetons différents ne doivent pas être égaux")
	}
	if token.Equals(domain.Token{}) {
		t.Error("un jeton non construit ne doit égaler aucun jeton")
	}

	// Un jeton trop court est refusé À LA CONSTRUCTION : la borne vit dans le
	// domaine, donc elle vaut quel que soit le port qui a produit la chaîne.
	if _, err := domain.NewToken(strings.Repeat("a", 42)); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("jeton de 42 caractères : attendu ErrIncomplete, obtenu %v", err)
	}
}
