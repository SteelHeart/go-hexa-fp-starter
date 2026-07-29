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

// welcome builds the handler for `user.registered.v1`.
//
// # This is WHERE the business meets the core, and nowhere else
//
// `notification` knows nothing of `user_registration`, of registration, nor of
// the word "welcome"; `user_registration` knows nothing of an email going out.
// The link between the two is a POLICY — "when an account is created, welcome
// it" — and a policy belongs to the application, not to the starter.
//
// It therefore lives in the composition root, which is allowed to know
// everything (ADR 004). Changing one's mind — no longer sending, sending
// something else, adding a second effect — touches no module.
//
// # The published contract, never the domain type
//
// The payload is decoded into `contract.UserRegisteredV1`: primitive types,
// readable by a consumer written in another language or deployed separately.
// Decoding into a `user_registration/domain` type would recreate the coupling
// the published language serves to avoid.
func welcome(mod notification.Module) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		var fact contract.UserRegisteredV1
		if err := json.Unmarshal(env.Payload, &fact); err != nil {
			// An unreadable payload is not replayed: it will be just as
			// unreadable on the next attempt. The error still travels up — it
			// is up to the dispatcher to give up after N attempts, not up to
			// this handler to decide on its own to forget an event.
			return fmt.Errorf("unreadable payload for %s: %w", env.Type, err)
		}

		message, err := welcomeMessage(fact)
		if err != nil {
			return err
		}
		if err := mod.Send(ctx, message); err != nil {
			return fmt.Errorf("sending the welcome email: %w", err)
		}
		return nil
	}
}

// welcomeMessage returns the email to send.
//
// # No template engine, and that is a decision
//
// The content is written here, in plain text. A template engine would impose
// its syntax on every application built on this starter, and rendering a text
// is not a problem the starter has to solve. The day an application wants
// i18n, it replaces this function — without touching a module.
//
// ⚠️ This body contains NO confirmation link, deliberately. The account is
// born `pending` and address confirmation is not written: fabricating a link
// here would produce a URL that serves nothing, and a user clicking on a dead
// page is worse than a user who waits.
func welcomeMessage(fact contract.UserRegisteredV1) (notifdomain.Message, error) {
	recipient, err := notifdomain.NewRecipient(fact.Email)
	if err != nil {
		return notifdomain.Message{}, fmt.Errorf("recipient for %s: %w", contract.EventUserRegisteredV1, err)
	}

	message, err := notifdomain.NewMessage(
		notifdomain.ChannelEmail,
		recipient,
		"Bienvenue",
		"Votre compte a bien été créé. Il attend la confirmation de votre adresse.",
	)
	if err != nil {
		return notifdomain.Message{}, fmt.Errorf("welcome message: %w", err)
	}
	return message, nil
}
