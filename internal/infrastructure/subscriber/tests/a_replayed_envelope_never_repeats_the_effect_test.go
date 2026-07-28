package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// TestAReplayedEnvelopeNeverRepeatsTheEffect est le TÉMOIN de l'issue #9.
//
// # Ce qu'il constate
//
// La même enveloppe, remise dix fois, ne produit l'effet QU'UNE fois — et les
// neuf autres remises acquittent sans erreur, parce qu'un rejeu reconnu n'est
// pas une panne.
//
// # Pourquoi ce test est indispensable
//
// Tous les transports d'ici sont « au moins une fois ». La même enveloppe arrive
// donc deux fois dès qu'un accusé de réception se perd — ce qui est banal, pas
// exceptionnel. Sans ce garde, un courriel de bienvenue part deux fois : le
// symptôme visible. Le jour où un consommateur débitera une carte, ce sera le
// débit, et il ne sera pas visible avant le relevé.
func TestAReplayedEnvelopeNeverRepeatsTheEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effet := &compteur{}
	handler := subscriber.Once(newGuard(t), effet.handler())
	env := envelope("evt-1")

	for essai := 1; essai <= 10; essai++ {
		if err := handler(ctx, env); err != nil {
			t.Fatalf("remise n°%d : un rejeu reconnu n'est pas une panne — %v", essai, err)
		}
	}

	if effet.total() != 1 {
		t.Fatalf("l'effet a eu lieu %d fois, attendu exactement 1", effet.total())
	}
}

// TestTwoDistinctEnvelopesBothProduceTheirEffect garde l'autre moitié.
//
// Un garde qui bloquerait TOUT serait vert au test précédent et parfaitement
// inutile. C'est la forme la plus courante de garde cassé : celui qui refuse
// tout ressemble en tout point à celui qui ne laisse passer que ce qu'il faut.
func TestTwoDistinctEnvelopesBothProduceTheirEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effet := &compteur{}
	handler := subscriber.Once(newGuard(t), effet.handler())

	for _, id := range []string{"evt-1", "evt-2", "evt-3"} {
		if err := handler(ctx, envelope(id)); err != nil {
			t.Fatalf("enveloppe %s : %v", id, err)
		}
	}

	if effet.total() != 3 {
		t.Fatalf("trois enveloppes distinctes ont produit %d effets, attendu 3", effet.total())
	}
}

// TestConcurrentDeliveriesProduceASingleEffect : le rejeu SIMULTANÉ.
//
// Le cas séquentiel passerait même avec une vérification naïve « déjà vu ? »
// posée avant l'exécution. C'est la remise simultanée qui fait apparaître la
// fenêtre — et derrière deux répliques, elle est la norme, pas l'exception.
//
// Les remises perdantes rendent une ERREUR, et c'est voulu : une autre réplique
// traite l'enveloppe et peut encore échouer. Les acquitter perdrait l'événement
// si l'autre abandonne.
func TestConcurrentDeliveriesProduceASingleEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effet := &compteur{}
	handler := subscriber.Once(newGuard(t), effet.handler())
	env := envelope("evt-simultane")

	const remises = 16
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		reussi int
	)

	wg.Add(remises)
	for range remises {
		go func() {
			defer wg.Done()
			if err := handler(ctx, env); err == nil {
				mu.Lock()
				reussi++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if effet.total() != 1 {
		t.Fatalf("%d remises simultanées ont produit %d effets, attendu 1", remises, effet.total())
	}
	if reussi < 1 {
		t.Fatal("aucune remise n'a abouti : l'événement serait perdu")
	}
}
