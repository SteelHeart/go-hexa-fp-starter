package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestHTTPModeWithoutAnAddressRefusesToResolve : pas d'adresse, pas d'appel.
//
// # Le défaut que ce test attrape
//
// Sans cette garde, l'adresse absente produit une URL relative : `"" + "/v1/things"`,
// soit `/v1/things`. `http.NewRequest` refuse une URL sans schéma, donc l'échec
// arrive — mais AU PREMIER APPEL, en production, avec un message de bibliothèque
// parlant d'URL malformée. Personne ne remonte de là à « la variable
// `interop.base_urls.some_module` n'a pas été définie dans l'environnement UAT ».
//
// La garde déplace l'échec du premier appel vers le DÉMARRAGE, et le message
// nomme le module. C'est la différence entre un incident et une ligne de journal.
//
// Une adresse présente mais VIDE est traitée comme absente : c'est exactement ce
// que produit une référence `${VAR}` non résolue, et c'est le cas le plus
// fréquent en pratique.
func TestHTTPModeWithoutAnAddressRefusesToResolve(t *testing.T) {
	t.Parallel()

	for name, baseURLs := range map[string]map[string]string{
		"table absente":   nil,
		"table vide":      {},
		"entrée vide":     {someModule: ""},
		"un AUTRE module": {"other_module": "http://other:8080"},
	} {
		var localCalls int

		_, err := modulebus.Resolve(
			modulebus.New(interop("http", baseURLs), noPublisher(t)),
			someModule, route(), someEvent, localCaller(&localCalls))

		if err == nil {
			t.Errorf("%s: résolution acceptée sans adresse — l'échec surviendrait au premier appel", name)
			continue
		}
		if !errors.Is(err, modulebus.ErrNoBaseURL) {
			t.Errorf("%s: rendu avec %v, attendu ErrNoBaseURL", name, err)
		}
	}
}
