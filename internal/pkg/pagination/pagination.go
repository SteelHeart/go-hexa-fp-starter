// Package pagination fournit la pagination par curseur, contrat commun à
// toutes les surfaces.
//
// Pourquoi pas OFFSET : au-delà de quelques milliers de lignes Postgres doit
// parcourir puis jeter tout ce qui précède, et surtout une insertion concurrente
// décale les pages — l'appelant saute alors des lignes sans le savoir. Un
// curseur désigne une position stable, pas un rang.
package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Bornes de taille de page. Une page non bornée est un déni de service offert.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// ErrInvalidCursor signale un curseur illisible ou falsifié.
var ErrInvalidCursor = errors.New("curseur invalide")

// Cursor désigne une position stable dans un ordre (CreatedAt, ID).
//
// Le couple horodatage + identifiant est nécessaire : l'horodatage seul n'est
// pas unique, et deux lignes créées dans la même microseconde feraient boucler
// la pagination.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// Encode sérialise le curseur pour un transport public.
//
// L'encodage est réversible et NON signé : un curseur n'est pas un secret et ne
// doit jamais porter d'information d'autorisation. La requête revérifie les
// droits, toujours.
func (c Cursor) Encode() string {
	raw := strconv.FormatInt(c.CreatedAt.UTC().UnixMicro(), 10) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor lit un curseur encodé.
func DecodeCursor(encoded string) (Cursor, error) {
	if encoded == "" {
		return Cursor{}, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: base64: %w", ErrInvalidCursor, err)
	}
	micros, id, found := strings.Cut(string(raw), "|")
	if !found || id == "" {
		return Cursor{}, fmt.Errorf("%w: format", ErrInvalidCursor)
	}
	parsed, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: horodatage: %w", ErrInvalidCursor, err)
	}
	return Cursor{CreatedAt: time.UnixMicro(parsed).UTC(), ID: id}, nil
}

// Request porte une demande de page.
type Request struct {
	After Cursor
	Limit int
	// HasAfter distingue « première page » de « curseur zéro ».
	HasAfter bool
}

// NewRequest construit une demande de page en bornant la taille.
// Un curseur vide vaut « première page », ce n'est pas une erreur.
func NewRequest(encodedCursor string, limit int) (Request, error) {
	req := Request{Limit: clampLimit(limit)}
	if encodedCursor == "" {
		return req, nil
	}
	cursor, err := DecodeCursor(encodedCursor)
	if err != nil {
		return Request{}, err
	}
	req.After = cursor
	req.HasAfter = true
	return req, nil
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultLimit
	case limit > MaxLimit:
		return MaxLimit
	default:
		return limit
	}
}

// FetchLimit est la taille à demander à la base : un élément de plus que
// nécessaire, pour savoir s'il existe une page suivante sans faire de COUNT.
func (r Request) FetchLimit() int { return r.Limit + 1 }

// Page porte une tranche de résultats et de quoi demander la suivante.
type Page[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}

// NewPage construit une page à partir des lignes réellement lues (FetchLimit
// éléments au plus) et de la fonction qui extrait le curseur d'un élément.
func NewPage[T any](fetched []T, req Request, cursorOf func(T) Cursor) Page[T] {
	hasMore := len(fetched) > req.Limit
	items := fetched
	if hasMore {
		items = fetched[:req.Limit]
	}
	page := Page[T]{Items: items, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		page.NextCursor = cursorOf(items[len(items)-1]).Encode()
	}
	return page
}
