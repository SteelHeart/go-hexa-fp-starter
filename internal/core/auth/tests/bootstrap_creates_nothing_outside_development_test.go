package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestBootstrapCreatesNothingOutsideDevelopment est le garde qui rend l'amorçage
// acceptable.
//
// # Ce qu'il empêche
//
// Un compte de démonstration créé en production. Le raccourci n'est tolérable
// QUE parce qu'il n'existe pas ailleurs qu'en local — et « il n'existe pas
// ailleurs » est une affirmation qui se vérifie ou qui se dégrade.
//
// Le test constate les deux moitiés : rien n'est rendu dans le compte rendu, ET
// aucune identité n'existe réellement — parce qu'un compte rendu vide ne prouve
// rien si l'écriture a eu lieu quand même.
func TestBootstrapCreatesNothingOutsideDevelopment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	for _, env := range []config.Environment{config.EnvProduction, config.EnvUAT, "n'importe quoi", ""} {
		mod, _ := newModule(t, nil)

		report, err := auth.Bootstrap(ctx, mod, env)
		if err != nil {
			t.Fatalf("env %q : l'amorçage refuse d'agir, il n'échoue pas — %v", env, err)
		}
		if report.Created || report.Subject != "" || report.Secret != "" {
			t.Fatalf("env %q : l'amorçage a agi hors développement — %+v", env, report)
		}

		// La preuve qui compte : aucun compte n'existe. Un compte rendu vide se
		// falsifie en une ligne ; un magasin peuplé, non.
		_, err = mod.Authenticate(ctx, auth.BootstrapSubject, "n'importe quel secret")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("env %q : un compte d'amorçage existe — %v", env, err)
		}
	}
}

// TestBootstrapOpensASessionInDevelopment referme le délai avant premier succès.
//
// C'est le critère d'acceptation de #99 : un serveur fraîchement démarré doit
// permettre d'obtenir une session. Avant, `POST /v1/auth/sessions` rendait 401
// pour tout le monde, sans exception, faute d'un compte créable.
//
// Le secret est ENGENDRÉ, donc le test ne peut pas le connaître d'avance — il le
// lit dans le compte rendu, ce qui est exactement ce que fera l'exploitant en
// lisant son journal.
func TestBootstrapOpensASessionInDevelopment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	report, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("amorçage: %v", err)
	}
	if !report.Created {
		t.Fatal("l'amorçage devait créer un compte en développement")
	}
	if len(report.Secret) < 12 {
		t.Fatalf("le secret engendré doit satisfaire les bornes du module, obtenu %d caractères", len(report.Secret))
	}

	if _, err := mod.Authenticate(ctx, report.Subject, report.Secret); err != nil {
		t.Fatalf("le compte amorcé doit s'authentifier : %v", err)
	}
}

// TestBootstrapEngendersADistinctSecretEachTime garde l'aléa.
//
// Un secret dérivé du sujet, de l'horloge ou d'une constante serait devinable —
// et un poste de développement est souvent joignable depuis le réseau local. Le
// fait qu'un secret soit « seulement » de développement ne le rend pas moins
// utilisable par quelqu'un d'autre.
func TestBootstrapEngendersADistinctSecretEachTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	seen := make(map[string]bool)

	for range 5 {
		mod, _ := newModule(t, nil)
		report, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
		if err != nil {
			t.Fatalf("amorçage: %v", err)
		}
		if seen[report.Secret] {
			t.Fatalf("secret réutilisé d'un amorçage à l'autre : %q", report.Secret)
		}
		seen[report.Secret] = true
	}
}

// TestBootstrapIsIdempotentAndNeverResetsAnAccount garde le compte existant.
//
// Redémarrer un serveur ne doit pas réinitialiser un compte existant, ni rendre
// un secret qu'on ne connaît pas. Le second appel ne crée rien et ne prétend
// rien : c'est ce qui évite qu'un exploitant croie avoir reçu le mot de passe
// courant alors qu'il lit celui d'un compte jamais créé.
func TestBootstrapIsIdempotentAndNeverResetsAnAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	first, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("premier amorçage: %v", err)
	}

	second, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("second amorçage: %v", err)
	}
	if second.Created || second.Secret != "" {
		t.Fatalf("le second amorçage a agi : %+v", second)
	}

	// Le premier secret vaut TOUJOURS : rien n'a été réinitialisé.
	if _, err := mod.Authenticate(ctx, first.Subject, first.Secret); err != nil {
		t.Fatalf("le compte a été réinitialisé par le second amorçage : %v", err)
	}
}

// TestBootstrapNeverBreaksTheStartupOfADisabledModule garde le démarrage.
//
// # Ce test disait EXACTEMENT le contraire, et il avait tort
//
// Il exigeait que le refus d'un module éteint remonte, au motif qu'un démarrage
// annonçant « compte amorcé » sur un module désactivé enverrait chercher la
// panne du côté du mot de passe. Le raisonnement portait sur le mauvais risque :
// remonter l'erreur **fait échouer le démarrage entier** dès qu'`auth` est
// désactivé.
//
// C'est arrivé. `IsLocal()` couvre `development` ET `test`, alors que
// l'activation ne vient que de la couche `development` : sous `APP_ENV=test`, le
// serveur refusait de démarrer avec `module auth désactivé`. La CI end-to-end
// l'a trouvé ; la mesure locale ne pouvait pas, elle avait porté sur
// `development` et `production` — jamais sur la seule combinaison qui casse,
// **environnement local ET module éteint**.
//
// Un module désactivé n'est pas une panne, c'est un état configuré. Il n'y a
// rien à amorcer, et le dire par une erreur fatale est un fail-closed posé au
// mauvais endroit.
func TestBootstrapNeverBreaksTheStartupOfADisabledModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	disabled, err := auth.New(config.Module{Enabled: false}, auth.Deps{})
	if err != nil {
		t.Fatalf("montage du module désactivé: %v", err)
	}

	for _, env := range []config.Environment{config.EnvDevelopment, config.EnvTest} {
		report, err := auth.Bootstrap(ctx, disabled, env)
		if err != nil {
			t.Fatalf("env %q : un module éteint ne doit pas faire échouer le démarrage — %v", env, err)
		}
		if report.Created || report.Secret != "" {
			t.Fatalf("env %q : un module désactivé ne crée rien — %+v", env, report)
		}
	}
}
