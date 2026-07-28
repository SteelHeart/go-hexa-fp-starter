// Package domain porte le vocabulaire de l'authentification et de
// l'autorisation, sans dépendance.
//
// # Pourquoi ce module existe
//
// Un évaluateur produit pose deux questions avant toute autre : « comment
// j'authentifie ? » et « comment j'isole mes clients ? ». Tant que la première
// n'a pas de réponse, aucun module métier n'est livrable — c'est le critère de
// P1 dans documentation/produit/personas.md, et il valait « tous » avant ce
// module.
//
// # Ce paquet est PUR
//
// Ni horloge, ni aléa, ni I/O. L'instant et l'aléa entrent par des ports ; les
// mots de passe n'y sont jamais hachés — le hachage est un effet coûteux et
// paramétré, fourni par internal/infrastructure/security.
package domain

import "errors"

// Erreurs sentinelles du module.
//
// # Pourquoi des sentinelles et non un Result[T, Error]
//
// `internal/core/**` retourne `error`, `internal/modules/**` retourne
// `Result[T, domain.Error]` : la frontière est nette et vérifiable, et un module
// noyau est technique.
//
// `auth` est le cas limite de cet invariant — il a bel et bien une taxonomie que
// les surfaces doivent traduire en 401, 403 et 422. Elle passe donc par des
// sentinelles ÉNUMÉRÉES, reconnaissables par `errors.Is`. C'est moins expressif
// qu'un `Result`, et c'est le prix assumé de l'homogénéité du noyau (ADR 017).
//
// # Ce que chaque sentinelle vaut pour une surface HTTP
//
//	ErrInvalidCredentials  401 — et JAMAIS de distinction « identifiant inconnu »
//	                             contre « mot de passe faux » : la différence dit
//	                             à un attaquant quels comptes existent
//	ErrTokenUnknown        401 — jeton inconnu, expiré, ou révoqué. Les trois se
//	                             confondent DÉLIBÉRÉMENT côté client
//	ErrForbidden           403 — authentifié, mais la permission manque
//	ErrIncomplete          422 — la demande elle-même est mal formée
var (
	// ErrInvalidCredentials refuse une authentification.
	//
	// Un seul message pour « ce sujet n'existe pas » et « le secret est faux ».
	// Les séparer transformerait le formulaire de connexion en oracle
	// d'existence de comptes — la faute la plus répandue du domaine.
	ErrInvalidCredentials = errors.New("identifiants invalides")

	// ErrTokenUnknown refuse un jeton.
	//
	// Inconnu, expiré ou révoqué : indiscernables pour l'appelant, et c'est
	// voulu. Dire « expiré » plutôt que « inconnu » confirme qu'un jeton a
	// existé, donc qu'un compte existe.
	ErrTokenUnknown = errors.New("jeton inconnu ou expiré")

	// ErrForbidden refuse une action à une identité pourtant authentifiée.
	ErrForbidden = errors.New("permission refusée")

	// ErrIncomplete refuse une demande mal formée AVANT tout accès au magasin.
	//
	// Le refus est en amont pour deux raisons : il ne coûte aucune requête, et
	// il ne laisse pas une chaîne vide atteindre un pilote, où elle deviendrait
	// une clé légitime.
	ErrIncomplete = errors.New("demande incomplète")

	// ErrSubjectTaken refuse un sujet déjà enregistré.
	ErrSubjectTaken = errors.New("ce sujet est déjà enregistré")
)
