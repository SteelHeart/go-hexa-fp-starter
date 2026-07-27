// Package ports déclare le contrat du journal d'audit.
//
// Ce paquet ne contient que des déclarations de types.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// Record enregistre un fait d'audit.
//
// Contrat : l'écriture passe par le querier du contexte, donc DANS la
// transaction métier si elle existe. Un fait annulé ne laisse pas de trace
// d'audit mensongère — c'est ce qui distingue un journal d'audit d'un log.
//
// Contrat d'erreur : une entrée incomplète est refusée (voir Entry.IsComplete).
// Toute autre erreur est technique.
type Record = func(ctx context.Context, entry domain.Entry) error
