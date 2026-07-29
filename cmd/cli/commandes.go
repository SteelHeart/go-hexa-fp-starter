package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	usercli "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/cli"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// errSeedEnProduction refuse de peupler une production.
//
// # Pourquoi un refus explicite, et pourquoi il compte
//
// `seed` crée des comptes de démonstration. En production, cela veut dire des
// identifiants connus de tous ceux qui ont lu le dépôt — donc une porte
// d'entrée, ouverte par une commande qu'on croyait inoffensive parce qu'elle
// l'est partout ailleurs.
//
// Le refus est un `sysexits.NoPerm` : ce n'est ni une panne, ni une erreur de
// saisie, c'est un « non » assumé. La distinction compte pour un script, qui ne
// doit ni réessayer ni alerter comme sur un incident.
var errSeedEnProduction = errors.New(
	"seed refusé hors développement et test : des comptes de démonstration en production sont une porte d'entrée")

// commandRegister branche la surface d'inscription.
func commandRegister(ctx context.Context, args []string) int {
	assemblage, err := composer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		return exit.Config
	}
	defer assemblage.conn.Close()

	commande := usercli.Command{Module: assemblage.users, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	return commande.Register(ctx, args)
}

// commandSeed peuple le socle PAR LES CAS D'USAGE.
//
// Jamais d'`INSERT` direct : un jeu de données fabriqué en SQL contourne les
// règles du domaine, et produit des comptes que le code n'aurait jamais laissé
// naître — mots de passe non hachés, adresses non normalisées, statuts
// impossibles. Le jour où un test s'appuie dessus, il valide un état que la
// production ne peut pas atteindre.
func commandSeed(ctx context.Context, args []string) int {
	assemblage, err := composer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		return exit.Config
	}
	defer assemblage.conn.Close()

	if err := autoriserSeed(assemblage.cfg.App.Env); err != nil {
		fmt.Fprintf(os.Stderr, "refus: %v\n", err)
		return exit.NoPerm
	}

	commande := usercli.Command{Module: assemblage.users, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	return commande.Seed(ctx, args)
}

// autoriserSeed refuse le peuplement hors des environnements locaux.
//
// La décision est prise sur `app.env` RÉSOLU, pas sur une variable lue à part :
// c'est la même valeur que celle qui gouverne le durcissement réseau et
// l'amorçage de l'authentification. Deux sources pour une même question finissent
// par se contredire, et c'est celle qui autorise qui l'emporte.
func autoriserSeed(env config.Environment) error {
	if !env.IsLocal() {
		return fmt.Errorf("%w (env=%s)", errSeedEnProduction, env)
	}
	return nil
}

// commandHealth vérifie que ce binaire POURRAIT démarrer.
//
// # Ce qu'elle vérifie réellement
//
// Que le catalogue s'assemble, que la configuration se charge et se valide, que
// les connexions réclamées s'ouvrent, et que les modules se montent. C'est
// exactement ce qu'un démarrage fait avant d'écouter — donc un `health` vert
// signifie « ce déploiement démarrerait ».
//
// Elle ne remplace PAS `/readyz` : celle-ci interroge un service en marche,
// celle-là interroge une configuration. Les deux répondent à des questions
// différentes, et confondre les deux ferait croire qu'un `health` vert prouve
// qu'un serveur répond.
func commandHealth(ctx context.Context) int {
	assemblage, err := composer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		return exit.Config
	}
	defer assemblage.conn.Close()

	fmt.Fprintf(os.Stdout, "ok\tenv=%s\tbase=%t\tcache=%t\n",
		assemblage.cfg.App.Env, assemblage.conn.Pool != nil, assemblage.conn.Cache != nil)
	return exit.OK
}
