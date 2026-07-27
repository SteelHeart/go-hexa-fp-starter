package tests

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestMalformedHashIsRefused : un condensé illisible rend une ERREUR, pas un refus.
//
// Un condensé est censé venir de notre propre base — mais une reprise de données,
// une colonne tronquée ou un magasin externe suffisent à y glisser autre chose. Le
// traiter comme « mauvais mot de passe » masquerait une corruption de données
// derrière des échecs de connexion que personne ne relierait entre eux.
func TestMalformedHashIsRefused(t *testing.T) {
	t.Parallel()

	valide := hash(t, "correct cheval batterie agrafe")
	parts := strings.Split(valide, "$")

	cases := map[string]string{
		"vide":                  "",
		"sans structure":        "pas un condensé",
		"algorithme inconnu":    strings.Replace(valide, "argon2id", "bcrypt", 1),
		"version inattendue":    strings.Replace(valide, "v=19", "v=13", 1),
		"paramètres illisibles": "$argon2id$v=19$memoire=beaucoup$" + parts[4] + "$" + parts[5],
		"sel non base64":        "$argon2id$v=19$" + parts[3] + "$!!!$" + parts[5],
		"clé non base64":        "$argon2id$v=19$" + parts[3] + "$" + parts[4] + "$!!!",
		"segment manquant":      strings.Join(parts[:5], "$"),
	}

	hasher := newHasher()
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ok, err := hasher.Verify("peu importe", encoded)
			if !errors.Is(err, security.ErrInvalidHash) {
				t.Errorf("erreur = %v, attendu ErrInvalidHash", err)
			}
			if ok {
				t.Error("un condensé illisible ne doit JAMAIS valider un mot de passe")
			}
		})
	}

	// Un condensé annonçant une clé absurde est refusé AVANT d'atteindre argon2.
	// Sans cette borne, la longueur annoncée pilote directement une allocation :
	// un « condensé » forgé ferait tomber le processus à la première vérification.
	geante := "$argon2id$v=19$" + parts[3] + "$" + parts[4] + "$" +
		base64.RawStdEncoding.EncodeToString(make([]byte, 1<<20))
	if ok, err := hasher.Verify("peu importe", geante); !errors.Is(err, security.ErrInvalidHash) || ok {
		t.Errorf("une clé démesurée doit être refusée: ok=%v err=%v", ok, err)
	}
}
