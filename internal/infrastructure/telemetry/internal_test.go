package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Ces tests visent des identifiants NON EXPORTÉS — `newLoggerTo`, `parseLevel`,
// `traceHandler` — donc un `internal_test.go` (rules/tests.md §2). Le journal
// écrit dans un tampon, jamais sur la sortie standard : un test qui polluerait la
// sortie rendrait la lecture des autres impossible.

func loggerConfig(env config.Environment, format, level string) config.Config {
	var cfg config.Config
	cfg.App.Env = env
	cfg.App.Name = "hexa-tests"
	cfg.App.Version = "1.2.3"
	cfg.Telemetry.LogFormat = format
	cfg.Telemetry.LogLevel = level
	return cfg
}

// TestNonDevelopmentAlwaysLogsJSON : hors développement, le format est imposé.
//
// # Le défaut que ce test attrape
//
// La condition est `format == "json" || !env.IsDevelopment()`. Le second terme
// est ce qui compte : il IMPOSE le JSON hors développement, même si la
// configuration demande du texte.
//
// Sans lui, un `log_format: text` recopié depuis un fichier de développement vers
// la production produirait des journaux que le collecteur ne sait pas
// désérialiser. Ils continueraient d'être émis, la supervision resterait verte —
// et le jour de l'incident, plus rien n'est requêtable. On ne s'en aperçoit qu'au
// moment où on en a besoin.
func TestNonDevelopmentAlwaysLogsJSON(t *testing.T) {
	t.Parallel()

	for _, env := range []config.Environment{
		config.EnvTest, config.EnvUAT, config.EnvProduction, "inconnu",
	} {
		var buf bytes.Buffer
		// La configuration demande explicitement du TEXTE : elle doit être
		// ignorée.
		newLoggerTo(loggerConfig(env, "text", "info"), &buf).Info("message")

		var decoded map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
			t.Errorf("env=%q, format=text : la sortie n'est pas du JSON (%v): %s",
				env, err, buf.String())
		}
	}
}

// TestDevelopmentLogsTextUnlessJSONIsAsked : en local, l'humain d'abord.
func TestDevelopmentLogsTextUnlessJSONIsAsked(t *testing.T) {
	t.Parallel()

	var text bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvDevelopment, "", "info"), &text).Info("message")
	if json.Valid(bytes.TrimSpace(text.Bytes())) {
		t.Errorf("development sans format demandé produit du JSON: %s", text.String())
	}

	var asJSON bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvDevelopment, "json", "info"), &asJSON).Info("message")
	if !json.Valid(bytes.TrimSpace(asJSON.Bytes())) {
		t.Errorf("development avec format=json ne produit pas du JSON: %s", asJSON.String())
	}
}

// TestEveryRecordCarriesTheServiceIdentity : service, env et version, toujours.
//
// Un journal agrégé mélange les services et les versions. Sans ces trois
// attributs, une ligne d'erreur ne dit pas QUI l'a émise ni QUELLE version — donc
// on ne peut ni corréler à un déploiement, ni écarter les répliques saines.
func TestEveryRecordCarriesTheServiceIdentity(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &buf).Info("message")

	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &decoded); err != nil {
		t.Fatalf("sortie illisible: %v", err)
	}
	for field, want := range map[string]string{
		"service": "hexa-tests",
		"env":     string(config.EnvProduction),
		"version": "1.2.3",
	} {
		if got, _ := decoded[field].(string); got != want {
			t.Errorf("%s = %q, attendu %q", field, got, want)
		}
	}
}

// TestAnUnknownLevelFallsBackToInfo : un niveau illisible ne coupe pas le journal.
//
// # Pourquoi le repli va vers Info et pas vers Error
//
// C'est le seul repli du dépôt qui n'est PAS un refus, et c'est justifié : replier
// vers Error rendrait un service muet sur une faute de frappe
// (`log_level: infoo`), et un service muet est indiagnosticable. Replier vers
// Debug inonderait la production.
//
// Info est le seul choix qui garde le journal utilisable dans les deux sens.
func TestAnUnknownLevelFallsBackToInfo(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"WARNING": slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"infoo":   slog.LevelInfo,
		"trace":   slog.LevelInfo,
	} {
		if got := parseLevel(raw); got != want {
			t.Errorf("parseLevel(%q) = %v, attendu %v", raw, got, want)
		}
	}
}

// TestTheSourceIsAttachedOnlyAtDebug : l'emplacement du code suit le niveau.
//
// # Ce que ce test verrouille, dans les deux sens
//
// `AddSource: level == slog.LevelDebug` lie deux réglages qui n'ont aucune raison
// évidente d'être liés. Les deux sens comptent :
//
//   - Toujours ACTIF, il ferait payer à chaque ligne de production la capture
//     d'une trace d'appel — le coût de `runtime.Callers`, sur le chemin le plus
//     chaud du service — et publierait l'arborescence des sources dans les
//     journaux.
//   - Jamais actif, le mode debug perdrait ce qui le rend utile : savoir QUELLE
//     ligne a émis le message. On active debug pour comprendre, pas pour avoir
//     plus de texte.
//
// Ce couplage n'est écrit nulle part ailleurs qu'ici.
func TestTheSourceIsAttachedOnlyAtDebug(t *testing.T) {
	t.Parallel()

	var atDebug bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "debug"), &atDebug).Debug("message")
	if !strings.Contains(atDebug.String(), "source") {
		t.Errorf("niveau debug SANS emplacement de source : %s — "+
			"le mode debug perd ce qui le rend utile", atDebug.String())
	}

	var atInfo bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &atInfo).Info("message")
	if strings.Contains(atInfo.String(), "source") {
		t.Errorf("niveau info AVEC emplacement de source : %s — "+
			"chaque ligne de production paierait une capture de pile", atInfo.String())
	}
}

// TestATraceIsAttachedToEveryRecordThatHasOne : le trace_id est injecté par le
// gestionnaire, pas par les appelants.
//
// # Pourquoi ça vaut un test
//
// « Un log sans trace_id est un log qu'on ne pourra pas recouper en incident. »
// Compter sur les appelants pour l'ajouter garantit qu'il manquera exactement là
// où il sert — dans le chemin d'erreur que personne n'a relu.
//
// Le gestionnaire le lit donc du contexte. Ce test vérifie les DEUX sens : présent
// quand il y a un span, ET ABSENT quand il n'y en a pas. Le second compte autant :
// écrire `trace_id: "00000000000000000000000000000000"` sur chaque ligne hors
// requête pollue les index et fait correspondre des lignes sans rapport.
func TestATraceIsAttachedToEveryRecordThatHasOne(t *testing.T) {
	t.Parallel()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("identifiant de trace de test invalide: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("identifiant de span de test invalide: %v", err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
	))

	var withTrace bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &withTrace).
		InfoContext(ctx, "dans une requête")

	emitted := withTrace.String()
	if !strings.Contains(emitted, "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("trace_id absent: %s", emitted)
	}
	if !strings.Contains(emitted, "00f067aa0ba902b7") {
		t.Errorf("span_id absent: %s", emitted)
	}

	var withoutTrace bytes.Buffer
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &withoutTrace).
		InfoContext(context.Background(), "hors requête")

	if strings.Contains(withoutTrace.String(), "trace_id") {
		t.Errorf("trace_id écrit sans span : %s — "+
			"un identifiant nul ferait correspondre des lignes sans rapport", withoutTrace.String())
	}
}

// TestTheTraceHandlerSurvivesAttrsAndGroups : la décoration ne perd pas le span.
//
// # Le défaut que ce test attrape
//
// `WithAttrs` et `WithGroup` doivent RÉENVELOPPER. Rendre `h.inner.WithAttrs(...)`
// directement — l'oubli naturel — retirerait le `traceHandler` de la chaîne, et
// le trace_id disparaîtrait de tout journal décoré.
//
// Or `NewLogger` appelle lui-même `.With(service, env, version)` : le défaut
// toucherait donc la TOTALITÉ des journaux du dépôt, immédiatement, sans que rien
// ne le signale — la ligne serait simplement un peu plus courte.
func TestTheTraceHandlerSurvivesAttrsAndGroups(t *testing.T) {
	t.Parallel()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("identifiant de trace de test invalide: %v", err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
		},
	))

	var buf bytes.Buffer
	// .With() puis .WithGroup() : les deux chemins de décoration, enchaînés.
	newLoggerTo(loggerConfig(config.EnvProduction, "json", "info"), &buf).
		With(slog.String("module", "user_registration")).
		WithGroup("request").
		InfoContext(ctx, "décoré deux fois")

	if !strings.Contains(buf.String(), "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("trace_id perdu après décoration: %s — "+
			"WithAttrs ou WithGroup ne réenveloppe pas le gestionnaire", buf.String())
	}
}

// TestDisabledTelemetryYieldsAnInertShutdown : désactivée, l'arrêt réussit.
//
// # Le défaut que ce test attrape
//
// C'est la configuration PAR DÉFAUT du socle : `telemetry.enabled: false`, aucun
// collecteur. L'arrêt doit rendre nil, sinon chaque arrêt propre d'un binaire
// remonterait une erreur et l'orchestrateur compterait le déploiement en échec.
//
// Le contrat est aussi que l'appelant n'ait PAS à savoir si la télémétrie est
// active : la fonction d'arrêt n'est jamais nil.
func TestDisabledTelemetryYieldsAnInertShutdown(t *testing.T) {
	t.Parallel()

	var cfg config.Config
	cfg.Telemetry.Enabled = false

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup désactivé a rendu une erreur: %v", err)
	}
	if shutdown == nil {
		t.Fatal("fonction d'arrêt nil — l'appelant devrait tester avant d'appeler")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("l'arrêt inerte a rendu %v — chaque déploiement serait compté en échec", err)
	}
}

// TestASuccessfulShutdownReportsNoError : un arrêt RÉUSSI rend nil.
//
// # Le défaut que ce test a mis au jour
//
// L'arrêt enveloppait inconditionnellement : `fmt.Errorf("arrêt de la
// télémétrie: %w", errJoin(...))`. Avec `%w` sur un nil, `fmt.Errorf` ne rend
// PAS nil — il rend une erreur portant littéralement « %!w(<nil>) ».
//
// Un arrêt parfaitement réussi remontait donc une erreur. Le défaut était encore
// LATENT — `Setup` n'est appelée nulle part, la télémétrie n'est pas branchée
// (#13) — mais il se serait déclenché le jour du branchement, sur le chemin le
// plus fréquent qui existe : l'arrêt normal, à chaque déploiement.
//
// Une seconde faute vivait à côté : l'aide s'appelait `errJoin` et ne joignait
// rien, elle rendait la première erreur et jetait la seconde. Les deux sont
// remplacées par `errors.Join`.
//
// # Pourquoi ce test tourne SANS collecteur
//
// gRPC se connecte PARESSEUSEMENT : `otlptracegrpc.New` réussit sans qu'aucun
// collecteur n'écoute, et l'arrêt rend la main en moins d'une milliseconde. Le
// chemin activé est donc exerçable sans infrastructure — ce qui ne rend pas pour
// autant l'EXPORT prouvé : personne n'a vérifié qu'un span arrive réellement à
// un collecteur. Cela reste du ressort de #13.
//
// Pas de `t.Parallel()` : `Setup` installe les fournisseurs GLOBAUX d'OpenTelemetry.
// Un test parallèle qui les lirait verrait un état modifié sous lui.
func TestASuccessfulShutdownReportsNoError(t *testing.T) {
	var cfg config.Config
	cfg.Telemetry.Enabled = true
	cfg.Telemetry.ServiceName = "hexa-tests"
	// Adresse plausible mais sans rien derrière : c'est le point du test.
	cfg.Telemetry.OTLPEndpoint = "127.0.0.1:4317"

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup activé a rendu une erreur: %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("un arrêt réussi a rendu %v — attendu nil.\n"+
			"C'est le défaut du `fmt.Errorf(\"...: %%w\", nil)` : il ne rend pas nil, "+
			"et chaque arrêt propre serait compté en échec", err)
	}
}
