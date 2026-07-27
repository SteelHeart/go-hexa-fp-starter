package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestTraversalKeysAreRefusedOnRead : à la LECTURE, la clé n'a pas été dérivée par
// SafeKey — elle vient d'une URL, donc d'un inconnu.
//
// C'est le trou que la protection à l'écriture ne couvre pas : hacher les noms
// entrants ne sert à rien si `GET /files/../../etc/passwd` est servi. La
// vérification a donc lieu AVANT de toucher au disque, pour la lecture comme pour
// la suppression.
func TestTraversalKeysAreRefusedOnRead(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	keys := map[string]domain.Key{
		"remontée":           "../secret.pem",
		"remontée profonde":  "../../../etc/passwd",
		"absolue":            "/etc/passwd",
		"remontée au milieu": "ab/../../cd",
		"parent nu":          "..",
		"vide":               "",
		"Windows":            "..\\secret.pem",
	}

	for name, key := range keys {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := mod.Get(ctx, key); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("Get(%q) = %v, attendu ErrUnsafeName", key, err)
			}
			if err := mod.Delete(ctx, key); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("Delete(%q) = %v, attendu ErrUnsafeName", key, err)
			}
		})
	}
}
