package tests

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// errNotAvailable est l'échec de sonde partagé par ces tests.
var errNotAvailable = errors.New("indisponible")

// TestReadyzNamesTheDependencyWithoutLeakingItsError : le NOM, jamais la CAUSE.
//
// # Le défaut que ce test attrape — et c'est une fuite de secret
//
// Le réflexe, en écrivant cette sonde, est d'inclure l'erreur : « sinon on ne
// sait pas pourquoi ». Or l'erreur d'une sonde de base de données contient
// couramment la CHAÎNE DE CONNEXION, mot de passe compris — c'est ce que rend
// pgx lorsqu'il échoue à se connecter.
//
// /readyz est interrogé par l'orchestrateur, mais rien ne le protège : c'est un
// point d'entrée HTTP non authentifié, monté avant toute autorisation, et
// souvent joignable depuis le réseau interne. Y écrire l'erreur brute publie le
// mot de passe de la base à quiconque peut faire un GET.
//
// La sortie correcte nomme la dépendance — assez pour agir — et rien de plus. La
// cause, elle, va dans les journaux, dont l'accès est contrôlé.
func TestReadyzNamesTheDependencyWithoutLeakingItsError(t *testing.T) {
	t.Parallel()

	// Une erreur de sonde réaliste : c'est la forme que prend un échec pgx.
	const dsnish = "failed to connect to " +
		"postgres://hexa_app:sup3rs3cr3t@db.internal:5432/hexa: connection refused"

	broken := map[string]httpserver.Probe{"database": failingProbe(dsnish)}

	rec := get(t, config.EnvProduction, broken, "/readyz")
	body := rec.Body.String()

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz = %d avec une dépendance en panne, attendu 503", rec.Code)
	}
	if !strings.Contains(body, "database") {
		t.Errorf("la dépendance en panne n'est pas nommée: %q — la sonde est inexploitable", body)
	}
	for _, secret := range []string{"sup3rs3cr3t", "hexa_app", "db.internal", "connection refused"} {
		if strings.Contains(body, secret) {
			t.Errorf("FUITE : %q apparaît dans le corps de /readyz: %s", secret, body)
		}
	}
}

// TestReadyzIsReadyWhenEveryProbePasses : toutes vertes → 200.
//
// Sans ce test, la sonde pourrait rendre 503 en permanence et le test du 503
// resterait vert. Un service jamais prêt ne reçoit jamais de trafic : le
// déploiement échoue au lieu de s'achever, et personne ne comprend pourquoi.
func TestReadyzIsReadyWhenEveryProbePasses(t *testing.T) {
	t.Parallel()

	healthy := map[string]httpserver.Probe{
		"database": healthyProbe(),
		"cache":    healthyProbe(),
		"outbox":   healthyProbe(),
	}

	rec := get(t, config.EnvProduction, healthy, "/readyz")

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz = %d avec toutes les sondes vertes, attendu 200 — "+
			"le service ne recevrait jamais de trafic", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "ready") {
		t.Errorf("corps = %q", body)
	}
}

// TestReadyzWithoutProbesIsReady : aucune sonde déclarée → prêt.
//
// C'est la configuration d'un socle fraîchement généré, sur ses pilotes par
// défaut : ni base, ni cache, donc rien à sonder. `hexa new` puis `go run` doit
// passer /readyz, sinon le premier déploiement d'un utilisateur échoue sans
// qu'il ait écrit une ligne.
func TestReadyzWithoutProbesIsReady(t *testing.T) {
	t.Parallel()

	for name, readiness := range map[string]map[string]httpserver.Probe{
		"table nil":  nil,
		"table vide": {},
	} {
		if rec := get(t, config.EnvProduction, readiness, "/readyz"); rec.Code != http.StatusOK {
			t.Errorf("%s: /readyz = %d, attendu 200", name, rec.Code)
		}
	}
}
