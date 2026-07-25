package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestConcurrentRegistrationsNeverDuplicateAnAddress : deux inscriptions
// simultanées sur la même adresse, une seule gagne.
//
// # Pourquoi ce test existe
//
// Le cas d'usage vérifie la disponibilité PUIS écrit. Entre les deux, il y a une
// fenêtre : deux requêtes concurrentes la franchissent toutes les deux, et
// concluent toutes les deux que l'adresse est libre. C'est le défaut classique
// « vérifier puis agir », et il ne se manifeste jamais en développement — il
// apparaît sous charge, en production, sous forme de deux comptes pour une même
// personne.
//
// Seul le MAGASIN peut trancher, parce que lui seul détient le verrou. Ce test
// verrouille cette garantie pour le pilote `memory` ; le pilote SQL l'obtiendra
// de sa contrainte d'unicité, et devra passer le même test.
func TestConcurrentRegistrationsNeverDuplicateAnAddress(t *testing.T) {
	t.Parallel()

	const attempts = 16

	publisher := &spyPublisher{}
	mod := newModule(t, publisher)

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
	)

	start := make(chan struct{})
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // relâche tout le monde en même temps

			_, failure, ok := mod.Register(context.Background(), domain.RegistrationCommand{
				Email:    "course@example.com",
				Password: validPassword,
			}).Get()

			mu.Lock()
			defer mu.Unlock()
			switch {
			case ok:
				succeeded++
			case failure.Code == domain.CodeEmailAlreadyExists:
				conflicts++
			default:
				t.Errorf("échec inattendu: %v", failure)
			}
		}()
	}
	close(start)
	wg.Wait()

	if succeeded != 1 {
		t.Errorf("inscriptions réussies = %d, attendu exactement 1 — l'unicité de l'adresse n'est pas tenue", succeeded)
	}
	if conflicts != attempts-1 {
		t.Errorf("conflits = %d, attendu %d — les perdants doivent être refusés explicitement", conflicts, attempts-1)
	}
}
