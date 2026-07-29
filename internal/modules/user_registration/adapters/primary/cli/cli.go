// Package cli expose le module d'inscription sur la surface LIGNE DE COMMANDE.
//
// # Ce paquet est la démonstration de la propriété n°2 du socle
//
// *Le nombre de frontends est un non-sujet.* Il ne se démontre pas en
// l'écrivant : il se démontre en ajoutant une surface et en constatant qu'aucune
// ligne de `domain/`, `ports/` ou `application/` ne bouge. C'est le cas ici — ce
// paquet appelle **le même port** que la surface HTTP, `mod.Register`, et le
// module ne sait pas laquelle des deux l'appelle.
//
// Avec une seule surface, la propriété était énoncée. Avec deux, elle est
// mesurée : la carte d'impact de cette PR ne touche que `adapters/primary/`.
//
// # Une surface est un TRADUCTEUR, jamais une place métier
//
// Ce paquet fait trois choses : lire des arguments, appeler le cas d'usage,
// traduire le résultat en sortie et en CODE DE RETOUR. Il ne valide rien
// lui-même — le domaine le fait déjà, et dupliquer la validation garantit qu'un
// jour les deux divergeront, au détriment de celle qui parle à l'utilisateur.
//
// # Carte des fichiers
//
//	cli.go        la commande, ses flux, et la traduction des erreurs
//	register.go   `register` — inscrire un compte
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Command porte ce dont la surface a besoin pour répondre.
//
// # Pourquoi une structure plutôt que des paramètres
//
// `Run(ctx, mod, args, out, err)` fait cinq arguments, et la règle
// d'architecture en autorise quatre. Elle a raison ici pour une raison propre au
// sujet : `out` et `err` ont le même type, donc les inverser compile — et un
// binaire qui écrit ses erreurs sur la sortie standard casse silencieusement
// tout script qui redirige l'une sans l'autre.
//
// # Les flux sont INJECTÉS, jamais `os.Stdout`
//
// C'est ce qui rend cette surface testable sans lancer de processus : un test
// lui donne deux tampons et lit ce qui en sort. Un paquet qui écrirait
// directement sur `os.Stdout` ne serait vérifiable qu'en capturant les
// descripteurs du processus — donc pas en parallèle, et pas de façon lisible.
type Command struct {
	Module userregistration.Module
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
}

// lireSecret lit le mot de passe sur l'entrée fournie.
//
// # Pourquoi l'entrée standard et pas un drapeau
//
// Un `--password` figure dans `ps`, dans l'historique du shell, et dans les
// journaux d'audit de commandes — trois endroits que personne ne pense à purger.
// C'est la faute la plus banale d'un outil d'administration, et elle ne coûte
// rien à éviter.
//
// # NON-GARANTIE : l'écho du terminal n'est PAS coupé
//
// En usage interactif, le mot de passe s'affiche pendant la frappe. Le couper
// exige de manipuler le terminal (`golang.org/x/term`), donc de distinguer un
// terminal d'un tube — et un binaire qui se comporte différemment selon qu'on le
// pipe ou non est un binaire dont les scripts cassent au premier changement
// d'environnement.
//
// L'usage visé est `printf '%s' "$SECRET" | hexa-cli register --email …`, où la
// question ne se pose pas. La faiblesse est écrite plutôt que tue.
func (c Command) lireSecret() (string, error) {
	if c.In == nil {
		return "", errAucuneEntree
	}

	// Lecture octet par octet, SANS tampon, et c'est une décision.
	//
	// `bufio.Reader` lit en AVANCE : il remplit son tampon bien au-delà du saut
	// de ligne, et tout ce qu'il a absorbé est perdu pour le lecteur suivant.
	// Un lecteur de secret doit consommer sa ligne et rien d'autre — sinon il
	// avale ce qui suit sur l'entrée, et l'appelant ne le découvre qu'en
	// constatant que sa donnée a disparu.
	//
	// Trouvé par un test qui appelait `Register` deux fois : le second appel
	// lisait une entrée vide alors qu'elle contenait bien un second secret.
	// Un mot de passe est court : le coût de la lecture octet par octet est nul.
	var secret strings.Builder
	octet := make([]byte, 1)
	for {
		n, err := c.In.Read(octet)
		if n > 0 {
			if octet[0] == '\n' {
				break
			}
			secret.WriteByte(octet[0])
		}
		if errors.Is(err, io.EOF) {
			// Une entrée sans saut de ligne final est LÉGITIME : `printf` sans
			// `\n` en produit une, et c'est la forme la plus sûre pour un secret.
			break
		}
		if err != nil {
			return "", fmt.Errorf("lecture du secret: %w", err)
		}
	}

	lu := strings.TrimRight(secret.String(), "\r")
	if lu == "" {
		return "", errSecretVide
	}
	return lu, nil
}

// Erreurs de la surface elle-même, distinctes de celles du domaine.
var (
	errAucuneEntree = errors.New("aucune entrée standard : le secret est lu sur stdin")
	errSecretVide   = errors.New("secret vide : rien n'a été lu sur l'entrée standard")
)

// codeFor traduit une erreur de domaine en code de sortie `sysexits`.
//
// # Le `switch` est EXHAUSTIF, et le linter le garde
//
// Ajouter un code d'erreur au domaine force à décider, ICI, ce qu'un appelant
// automatisé en voit. Sans cette contrainte, un code neuf se traduirait
// silencieusement en « erreur interne » — et un script cesserait de réessayer
// une panne transitoire, ou réessaierait à l'infini une saisie invalide.
//
// C'est la même exigence que `statusFor` côté HTTP, avec la même raison : la
// distinction sert à décider du RÉESSAI, et elle ne se devine pas d'un message.
//
// Le `default` reste malgré l'exhaustivité : il couvre la valeur zéro et toute
// chaîne fabriquée par conversion. Deny par défaut — l'inconnu est une panne,
// jamais une réussite.
func codeFor(err domain.Error) int {
	switch err.Code {
	case domain.CodeInvalidEmail, domain.CodeWeakPassword:
		// Donnée invalide : réessayer à l'identique échouera pareil.
		return exit.DataErr

	case domain.CodeEmailAlreadyExists:
		// L'ÉTAT du serveur s'y oppose, pas la commande. `DataErr` et non
		// `Unavailable` : réessayer ne changera rien tant que le compte existe.
		return exit.DataErr

	case domain.CodeUnavailable:
		// Le SEUL code qui autorise un réessai, et toute l'utilité de la table.
		return exit.Unavailable

	case domain.CodeInternal:
		return exit.Software

	default:
		return exit.Software
	}
}
