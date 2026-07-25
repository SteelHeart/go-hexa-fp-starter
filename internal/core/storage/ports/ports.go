// Package ports déclare les contrats du stockage d'objets.
//
// Ce paquet ne contient QUE des déclarations de types.
package ports

import (
	"context"
	"io"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// Put écrit un objet et retourne son emplacement.
//
// Contrat : la clé est TOUJOURS dérivée par domain.SafeKey, jamais construite par
// le pilote. Un nom refusé rend domain.ErrUnsafeName, quel que soit le pilote.
//
// Le contenu est consommé en flux : un pilote ne charge pas l'objet entier en
// mémoire, sinon un téléversement de deux gigaoctets tuerait le processus.
type Put = func(ctx context.Context, obj domain.Object) (domain.Located, error)

// Get lit un objet stocké.
//
// L'appelant DOIT fermer le flux retourné. Une clé hors des limites du magasin
// rend domain.ErrUnsafeName — la clé vient d'une URL, donc d'un inconnu.
type Get = func(ctx context.Context, key domain.Key) (io.ReadCloser, error)

// Delete supprime un objet.
//
// Contrat : supprimer un objet absent n'est PAS une erreur. L'appelant veut que
// l'objet ne soit plus là, et il n'y est plus. Faire échouer une suppression
// idempotente obligerait chaque appelant à distinguer deux cas équivalents.
type Delete = func(ctx context.Context, key domain.Key) error
