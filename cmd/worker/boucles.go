package main

import (
	"context"
	"fmt"
)

// boucler tient les DEUX boucles du dépileur jusqu'à l'annulation.
//
// # Pourquoi deux et pas une
//
// Le dépileur sort les messages de l'outbox et les publie ; le consommateur
// reçoit les enveloppes et déclenche les effets. Avec le relais `inproc` les
// deux vivent dans le même processus, et l'une remplit ce que l'autre vide.
//
// # La panne de l'une doit arrêter l'autre
//
// Sans cela, un consommateur mort laisserait le dépileur publier vers personne —
// et un dépileur qui publie vers personne ressemble EN TOUT POINT à un dépileur
// qui fonctionne : les messages sont marqués publiés, l'outbox se vide, les
// métriques sont vertes, et aucun effet n'a lieu.
//
// Le contexte est donc annulé dès la première sortie, quelle qu'elle soit, et
// l'erreur remontée est celle qui a provoqué l'arrêt.
func (w worker) boucler(ctx context.Context) error {
	ctx, arreter := context.WithCancel(ctx)
	defer arreter()

	// Tampon de 2 : les deux boucles doivent pouvoir écrire même si personne ne
	// lit plus. Sans tampon, la seconde resterait bloquée pour toujours sur
	// l'envoi, et l'arrêt propre n'en serait plus un.
	sorties := make(chan error, 2)

	go func() {
		if err := w.dispatch.Run(ctx); err != nil {
			sorties <- fmt.Errorf("dépileur: %w", err)
			return
		}
		sorties <- nil
	}()

	go func() {
		if err := w.consume.Run(ctx); err != nil {
			sorties <- fmt.Errorf("consommateur d'événements: %w", err)
			return
		}
		sorties <- nil
	}()

	// La PREMIÈRE sortie décide : elle annule le contexte, ce qui fait sortir
	// l'autre boucle. On attend quand même les deux, pour ne pas quitter
	// pendant qu'une réservation est encore ouverte.
	premiere := <-sorties
	arreter()
	seconde := <-sorties

	if premiere != nil {
		return premiere
	}
	return seconde
}
