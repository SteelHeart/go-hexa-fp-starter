package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Register inscrit un compte et rend un code de sortie `sysexits`.
//
// # Le MÊME port que la surface HTTP
//
// `POST /v1/users` et cette commande appellent tous deux `mod.Register`. Le
// module ne sait pas laquelle l'appelle, et il n'a rien fallu lui ajouter pour
// qu'il serve les deux. C'est la propriété n°2 du socle, mesurée plutôt
// qu'énoncée.
//
// # Le mot de passe n'est PAS un drapeau
//
// Il est lu sur l'entrée standard. Un `--password` figure dans `ps`, dans
// l'historique du shell, et dans les journaux d'audit de commandes — trois
// endroits que personne ne pense à purger. C'est la faute la plus banale d'un
// outil d'administration, et elle ne coûte rien à éviter.
func (c Command) Register(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	email := fs.String("email", "", "adresse du compte à créer")
	fs.Usage = func() {
		fmt.Fprintln(c.Err, "usage: hexa-cli register --email <adresse>")
		fmt.Fprintln(c.Err, "  le mot de passe est lu sur l'entrée standard")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return exit.Usage
	}
	if *email == "" {
		fmt.Fprintln(c.Err, "erreur: --email est obligatoire")
		fs.Usage()
		return exit.Usage
	}

	secret, err := c.lireSecret()
	if err != nil {
		fmt.Fprintf(c.Err, "erreur: %v\n", err)
		return exit.Usage
	}

	return c.inscrire(ctx, *email, secret)
}

// inscrire appelle le cas d'usage et traduit son issue.
//
// Isolée pour que `Register` reste sous le seuil de lignes d'`arch-go`, et parce
// qu'elle nomme la frontière : au-dessus on lit des arguments, en dessous on ne
// parle plus que domaine.
func (c Command) inscrire(ctx context.Context, email, secret string) int {
	value, failure, ok := c.Module.Register(ctx, domain.RegistrationCommand{
		Email:    email,
		Password: secret,
	}).Get()
	if !ok {
		// Le message vient du DOMAINE, pas d'ici. C'est la même exigence que
		// côté HTTP : deux formulations d'une même règle divergent, et c'est la
		// mauvaise qui parle à l'utilisateur.
		fmt.Fprintf(c.Err, "erreur: %s\n", failure.Message)
		if failure.Field != "" {
			fmt.Fprintf(c.Err, "champ: %s\n", failure.Field)
		}
		return codeFor(failure)
	}

	// Sur la sortie STANDARD, et rien d'autre que le fait : un script en aval
	// lit cette ligne. Y ajouter du décor la casserait.
	fmt.Fprintf(c.Out, "%s\t%s\t%s\n", value.ID, value.Email, value.Status)
	return exit.OK
}
