package main

import (
	"context"
	"encoding/json"
	"fmt"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	notifdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// bienvenue construit le gestionnaire de `user.registered.v1`.
//
// # C'est ICI que le métier rencontre le noyau, et nulle part ailleurs
//
// `notification` ignore `user_registration`, l'inscription et le mot
// « bienvenue » ; `user_registration` ignore qu'un courriel part. Le lien entre
// les deux est une POLITIQUE — « quand un compte est créé, souhaiter la
// bienvenue » — et une politique appartient à l'application, pas au socle.
//
// Elle vit donc dans le composition root, qui a le droit de tout connaître
// (ADR 004). Changer d'avis — ne plus envoyer, envoyer autre chose, ajouter un
// second effet — ne touche aucun module.
//
// # Le contrat publié, jamais le type du domaine
//
// La charge utile est décodée en `contract.UserRegisteredV1` : des types
// primitifs, lisibles par un consommateur écrit dans un autre langage ou déployé
// séparément. Décoder vers un type de `user_registration/domain` recréerait le
// couplage que le langage publié sert à éviter.
func bienvenue(mod notification.Module) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		var fait contract.UserRegisteredV1
		if err := json.Unmarshal(env.Payload, &fait); err != nil {
			// Une charge utile illisible ne se rejoue pas : elle sera tout aussi
			// illisible au prochain essai. L'erreur remonte quand même — c'est
			// au dépileur d'abandonner après N tentatives, pas à ce
			// gestionnaire de décider seul d'oublier un événement.
			return fmt.Errorf("charge utile de %s illisible: %w", env.Type, err)
		}

		message, err := messageDeBienvenue(fait)
		if err != nil {
			return err
		}
		if err := mod.Send(ctx, message); err != nil {
			return fmt.Errorf("envoi du courriel de bienvenue: %w", err)
		}
		return nil
	}
}

// messageDeBienvenue rend le courriel à envoyer.
//
// # Aucun gabarit, et c'est une décision
//
// Le contenu est écrit ici, en clair. Un moteur de gabarits imposerait sa
// syntaxe à toute application bâtie sur ce socle, et le rendu d'un texte n'est
// pas un problème que le socle a à résoudre. Le jour où une application veut de
// l'i18n, elle remplace cette fonction — sans toucher un module.
//
// ⚠️ Ce corps ne contient AUCUN lien de confirmation, délibérément. Le compte
// naît `pending` et la confirmation d'adresse n'est pas écrite : fabriquer un
// lien ici produirait une URL que rien ne sert, et un utilisateur qui clique sur
// une page morte est pire qu'un utilisateur qui attend.
func messageDeBienvenue(fait contract.UserRegisteredV1) (notifdomain.Message, error) {
	destinataire, err := notifdomain.NewRecipient(fait.Email)
	if err != nil {
		return notifdomain.Message{}, fmt.Errorf("destinataire de %s: %w", contract.EventUserRegisteredV1, err)
	}

	message, err := notifdomain.NewMessage(
		notifdomain.ChannelEmail,
		destinataire,
		"Bienvenue",
		"Votre compte a bien été créé. Il attend la confirmation de votre adresse.",
	)
	if err != nil {
		return notifdomain.Message{}, fmt.Errorf("message de bienvenue: %w", err)
	}
	return message, nil
}
