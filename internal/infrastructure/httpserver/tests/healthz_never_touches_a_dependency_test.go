package tests

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// TestHealthzNeverTouchesADependency : /healthz est INDÉPENDANT des dépendances.
//
// # Le défaut que ce test attrape, et il transforme une panne partielle en panne totale
//
// C'est l'erreur la plus tentante de tout le montage HTTP : « autant vérifier la
// base dans /healthz aussi, c'est plus complet ». Elle est catastrophique.
//
// /healthz est la sonde de VIVACITÉ : l'orchestrateur tue et redémarre le
// conteneur qui y répond mal. Si elle interroge la base, alors une panne de base
// fait échouer la sonde sur TOUTES les répliques à la fois. Kubernetes les tue
// toutes, les redémarre, elles échouent encore, il les tue encore.
//
// Résultat : une base momentanément indisponible — dont on se relève — devient un
// service entièrement mort, en boucle de redémarrage, incapable même de servir ce
// qui ne dépendait pas de la base. Le remède aggrave la maladie.
//
// La distinction est nette et doit rester outillée :
//
//	/healthz  suis-je VIVANT ?      aucune dépendance      → sinon on me tue
//	/readyz   puis-je SERVIR ?      toutes les dépendances → sinon on me retire du trafic
func TestHealthzNeverTouchesADependency(t *testing.T) {
	t.Parallel()

	var probeCalls atomic.Int64
	watched := map[string]httpserver.Probe{
		// Une sonde qui compte ses appels ET qui échoue : si /healthz la
		// consultait, le test le verrait deux fois.
		"database": func(context.Context) error {
			probeCalls.Add(1)
			return errNotAvailable
		},
	}

	rec := get(t, config.EnvProduction, watched, "/healthz")

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d alors qu'une dépendance est en panne, attendu 200 — "+
			"l'orchestrateur tuerait toutes les répliques en boucle", rec.Code)
	}
	if calls := probeCalls.Load(); calls != 0 {
		t.Errorf("/healthz a consulté %d sonde(s) — il ne doit en consulter AUCUNE", calls)
	}
}

// TestHealthzAnswersJSON : le corps est du JSON, avec son type de contenu.
//
// Une sonde qui rend du texte brut annoncé `text/plain` casse les collecteurs qui
// désérialisent la réponse — et personne ne le remarque avant la mise en place de
// la supervision, c'est-à-dire trop tard.
func TestHealthzAnswersJSON(t *testing.T) {
	t.Parallel()

	rec := get(t, config.EnvProduction, nil, "/healthz")

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, attendu application/json", got)
	}
	if body := rec.Body.String(); body != `{"status":"ok"}` {
		t.Errorf("corps = %q", body)
	}
}
