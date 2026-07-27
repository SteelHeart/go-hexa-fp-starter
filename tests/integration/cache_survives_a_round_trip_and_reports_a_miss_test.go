//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/cache"
)

// TestCacheSurvivesARoundTripAndReportsAMiss éprouve le paquet `cache`, qui
// n'avait AUCUN test — pas même de compilation exercée (#37).
//
// La raison de cette absence est structurelle : `cache.New` fait un `Ping` à la
// construction et `cache.JSON` exige un client. Ce paquet n'est donc atteignable
// qu'à ce niveau, jamais par `go test ./...`.
//
// Ce que le test garde : l'aller-retour d'une valeur typée, et surtout le
// comportement en ABSENCE. `Getter` traite l'absence et la panne de la même
// façon — une option vide — parce que dans les deux cas l'appelant n'a pas la
// valeur et doit se rabattre sur la source de vérité. Ce choix est délibéré, et
// il ne se vérifie que contre un vrai Redis.
func TestCacheSurvivesARoundTripAndReportsAMiss(t *testing.T) {
	ctx := ctxTest(t)
	client := redisClient(t)

	type profil struct {
		Nom string `json:"nom"`
		Age int    `json:"age"`
	}

	namespace := unique(t, "integration-cache")
	get, set, del := cache.JSON[profil](client, namespace, time.Minute)

	// Absence AVANT toute écriture : l'option doit être vide.
	if absent := get(ctx, "inconnu"); absent.IsSome() {
		t.Fatal("une clé jamais écrite doit rendre une option vide")
	}

	// `Setter` et `Deleter` ne retournent RIEN, et c'est une décision : un cache
	// qui tombe ne doit jamais faire échouer une requête métier. La conséquence
	// est qu'une écriture muette est indiscernable d'une écriture réussie — d'où
	// la relecture qui suit, seule preuve possible.
	attendu := profil{Nom: "alice", Age: 30}
	set(ctx, "cle", attendu)
	t.Cleanup(func() { del(ctxTest(t), "cle") })

	lu := get(ctx, "cle")
	if lu.IsNone() {
		t.Fatal("la valeur écrite doit être relue : sinon le cache ne cache rien")
	}
	if got := lu.ValueOr(profil{}); got != attendu {
		t.Errorf("valeur relue = %+v, attendu %+v", got, attendu)
	}

	// Après suppression, retour à l'absence.
	del(ctx, "cle")
	if apres := get(ctx, "cle"); apres.IsSome() {
		t.Fatal("une clé supprimée doit redevenir absente — une invalidation qui " +
			"n'invalide rien est pire que pas de cache")
	}
}
