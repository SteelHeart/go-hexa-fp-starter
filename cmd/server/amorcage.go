package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// amorcerAuthentification crée le compte d'amorçage et l'ANNONCE, une fois.
//
// # Pourquoi le secret est écrit dans le journal
//
// Parce que l'alternative est pire. Un mot de passe par défaut dans un fichier
// versionné est un mot de passe déployé : personne ne le change avant
// l'incident. Un secret engendré, affiché une seule fois, et qui disparaît au
// redémarrage ne peut pas suivre le projet en production.
//
// Le compromis est BORNÉ : `auth.Bootstrap` ne crée rien hors `development` et
// `test`, et le refus a son test. Ici, le journal ne peut donc contenir un
// secret que sur une machine de développement.
//
// # Pourquoi ce n'est pas au module de le journaliser
//
// Un module noyau ne journalise pas — il rend compte. C'est cette frontière qui
// garantit qu'un secret ne part pas dans un collecteur d'observabilité parce
// qu'un module a cru bien faire. La décision d'écrire est prise ici, une seule
// fois, à un endroit qu'on relit.
func amorcerAuthentification(
	ctx context.Context, mod auth.Module, env config.Environment, logger *slog.Logger,
) error {
	report, err := auth.Bootstrap(ctx, mod, env)
	if err != nil {
		return fmt.Errorf("amorçage de l'authentification: %w", err)
	}
	if !report.Created {
		return nil
	}

	logger.WarnContext(ctx,
		"COMPTE D'AMORÇAGE CRÉÉ — développement uniquement, affiché une seule fois",
		slog.String("sujet", report.Subject),
		slog.String("secret", report.Secret),
		slog.String("note", "perdu au redémarrage : le pilote memory ne conserve rien"),
	)
	return nil
}
