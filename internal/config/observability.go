package config

// Observability porte le paramétrage des quatre signaux, sans nommer de
// fournisseur dans le code.
//
// # Le principe qui rend le choix d'outil réversible
//
// Les signaux sortent par des formats STANDARDS, jamais par des SDK
// propriétaires :
//
//	journaux  → JSON sur stdout   lu par Filebeat/Fluent Bit (ELK), Loki,
//	                              Datadog, CloudWatch — zéro ligne de code
//	traces    → OTLP              Tempo (Grafana), Jaeger, Datadog, Honeycomb,
//	                              Elastic APM
//	métriques → Prometheus / OTLP  Mimir, Prometheus, Datadog, Elastic
//
// Passer d'ELK à Grafana, ou de Jaeger à Datadog, se fait donc dans
// l'infrastructure — collecteur OpenTelemetry, agent de journaux — et jamais
// dans l'application.
//
// Le suivi d'erreurs est le SEUL signal qui exige un client dédié : il est isolé
// derrière un port pour qu'aucun paquet du cœur n'importe son SDK.
type Observability struct {
	Logs    LogSink        `yaml:"logs"`
	Traces  TraceExporter  `yaml:"traces"`
	Metrics MetricExporter `yaml:"metrics"`
	Errors  ErrorReporter  `yaml:"errors"`
}

// LogSink choisit la destination des journaux.
type LogSink struct {
	// stdout | file
	//
	// stdout dans tous les cas conteneurisés : c'est l'agent qui collecte, pas
	// l'application qui pousse. Une application qui écrit un fichier dans un
	// conteneur crée un volume à gérer et un risque de saturation disque.
	Sink     string `yaml:"sink"`
	FilePath string `yaml:"file_path"`
}

// TraceExporter choisit l'exportateur de traces.
type TraceExporter struct {
	// otlp | none
	//
	// Un seul exportateur : le collecteur OpenTelemetry duplique vers plusieurs
	// destinations. Multiplier les exportateurs dans l'application, c'est
	// multiplier les points de panne.
	Exporter string `yaml:"exporter"`
	// grpc | http
	Protocol string `yaml:"protocol"`
	// SampleRatio réduit le volume sous charge. 1.0 en développement.
	SampleRatio float64 `yaml:"sample_ratio"`
}

// MetricExporter choisit l'exportateur de métriques.
type MetricExporter struct {
	// prometheus (tirage) | otlp (poussée) | none
	Exporter string `yaml:"exporter"`
}

// ErrorReporter choisit le rapporteur d'erreurs.
type ErrorReporter struct {
	// none | sentry
	Reporter string `yaml:"reporter"`
	Sentry   Sentry `yaml:"sentry"`
}

// Sentry porte le paramétrage du rapporteur Sentry.
type Sentry struct {
	DSN              string  `yaml:"dsn"`
	TracesSampleRate float64 `yaml:"traces_sample_rate"`
	// SendDefaultPII doit rester false : les données personnelles ne sortent pas
	// du système (rules/securite.md §5). La validation le refuse hors local.
	SendDefaultPII bool `yaml:"send_default_pii"`
}

// applyObservabilityDefaults comble les valeurs absentes par les choix les plus
// sûrs : stdout, OTLP, Prometheus, aucun rapporteur d'erreurs.
func (o *Observability) applyDefaults() {
	if o.Logs.Sink == "" {
		o.Logs.Sink = "stdout"
	}
	if o.Traces.Exporter == "" {
		o.Traces.Exporter = "otlp"
	}
	if o.Traces.Protocol == "" {
		o.Traces.Protocol = "grpc"
	}
	if o.Traces.SampleRatio == 0 {
		o.Traces.SampleRatio = 1.0
	}
	if o.Metrics.Exporter == "" {
		o.Metrics.Exporter = "prometheus"
	}
	if o.Errors.Reporter == "" {
		o.Errors.Reporter = "none"
	}
}

// validateObservability vérifie la cohérence des choix d'outillage.
func (o Observability) validate(local bool) []error {
	var problems []error

	problems = appendUnlessOneOf(problems, "observability.logs.sink", o.Logs.Sink, "stdout", "file")
	problems = appendUnlessOneOf(problems, "observability.traces.exporter", o.Traces.Exporter, "otlp", "none")
	problems = appendUnlessOneOf(problems, "observability.traces.protocol", o.Traces.Protocol, "grpc", "http")
	problems = appendUnlessOneOf(problems, "observability.metrics.exporter", o.Metrics.Exporter,
		"prometheus", "otlp", "none")
	problems = appendUnlessOneOf(problems, "observability.errors.reporter", o.Errors.Reporter, "none", "sentry")

	if o.Traces.SampleRatio < 0 || o.Traces.SampleRatio > 1 {
		problems = append(problems, errorf("observability.traces.sample_ratio doit être entre 0 et 1"))
	}
	if o.Errors.Reporter == "sentry" && o.Errors.Sentry.DSN == "" {
		problems = append(problems, errorf("observability.errors.sentry.dsn est requis quand le rapporteur est sentry"))
	}
	if !local && o.Errors.Sentry.SendDefaultPII {
		problems = append(problems, errorf(
			"observability.errors.sentry.send_default_pii doit être false : "+
				"les données personnelles ne sortent pas du système"))
	}
	if !local && o.Logs.Sink == "file" {
		problems = append(problems, errorf(
			"observability.logs.sink=file est refusé hors développement : "+
				"en conteneur, c'est l'agent qui collecte stdout"))
	}
	return problems
}
