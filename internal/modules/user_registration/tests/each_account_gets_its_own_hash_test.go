package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEachAccountGetsItsOwnHash : le mot de passe RÉEL est haché, pas sa forme
// masquée.
//
// # Le défaut que ce test attrape
//
// `RawPassword.String()` rend « [redacted] » — c'est la protection qui empêche
// une journalisation accidentelle de faire fuir un mot de passe. Un adaptateur
// de hachage qui appelle `String()` au lieu de `Expose()` compile, passe la
// revue, et hache la MÊME chaîne pour tous les comptes.
//
// Conséquence : tous les utilisateurs partagent un condensé, et n'importe quel
// mot de passe ouvre n'importe quel compte. La faille est totale et parfaitement
// invisible — l'inscription réussit, la connexion réussit, rien ne semble
// anormal.
//
// Ce défaut a réellement été écrit pendant le développement de cet adaptateur.
// Ce test existe pour qu'il ne puisse pas revenir.
func TestEachAccountGetsItsOwnHash(t *testing.T) {
	t.Parallel()

	// Coût minimal : on éprouve le CÂBLAGE, pas la robustesse d'Argon2, qui a
	// ses propres tests dans internal/infrastructure/security.
	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  1 << 10,
		Iterations: 1,
		Threads:    1,
	})
	hashPort := hashing.New(hasher)

	publisher := &spyPublisher{}
	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: hashPort,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err != nil {
		t.Fatalf("montage du module: %v", err)
	}

	first := register(t, mod, "alice@example.com", "correct cheval batterie agrafe")
	second := register(t, mod, "bob@example.com", "une tout autre phrase secrete")

	assertRealPasswordWasHashed(t, hasher, first, "correct cheval batterie agrafe")
	assertRealPasswordWasHashed(t, hasher, second, "une tout autre phrase secrete")

	if first.PasswordHash.String() == second.PasswordHash.String() {
		t.Fatal("deux comptes partagent le même condensé — la valeur masquée a été hachée à la place du mot de passe")
	}
}

// assertRealPasswordWasHashed vérifie que le condensé correspond au mot de passe
// réellement fourni.
func assertRealPasswordWasHashed(
	t *testing.T,
	hasher security.Hasher,
	user domain.User,
	plain string,
) {
	t.Helper()

	ok, err := hasher.Verify(plain, user.PasswordHash.String())
	if err != nil {
		t.Fatalf("vérification du condensé de %s: %v", user.Email, err)
	}
	if !ok {
		t.Errorf("le condensé de %s ne correspond pas au mot de passe fourni", user.Email)
	}
	if verified, _ := hasher.Verify("[redacted]", user.PasswordHash.String()); verified {
		t.Errorf("le condensé de %s correspond à « [redacted] » : c'est la valeur MASQUÉE qui a été hachée", user.Email)
	}
}
