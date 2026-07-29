package tests

import (
	"net/http"
	"testing"
)

// TestAProtectedRouteRefusesARevokedRight porte l'ADR 017 JUSQU'À LA SURFACE.
//
// # Pourquoi le rejouer ici alors que le module le garantit déjà
//
// Parce que la surface a sa propre occasion de le rompre. Le témoin du module
// prouve qu'`Authorize` interroge l'état persisté ; celui-ci prouve que la route
// l'APPELLE — à chaque requête, et pas une seule fois au montage.
//
// La faute qu'il attrape est banale et invisible : résoudre la permission au
// démarrage puis mémoriser la décision, ou poser un intergiciel qui met en cache
// l'identité résolue. Les deux passent tous les tests du module, et les deux
// rendent un droit révoqué encore actif.
//
// Le jeton reste valide tout du long — le test le vérifie explicitement entre
// les deux appels — sans quoi le refus pourrait venir d'une session invalidée et
// la démonstration serait vide.
func TestAProtectedRouteRefusesARevokedRight(t *testing.T) {
	t.Parallel()

	server, token := serveurAmorce(t)

	// Le compte d'amorçage détient le rôle admin : la route répond.
	corps := `{"subject":"bob@example.com","secret":"correct cheval batterie agrafe"}`
	if resp := envoyer(t, server, requete{http.MethodPost, identitiesPath, porteur(token), corps}); resp.status != http.StatusCreated {
		t.Fatalf("le compte d'amorçage doit pouvoir créer : attendu 201, obtenu %d — %s",
			resp.status, resp.raw)
	}

	// Révocation du DROIT, pas de la session : le rôle est redéfini sans la
	// permission de création. Aucun jeton n'est touché.
	vide := `{"permissions":[]}`
	if resp := envoyer(t, server, requete{http.MethodPut, rolePath, porteur(token), vide}); resp.status != http.StatusNoContent {
		t.Fatalf("redéfinition du rôle : attendu 204, obtenu %d — %s", resp.status, resp.raw)
	}

	// Le jeton vaut TOUJOURS : c'est ce qui rend le refus qui suit concluant.
	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusOK {
		t.Fatalf("le jeton ne devait pas être affecté par la révocation : %d — %s",
			resp.status, resp.raw)
	}

	autre := `{"subject":"carol@example.com","secret":"correct cheval batterie agrafe"}`
	resp := envoyer(t, server, requete{http.MethodPost, identitiesPath, porteur(token), autre})
	if resp.status != http.StatusForbidden {
		t.Fatalf("droit révoqué : attendu 403, obtenu %d — %s", resp.status, resp.raw)
	}
}

// TestAnUnauthenticatedCallerNeverLearnsThatARouteIsGuarded garde l'ordre des refus.
//
// # L'ordre des refus est la décision
//
// Un jeton absent ou invalide rend **401**, jamais 403. Dire « permission
// refusée » à quelqu'un qui n'est pas authentifié lui apprendrait deux choses :
// que la route existe, et qu'un droit la garde. C'est exactement la cartographie
// qu'un attaquant construit avant d'agir.
//
// Le 403 est réservé à un porteur AUTHENTIFIÉ — et là il est utile : il lui
// évite de se reconnecter en boucle pour un droit qu'il n'aura pas davantage.
func TestAnUnauthenticatedCallerNeverLearnsThatARouteIsGuarded(t *testing.T) {
	t.Parallel()

	server, _ := serveurAmorce(t)
	corps := `{"subject":"bob@example.com","secret":"correct cheval batterie agrafe"}`

	entetes := map[string]string{
		"absent":           "",
		"schéma inconnu":   "Basic " + jetonFactice,
		"jeton inventé":    "Bearer " + jetonFactice,
		"jeton trop court": "Bearer court",
	}
	for nom, entete := range entetes {
		resp := envoyer(t, server, requete{http.MethodPost, identitiesPath, entete, corps})
		if resp.status != http.StatusUnauthorized {
			t.Errorf("%s : attendu 401, obtenu %d — %s", nom, resp.status, resp.raw)
		}
	}
}

// TestAnAuthenticatedCallerWithoutTheRightGets403 garde l'autre moitié.
//
// Le compte créé par l'administrateur n'a AUCUN rôle : il s'authentifie
// parfaitement et n'obtient rien. C'est le deny par défaut vu depuis la surface,
// et c'est la moitié que le test précédent ne couvre pas — un garde qui rendrait
// 401 à tout le monde y serait vert et parfaitement inutile.
func TestAnAuthenticatedCallerWithoutTheRightGets403(t *testing.T) {
	t.Parallel()

	server, admin := serveurAmorce(t)

	const sujet = "bob@example.com"
	corps := `{"subject":"` + sujet + `","secret":"` + secret + `"}`
	if resp := envoyer(t, server, requete{http.MethodPost, identitiesPath, porteur(admin), corps}); resp.status != http.StatusCreated {
		t.Fatalf("création du compte: %d — %s", resp.status, resp.raw)
	}

	sansDroit := tokenOf(t, openSession(t, server, sujet, secret))
	resp := envoyer(t, server, requete{
		http.MethodPost, identitiesPath, porteur(sansDroit),
		`{"subject":"carol@example.com","secret":"` + secret + `"}`,
	})
	if resp.status != http.StatusForbidden {
		t.Fatalf("authentifié sans droit : attendu 403, obtenu %d — %s", resp.status, resp.raw)
	}
}

// TestClosingAnAccountTakesEffectAtOnce : la fermeture vaut au prochain appel.
//
// C'est le geste qu'on fait quand un compte est compromis, donc le seul moment
// où la latence compte vraiment. Le jeton du compte fermé a été émis AVANT la
// fermeture — c'est celui qu'un attaquant détient au moment où l'on réagit.
func TestClosingAnAccountTakesEffectAtOnce(t *testing.T) {
	t.Parallel()

	server, admin := serveurAmorce(t)

	const sujet = "bob@example.com"
	creation := envoyer(t, server, requete{
		http.MethodPost, identitiesPath, porteur(admin),
		`{"subject":"` + sujet + `","secret":"` + secret + `"}`,
	})
	if creation.status != http.StatusCreated {
		t.Fatalf("création: %d — %s", creation.status, creation.raw)
	}
	id, _ := creation.body["identity_id"].(string)

	jeton := tokenOf(t, openSession(t, server, sujet, secret))
	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+jeton); resp.status != http.StatusOK {
		t.Fatalf("le jeton vient d'être émis : %d", resp.status)
	}

	if resp := envoyer(t, server, requete{http.MethodDelete, identitiesPath + "/" + id, porteur(admin), ""}); resp.status != http.StatusNoContent {
		t.Fatalf("fermeture : attendu 204, obtenu %d — %s", resp.status, resp.raw)
	}

	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+jeton); resp.status != http.StatusUnauthorized {
		t.Fatalf("compte fermé, jeton déjà émis : attendu 401, obtenu %d — %s", resp.status, resp.raw)
	}
}
