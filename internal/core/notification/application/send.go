// Package application compose les cas d'usage de la notification.
//
// Il ne journalise pas et ne lit pas l'horloge : il rend compte par ses retours.
// C'est ce qui le garde testable sans analyser de journaux.
//
// # Carte des fichiers
//
//	send.go   acheminer un message déjà rendu
package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/ports"
)

// Deps porte les ports dont les cas d'usage ont besoin.
//
// Tous sont des types fonction : en test, chacun est une closure de trois
// lignes, et aucune bibliothèque de simulacre n'est nécessaire — donc aucune
// n'est autorisée (rules/dependances.md).
type Deps struct {
	Deliver ports.Deliver
}

// NewSend compose l'acheminement d'un message.
//
// # Deux lignes utiles, et c'est le point
//
// Valider, remettre. Il n'y a nulle part où glisser un gabarit, une traduction,
// ou une politique de réessai — les trois sont des décisions d'application, pas
// de socle, et les enfouir ici les imposerait à tout le monde.
//
// La revalidation n'est pas de la paranoïa : `domain.Message` a des champs
// exportés, donc un appelant peut en fabriquer un sans passer par `NewMessage`.
// Le cas d'usage est le dernier endroit où l'on peut encore refuser avant qu'une
// adresse vide n'atteigne un fournisseur — où elle deviendrait un rejet facturé
// et journalisé chez un tiers.
func NewSend(deps Deps) ports.Send {
	return func(ctx context.Context, message domain.Message) error {
		validated, err := domain.NewMessage(
			message.Channel, message.To, message.Subject, message.Body)
		if err != nil {
			return fmt.Errorf("message: %w", err)
		}
		if err := deps.Deliver(ctx, validated); err != nil {
			return fmt.Errorf("remise au fournisseur: %w", err)
		}
		return nil
	}
}
