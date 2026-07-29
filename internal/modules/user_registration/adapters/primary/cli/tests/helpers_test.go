// Package tests éprouve la SURFACE LIGNE DE COMMANDE du module d'inscription.
//
// # Ce qu'ils vérifient
//
// La TRADUCTION, et rien d'autre : des arguments vers une commande de domaine,
// puis d'un résultat vers une sortie et un CODE DE RETOUR. Le domaine et le
// pipeline ont leurs propres tests.
//
// Le code de retour compte autant que le texte : un binaire de ligne de commande
// est appelé par des scripts plus souvent que par des humains, et un script lit
// le code pour décider s'il faut réessayer.
package tests

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	usercli "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/cli"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

const (
	// email et secret satisfont les bornes du domaine.
	email  = "alice@example.com"
	secret = "correct cheval batterie agrafe"
)

// sortie retient ce que la commande a écrit, séparément.
//
// Les deux flux sont distincts DÉLIBÉRÉMENT : un binaire qui écrit ses erreurs
// sur la sortie standard casse tout script qui redirige l'une sans l'autre, et
// c'est le genre de défaut qu'aucun test ne voit s'il fusionne les deux tampons.
type sortie struct {
	out *bytes.Buffer
	err *bytes.Buffer
}

// nouvelleCommande monte la surface sur le pilote par défaut du module.
//
// Chaque test a donc son propre magasin, vierge, et peut s'exécuter en parallèle
// sans interférer.
func nouvelleCommande(t *testing.T, entree string) (usercli.Command, *sortie) {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fauxHachage,
		PublishEvent: publicationInerte,
		GenerateID:   func() domain.UserID { return "019f9b46-3aec-735a-977d-129192ef130f" },
		Now:          func() time.Time { return time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("montage du module: %v", err)
	}

	flux := &sortie{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	return usercli.Command{
		Module: mod,
		In:     strings.NewReader(entree),
		Out:    flux.out,
		Err:    flux.err,
	}, flux
}

// fauxHachage est un hachage FACTICE, instantané.
//
// Argon2id est délibérément lent : le payer ici ferait durer chaque test des
// dizaines de millisecondes sans rien éprouver de plus. Ce qui est testé, c'est
// la traduction, pas la solidité du condensé.
func fauxHachage(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("condensé:" + password.String()))
}

// publicationInerte accepte l'événement sans rien en faire.
func publicationInerte(
	_ context.Context, _, _ string, _ any,
) result.Result[domain.Ack, domain.Error] {
	return result.Ok[domain.Ack, domain.Error](domain.Ack{})
}
