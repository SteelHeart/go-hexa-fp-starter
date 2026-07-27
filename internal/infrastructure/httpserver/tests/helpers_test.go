// Package tests exerce le montage HTTP par son API PUBLIQUE.
//
// Ce paquet monte la pile que TOUTE requête traverse, et les deux sondes que
// l'orchestrateur interroge pour décider de tuer un conteneur ou de lui envoyer
// du trafic. Une erreur ici ne se traduit pas par un bug fonctionnel : elle se
// traduit par une indisponibilité, ou par une fuite dans un corps de réponse.
//
// Rien de tout cela n'exige de service : le routeur se construit en mémoire et
// les sondes sont des closures.
package tests

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// quietLogger rend un journal muet : ces tests observent des RÉPONSES, pas des
// traces.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(newDiscard(), nil))
}

// newDiscard évite d'importer io.Discard dans chaque fichier.
func newDiscard() *discard { return &discard{} }

type discard struct{}

func (*discard) Write(p []byte) (int, error) { return len(p), nil }

// serverConfig construit une configuration suffisante pour monter le routeur.
//
// Les bornes sont larges : ces tests ne mesurent pas la limitation de débit, elle
// a ses propres tests dans `internal/pkg/middleware/tests`. Un débit trop bas ici
// ferait échouer des tests pour une raison qui n'est pas la leur.
func serverConfig(env config.Environment) config.Config {
	var cfg config.Config
	cfg.App.Env = env
	cfg.App.Name = "hexa-tests"
	cfg.App.Version = "0.0.0-tests"
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 0
	cfg.HTTP.MaxBodyBytes = 1 << 20
	cfg.HTTP.AllowedOrigins = []string{"https://app.example.com"}
	cfg.HTTP.ReadTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.WriteTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.IdleTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.ShutdownTimeout = config.Duration(time.Second)
	cfg.Limits.RPS = 1000
	cfg.Limits.Burst = 1000
	return cfg
}

// router monte le routeur avec les sondes fournies.
func router(env config.Environment, readiness map[string]httpserver.Probe) *httpserver.Router {
	return httpserver.NewRouter(serverConfig(env), quietLogger(), readiness)
}

// get interroge un chemin du routeur et rend la réponse enregistrée.
func get(t *testing.T, env config.Environment, readiness map[string]httpserver.Probe, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "http://127.0.0.1"+path, nil)
	rec := httptest.NewRecorder()
	router(env, readiness).Mux.ServeHTTP(rec, req)
	return rec
}

// failingProbe rend une sonde qui échoue avec l'erreur donnée.
func failingProbe(message string) httpserver.Probe {
	return func(context.Context) error { return errors.New(message) }
}

// healthyProbe rend une sonde qui réussit.
func healthyProbe() httpserver.Probe {
	return func(context.Context) error { return nil }
}
