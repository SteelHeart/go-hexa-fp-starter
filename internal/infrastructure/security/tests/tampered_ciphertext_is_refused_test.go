package tests

import (
	"encoding/base64"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestTamperedCiphertextIsRefused : une altération d'un seul bit fait échouer le
// déchiffrement.
//
// C'est la différence entre chiffrer et AUTHENTIFIER. Un mode sans authentification
// déchiffrerait sans broncher un message modifié, et rendrait un clair silencieusement
// faux — un montant, un droit d'accès, un identifiant. GCM refuse, et ce test le
// prouve plutôt que de le supposer.
func TestTamperedCiphertextIsRefused(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(cleAES())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	chiffre, err := cipher.Encrypt([]byte("une donnée personnelle"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Le chiffré voyage encodé en base64 : on altère les OCTETS puis on ré-encode,
	// pour viser le message lui-même et non sa représentation textuelle.
	brut, err := base64.RawURLEncoding.DecodeString(chiffre)
	if err != nil {
		t.Fatalf("le chiffré doit être décodable: %v", err)
	}

	cases := map[string]func([]byte) []byte{
		"un bit du message": func(c []byte) []byte { d := clone(c); d[len(d)-1] ^= 1; return d },
		"un bit du nonce":   func(c []byte) []byte { d := clone(c); d[0] ^= 1; return d },
		"message tronqué":   func(c []byte) []byte { return c[:len(c)-1] },
		"message vide":      func([]byte) []byte { return nil },
		"trop court":        func([]byte) []byte { return []byte{1, 2, 3} },
	}

	for name, altere := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			falsifie := base64.RawURLEncoding.EncodeToString(altere(brut))
			if _, err := cipher.Decrypt(falsifie); err == nil {
				t.Error("un message altéré doit être REFUSÉ, jamais déchiffré")
			}
		})
	}

	// Un chiffré qui n'est même pas du base64 doit être refusé lui aussi.
	if _, err := cipher.Decrypt("!!! pas du base64 !!!"); err == nil {
		t.Error("un chiffré illisible doit être refusé")
	}
}
