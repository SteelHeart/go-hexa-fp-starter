package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestEveryShippedOptionIsDeclared : la configuration LIVRÉE passe le nouveau
// contrôle.
//
// Sans lui, ce lot pourrait être vert en n'ayant qu'ajouté une garde que le
// dépôt lui-même ne franchit pas — et le défaut n'apparaîtrait qu'au premier
// `go run` de quelqu'un d'autre.
//
// C'est le pendant positif du témoin d'échec : un garde doit refuser ce qui est
// faux ET accepter ce qui est juste. Les deux moitiés sont nécessaires, et
// l'expérience de ce dépôt est que c'est la première qu'on oublie de vérifier.
func TestEveryShippedOptionIsDeclared(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("la configuration livrée doit se charger: %v", err)
	}

	// Et le contrôle doit avoir quelque chose à contrôler : si `modules.yaml`
	// cessait de porter la moindre option, ce test resterait vert sans rien
	// prouver.
	options := 0
	for nom := range cfg.Modules {
		options += len(cfg.Modules.Get(nom).Options)
	}
	if options == 0 {
		t.Fatal("la configuration livrée ne porte aucune option — ce test ne prouve alors rien")
	}
	t.Logf("%d options déclarées dans la configuration livrée", options)
}
