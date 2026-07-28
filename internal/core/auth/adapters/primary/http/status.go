package http

import (
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// bearerPrefix est le schéma d'authentification attendu.
const bearerPrefix = "Bearer "

// bearer extrait le jeton d'un en-tête `Authorization`.
//
// # Tout échec rend 401, jamais 422
//
// En-tête absent, schéma inconnu, jeton trop court : un seul statut. Distinguer
// « jeton mal formé » de « jeton inconnu » dirait à un attaquant que sa chaîne a
// la bonne FORME — donc qu'il approche — et c'est exactement l'indication qu'on
// ne veut pas donner sur un identifiant.
//
// Le préfixe est comparé sans tenir compte de la casse : la RFC 7235 déclare le
// schéma insensible à la casse, et refuser `bearer ` ferait échouer des clients
// parfaitement corrects avec un message qui accuserait le jeton.
func bearer(header string) (domain.Token, error) {
	raw := strings.TrimSpace(header)
	if len(raw) <= len(bearerPrefix) || !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
		return domain.Token{}, errBadToken
	}

	token, err := domain.NewToken(strings.TrimSpace(raw[len(bearerPrefix):]))
	if err != nil {
		return domain.Token{}, errBadToken
	}
	return token, nil
}

// Les deux refus 401, séparés par ROUTE et non par cause.
//
// # Le défaut que cette séparation corrige
//
// La première version rendait « authentification requise » pour un jeton mal
// formé et « identifiants invalides » pour un jeton bien formé mais inconnu —
// les deux sur la MÊME route. L'écart disait à un attaquant que sa chaîne avait
// la bonne forme, donc qu'il approchait : très exactement l'indication qu'un 401
// unique cherche à supprimer. Trouvé par le test, pas par la relecture.
//
// Le découpage tient parce que les deux sentinelles ne se croisent jamais :
// `Verify` ne rend que `ErrTokenUnknown`, `Authenticate` que
// `ErrInvalidCredentials`. Chaque route a donc UN seul message, et chacun se lit
// naturellement là où il apparaît.
var (
	// errBadToken couvre tout ce qui touche au JETON — absent, mal formé,
	// inconnu, expiré, révoqué.
	errBadToken = huma.Error401Unauthorized("authentification requise")

	// errBadCredentials couvre tout ce qui touche au SECRET — sujet inconnu,
	// sujet mal formé, secret faux, compte fermé.
	errBadCredentials = huma.Error401Unauthorized("identifiants invalides")
)

// statusFor traduit un refus du module en réponse HTTP.
//
// # Pourquoi `errors.Is` et non un `switch` exhaustif
//
// `internal/core/**` rend une `error`, pas un `Result[T, domain.Error]` : il n'y
// a donc pas de code énumérable sur lequel `exhaustive` pourrait veiller, comme
// il le fait pour `user_registration`. Le prix de l'homogénéité du noyau est
// écrit ici plutôt que tu : ajouter une sentinelle au domaine sans l'ajouter à
// cette liste la ferait silencieusement tomber en 500.
//
// Le repli EST un 500 — jamais un 200, jamais un 403 « par défaut ». L'inconnu
// est traité comme une panne, pas comme une requête acceptable.
func statusFor(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		// Un seul message pour « ce compte n'existe pas », « le secret est
		// faux » et « le compte est fermé ». La distinction est précisément ce
		// qu'un attaquant cherche.
		return errBadCredentials

	case errors.Is(err, domain.ErrTokenUnknown):
		// Le MÊME message que `bearer` rend sur un jeton mal formé : sur une
		// route protégée, tout ce qui touche au jeton se ressemble.
		return errBadToken

	case errors.Is(err, domain.ErrForbidden):
		// 403 et non 401 : l'appelant est authentifié. Renvoyer 401 le pousserait
		// à se reconnecter en boucle pour un droit qu'il n'aura pas davantage.
		return huma.Error403Forbidden("permission refusée")

	case errors.Is(err, domain.ErrSubjectTaken):
		// 409 et non 422 : la requête est bien formée, c'est l'ÉTAT du serveur
		// qui s'y oppose.
		return huma.Error409Conflict("ce sujet est déjà enregistré")

	case errors.Is(err, domain.ErrIncomplete):
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, auth.ErrDisabled):
		// 503 et non 501 : la capacité EXISTE, elle n'est pas activée sur ce
		// déploiement. C'est une décision d'exploitation, pas une fonctionnalité
		// manquante, et 503 est ce qui autorise un client à réessayer plus tard.
		return huma.Error503ServiceUnavailable("authentification indisponible")

	default:
		// La cause technique n'est JAMAIS rendue à l'appelant : elle est
		// journalisée par le middleware, relié à l'identifiant de corrélation.
		return huma.Error500InternalServerError("une erreur interne est survenue")
	}
}
