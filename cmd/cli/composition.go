package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/ressources"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/outboxpub"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// monte porte ce que la CLI a assemblé, et de quoi le relâcher.
//
// Type plutôt que trois retours : la règle d'architecture en autorise deux, et
// c'est la SEPTIÈME fois que ce dépôt paie la même leçon.
type monte struct {
	cfg   config.Config
	users userregistration.Module
	conn  ressources.Connexions
}

// composer charge la configuration et monte ce dont la CLI a besoin.
//
// # La MÊME configuration que les autres binaires
//
// Le catalogue avant la configuration, exactement comme `cmd/server` et
// `cmd/worker` (ADR 014). Un binaire d'administration qui lirait sa propre
// configuration finirait par agir sur un autre magasin que celui qu'il prétend
// administrer — et le défaut ne se verrait qu'en constatant l'absence d'un
// compte pourtant créé.
//
// # Le journal part sur la sortie d'ERREUR
//
// La sortie standard d'une CLI est une donnée : un script la lit. Y mélanger des
// lignes de journal casserait tout appelant qui en fait un `awk` ou un `cut`.
func composer(ctx context.Context) (monte, error) {
	catalog, err := moduleCatalog()
	if err != nil {
		return monte{}, err
	}

	cfg, err := config.Load(catalog)
	if err != nil {
		return monte{}, fmt.Errorf("configuration: %w", err)
	}

	conn, err := ressources.Open(ctx, cfg, catalog)
	if err != nil {
		return monte{}, fmt.Errorf("ouverture des connexions: %w", err)
	}

	users, err := monterInscription(cfg, conn)
	if err != nil {
		conn.Close()
		return monte{}, err
	}

	avertirMagasinVolatile(cfg)
	return monte{cfg: cfg, users: users, conn: conn}, nil
}

// avertirMagasinVolatile dit ce qu'un compte créé ici vaut réellement.
//
// # Pourquoi un avertissement et non un refus
//
// Le dépileur, lui, REFUSE un pilote non partagé : il dépilerait un magasin vide
// et tournerait sans rien faire. Le raisonnement est le même ici — un compte
// écrit dans un magasin qui meurt avec le processus n'existe pas — mais la
// conclusion diffère, et pour une raison précise : `user_registration` n'a
// AUCUN autre pilote. Refuser rendrait cette commande définitivement
// inutilisable, ce qui n'est pas un garde, c'est une suppression.
//
// L'avertissement va sur la sortie d'ERREUR : la sortie standard est une donnée
// qu'un script découpe, et y glisser une phrase casserait le découpage.
//
// Le jour où un pilote partagé existera, ce cas devra devenir un refus — comme
// pour le dépileur.
func avertirMagasinVolatile(cfg config.Config) {
	if cfg.Modules.DriverOf(userregistration.Name) != userregistration.DriverMemory {
		return
	}
	fmt.Fprintln(os.Stderr,
		"avertissement: pilote `memory` — le compte créé MEURT avec ce processus "+
			"et ne sera visible d'aucun autre binaire. Aucun pilote partagé n'existe "+
			"encore pour user_registration.")
}

// monterInscription branche le module d'inscription.
//
// Le pilote et les paramètres de hachage viennent de la configuration, comme
// dans `cmd/server` : un compte créé par la CLI doit être en tout point celui
// qu'aurait créé la surface HTTP, sans quoi la démonstration ne vaut rien.
func monterInscription(cfg config.Config, conn ressources.Connexions) (userregistration.Module, error) {
	outboxMod, err := outbox.New(cfg.Modules[outbox.Name], outbox.Deps{Pool: conn.Pool, Now: time.Now})
	if err != nil {
		return userregistration.Module{}, fmt.Errorf("module outbox: %w", err)
	}

	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  cfg.Security.Argon2.MemoryKiB,
		Iterations: cfg.Security.Argon2.Iterations,
		Threads:    cfg.Security.Argon2.Threads,
	})

	users, err := userregistration.New(
		cfg.Modules.DriverOf(userregistration.Name), userregistration.Deps{
			HashPassword: hashing.New(hasher),
			PublishEvent: outboxpub.New(outboxMod.Enqueue),
			GenerateID:   nouvelIdentifiant,
			Now:          userregistration.SystemClock(),
		})
	if err != nil {
		return userregistration.Module{}, fmt.Errorf("module user_registration: %w", err)
	}
	return users, nil
}

// nouvelIdentifiant produit un identifiant ordonné dans le temps.
//
// UUID v7 et non v4 : l'identifiant devient une clé primaire, et une clé
// aléatoire disperse les insertions sur tout l'index.
func nouvelIdentifiant() domain.UserID {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.UserID(uuid.NewString())
	}
	return domain.UserID(id.String())
}

// moduleCatalog assemble ce que la configuration a le droit de nommer.
func moduleCatalog() (config.ModuleCatalog, error) {
	coreCatalog, err := core.Catalog()
	if err != nil {
		return nil, fmt.Errorf("catalogue du noyau: %w", err)
	}
	businessCatalog, err := modules.Catalog()
	if err != nil {
		return nil, fmt.Errorf("catalogue des modules métier: %w", err)
	}
	merged, err := config.MergeCatalogs(coreCatalog, businessCatalog)
	if err != nil {
		return nil, fmt.Errorf("fusion des catalogues: %w", err)
	}
	return merged, nil
}
