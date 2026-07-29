// Package modules assemble le catalogue des modules MÉTIER — ADR 014.
//
// Pendant du paquet `internal/core`, et pour la même raison : `internal/config`
// ne peut nommer aucun module (règle 7 d'`arch-go`), donc le catalogue est
// construit ailleurs et lui est passé.
//
// # Pourquoi le catalogue appartient à l'APPLICATION, pas au binaire
//
// L'ADR 014 dit « le composition root fusionne », ce qui laisse entendre que
// chaque binaire déclare ce qu'il monte. Appliqué tel quel, ça casse — mesuré en
// CI, pas déduit :
//
//	configuration: invalid configuration:
//	  modules.user_registration: unknown module
//
// `cmd/server` monte `user_registration` ; `cmd/worker` non. Mais les deux
// lisent le MÊME `config/modules.yaml`. Une configuration valide pour un binaire
// devenait donc invalide pour l'autre — exactement la divergence que ce dépôt
// paie à chaque fois qu'une même vérité vit à deux endroits.
//
// L'ensemble des modules DÉCLARABLES est une propriété de l'application, pas du
// binaire qui la démarre. Ce fichier la porte, et les deux composition roots la
// lisent. Monter un module reste, lui, une décision par binaire.
//
// # Ce que ça n'autorise pas
//
// Figurer au catalogue ne monte rien. Un module déclarable mais non monté par un
// binaire donné est simplement inutilisé par lui. Le deny-par-défaut porte sur
// les NOMS : ce qui n'est pas ici n'est pas configurable.
package modules

import (
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// Catalog rend le catalogue fusionné des modules métier de cette application.
//
// C'est LA ligne qu'une application ajoute pour déclarer son module `billing` —
// et la seule. Aucun fichier du framework n'est touché : c'est très exactement
// la friction que l'ADR 014 supprime.
func Catalog() (config.ModuleCatalog, error) {
	merged, err := config.MergeCatalogs(
		userregistration.Catalog(),
	)
	if err != nil {
		return nil, fmt.Errorf("catalogue des modules métier: %w", err)
	}
	return merged, nil
}
