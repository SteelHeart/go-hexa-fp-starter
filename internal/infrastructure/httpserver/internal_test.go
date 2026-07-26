package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Ces tests visent des champs NON EXPORTÉS : `Server.http` est privé, et les
// délais qu'il porte sont précisément ce qu'il faut vérifier. C'est le cas
// prévu par rules/tests.md §2 pour un `internal_test.go`.

func testConfig() config.Config {
	var cfg config.Config
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 0
	cfg.HTTP.ReadTimeout = config.Duration(3 * time.Second)
	cfg.HTTP.WriteTimeout = config.Duration(4 * time.Second)
	cfg.HTTP.IdleTimeout = config.Duration(5 * time.Second)
	cfg.HTTP.ShutdownTimeout = config.Duration(time.Second)
	return cfg
}

func quiet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestEveryTimeoutIsSet : aucun délai laissé à zéro.
//
// # Le défaut que ce test attrape : une attaque à une ligne
//
// `ReadHeaderTimeout` à zéro signifie « pas de limite ». Une connexion qui
// envoie ses en-têtes un octet à la fois immobilise alors une goroutine
// INDÉFINIMENT. Quelques milliers de connexions de ce type — un script trivial,
// pas d'outil spécialisé — épuisent le serveur.
//
// C'est l'attaque Slowloris, et sa particularité est de ne ressembler à rien :
// pas de pic de trafic, pas d'erreur, pas de log. Le service arrête simplement
// d'accepter des connexions.
//
// Go ne met AUCUN de ces délais par défaut. Un `http.Server{}` écrit sans y
// penser est vulnérable, et c'est le cas le plus courant.
func TestEveryTimeoutIsSet(t *testing.T) {
	t.Parallel()

	server := New(testConfig(), nil, quiet())

	for name, got := range map[string]time.Duration{
		"ReadHeaderTimeout": server.http.ReadHeaderTimeout,
		"ReadTimeout":       server.http.ReadTimeout,
		"WriteTimeout":      server.http.WriteTimeout,
		"IdleTimeout":       server.http.IdleTimeout,
	} {
		if got == 0 {
			t.Errorf("%s vaut 0 — « pas de limite », donc une goroutine par connexion lente", name)
		}
	}
	// ReadHeaderTimeout doit suivre la configuration, pas une constante oubliée.
	if server.http.ReadHeaderTimeout != 3*time.Second {
		t.Errorf("ReadHeaderTimeout = %s, attendu 3s (la valeur configurée)",
			server.http.ReadHeaderTimeout)
	}
}

// TestMetricsServerListensOnLoopbackOnly : les métriques ne sortent pas de la machine.
//
// # Pourquoi c'est une exigence de sécurité et pas de propreté
//
// `/metrics` publie la volumétrie, les noms de routes, les latences, le nombre
// d'erreurs par type. C'est une carte de la structure interne et de l'activité du
// service, sans authentification.
//
// Écouter sur `0.0.0.0` l'exposerait à tout ce qui peut joindre le conteneur. La
// liaison est donc explicitement `127.0.0.1`, et un collecteur y accède par un
// side-car ou une redirection choisie — décision explicite, pas défaut implicite.
func TestMetricsServerListensOnLoopbackOnly(t *testing.T) {
	t.Parallel()

	server := NewMetricsServer(9100, quiet())

	if got := server.http.Addr; got != "127.0.0.1:9100" {
		t.Errorf("adresse des métriques = %q, attendu 127.0.0.1:9100 — "+
			"toute autre liaison publie la structure interne du service", got)
	}
	if server.http.ReadHeaderTimeout == 0 {
		t.Error("le serveur de métriques n'a pas de ReadHeaderTimeout")
	}
}

// TestRunReturnsCleanlyOnCancellation : l'arrêt rend la main, sans erreur.
//
// # Le défaut que ce test attrape
//
// Un `Run` qui rendrait l'erreur `http.ErrServerClosed` ferait échouer l'arrêt
// NORMAL : le processus sortirait avec un code non nul à chaque déploiement,
// l'orchestrateur le compterait comme un plantage, et le déploiement serait
// marqué en échec alors que tout s'est bien passé.
//
// Le test se lie au port 0 — le système en choisit un libre — donc il n'entre en
// conflit avec rien, ni avec un autre test, ni avec un service de la machine.
func TestRunReturnsCleanlyOnCancellation(t *testing.T) {
	t.Parallel()

	server := New(testConfig(), http.NotFoundHandler(), quiet())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	// L'écoute est asynchrone : on annule sans attendre, et Run doit gérer les
	// deux ordres possibles (annulé avant ou après le début de l'écoute).
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run a rendu %v à l'arrêt — le déploiement serait compté en échec", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run n'a pas rendu la main : l'arrêt bloquerait jusqu'au SIGKILL")
	}
}
