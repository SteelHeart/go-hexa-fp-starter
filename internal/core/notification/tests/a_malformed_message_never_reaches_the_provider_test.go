package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
)

// TestAMalformedMessageNeverReachesTheProvider garde le refus EN AMONT.
//
// # Pourquoi le cas d'usage revalide ce que le domaine sait déjà
//
// `domain.Message` a des champs EXPORTÉS : un appelant peut donc en fabriquer un
// sans passer par `NewMessage`, et rien dans le type ne l'en empêche. Le cas
// d'usage est le dernier endroit où l'on peut encore refuser avant qu'une
// adresse vide n'atteigne un fournisseur — où elle deviendrait un rejet facturé
// et journalisé chez un tiers.
//
// Le test construit délibérément des messages À LA MAIN, exactement comme le
// ferait un appelant pressé.
func TestAMalformedMessageNeverReachesTheProvider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, j := newModule(t, nil)

	valide := message(t)
	cases := map[string]domain.Message{
		"sans destinataire": {Channel: domain.ChannelEmail, Subject: subject, Body: body},
		"sans sujet":        {Channel: domain.ChannelEmail, To: valide.To, Body: body},
		"sujet en espaces":  {Channel: domain.ChannelEmail, To: valide.To, Subject: "   ", Body: body},
		"sans corps":        {Channel: domain.ChannelEmail, To: valide.To, Subject: subject},
		"corps en espaces":  {Channel: domain.ChannelEmail, To: valide.To, Subject: subject, Body: "  \n "},
		"message nu":        {},
	}

	for name, msg := range cases {
		err := mod.Send(ctx, msg)
		if !errors.Is(err, domain.ErrIncomplete) && !errors.Is(err, domain.ErrUnknownChannel) {
			t.Errorf("%s : attendu un refus du domaine, obtenu %v", name, err)
		}
	}

	if j.texte() != "" {
		t.Fatalf("aucun message refusé ne doit atteindre le pilote : %s", j.texte())
	}
}

// TestAnUnservedChannelIsRefusedNeverSubstituted : deny par défaut sur le canal.
//
// Se rabattre sur le courriel parce que le SMS n'est pas livré enverrait un code
// de vérification à la mauvaise adresse — un canal choisi précisément parce que
// l'autre ne convenait pas. Le catalogue dit ce qui EXISTE, et le domaine refuse
// le reste.
func TestAnUnservedChannelIsRefusedNeverSubstituted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, j := newModule(t, nil)
	valide := message(t)

	for _, canal := range []domain.Channel{"sms", "push", "webhook", "", "EMAIL"} {
		err := mod.Send(ctx, domain.Message{
			Channel: canal, To: valide.To, Subject: subject, Body: body,
		})
		if !errors.Is(err, domain.ErrUnknownChannel) {
			t.Errorf("canal %q : attendu ErrUnknownChannel, obtenu %v", canal, err)
		}
	}

	if j.texte() != "" {
		t.Fatalf("aucun canal refusé ne doit atteindre le pilote : %s", j.texte())
	}
}

// TestRecipientIsNormalisedSoDeduplicationHolds garde la forme canonique.
//
// `Alice@Example.COM ` et `alice@example.com` sont UNE adresse. Sans
// normalisation, une déduplication d'envois laisserait passer le doublon — et le
// destinataire recevrait deux fois le même courriel de bienvenue, ce qui est
// exactement le symptôme que l'idempotence est censée supprimer.
func TestRecipientIsNormalisedSoDeduplicationHolds(t *testing.T) {
	t.Parallel()

	for _, variante := range []string{"  Alice@Example.COM ", "ALICE@EXAMPLE.COM", "alice@example.com"} {
		destinataire, err := domain.NewRecipient(variante)
		if err != nil {
			t.Fatalf("adresse %q refusée: %v", variante, err)
		}
		if destinataire.String() != recipient {
			t.Errorf("adresse %q normalisée en %q, attendu %q", variante, destinataire.String(), recipient)
		}
	}
}

// TestAMalformedRecipientIsRefused garde les bornes de l'adresse.
//
// La validation est délibérément MINIMALE — pas d'expression rationnelle
// « conforme RFC 5322 ». Elles rejettent des adresses valides (apostrophes,
// accents, sous-adresses `+`), et un destinataire refusé à tort ne reçoit jamais
// rien sans que personne s'en aperçoive. Ce test garde donc les refus
// INCONTESTABLES, et vérifie que les formes exotiques passent.
func TestAMalformedRecipientIsRefused(t *testing.T) {
	t.Parallel()

	refusees := []string{"", "   ", "sans-arobase", "@example.com", "alice@", "a@b@c.com", "ali ce@x.com"}
	for _, raw := range refusees {
		if _, err := domain.NewRecipient(raw); !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("adresse %q : attendu ErrIncomplete, obtenu %v", raw, err)
		}
	}

	acceptees := []string{"alice+facture@example.com", "l'éve@example.com", "a@b.co", "prénom.nom@sous.domaine.fr"}
	for _, raw := range acceptees {
		if _, err := domain.NewRecipient(raw); err != nil {
			t.Errorf("adresse %q valide refusée : %v", raw, err)
		}
	}
}
