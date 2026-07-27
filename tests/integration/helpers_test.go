//go:build integration

// Package integration éprouve les pilotes contre les SERVICES RÉELS.
//
// # Ce que ce niveau ajoute, et pourquoi il n'était pas optionnel
//
// Huit paquets de pilotes avaient zéro test, à aucun niveau (#37). Le pilote
// `memory` d'`idempotency` a vingt-quatre tests prouvant l'exclusivité ; le
// pilote `postgres`, celui qui tourne en PRODUCTION, en avait zéro.
//
// Ce n'était pas un oubli mais une conséquence : ces pilotes exigent un
// service, donc `go test ./...` sans tag ne peut pas les atteindre. Le tag
// `integration` n'était porté par AUCUN fichier du dépôt, et la CI n'avait pas
// de job correspondant — la dette était nommée, jamais payée.
//
// # Ce que ce niveau ne fait PAS
//
// Il ne rejoue pas les tests de domaine. Les propriétés pures — refus d'une
// clé vide, bornes du recul exponentiel, normalisation — sont déjà vérifiées en
// microsecondes dans `{module}/tests/`. Les répéter ici coûterait un service
// pour ne rien apprendre.
//
// Il vérifie ce qu'eux ne PEUVENT pas vérifier : que le SQL est valide, que les
// contraintes de la migration font ce qu'on croit, et surtout que les
// garanties de CONCURRENCE tiennent — un `FOR UPDATE SKIP LOCKED` ou un verrou
// consultatif ne se testent que contre un vrai moteur.
//
// # Aucun `t.Skip`, jamais
//
// Un test sauté ressemble en tout point à un test réussi : c'est la forme même
// que l'ADR 013 combat. Si le service est injoignable, ces tests ÉCHOUENT en
// nommant la commande qui le démarre. Le faux vert n'a pas sa place ici, et
// surtout pas au niveau qui garde le code de production.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// dsnPostgres rend le DSN de migration, seul rôle autorisé à créer.
//
// `DB_MIGRATION_DSN` et non `DB_DSN` : le rôle applicatif n'a ni CREATE, ni
// ALTER, ni DROP (ADR 011 §4). Ces tests nettoient leurs propres lignes, ce qui
// exige des droits que le rôle applicatif ne doit PAS avoir.
func dsnPostgres(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("DB_MIGRATION_DSN"); dsn != "" {
		return dsn
	}
	// Le même défaut que `config/env/development.yaml`. Ne PAS échouer ici :
	// l'absence de variable n'est pas l'absence de service, et c'est le Ping
	// qui tranche. Échouer sur la variable enverrait chercher au mauvais endroit.
	return "postgres://hexa:hexa@localhost:5432/hexa?sslmode=disable"
}

// pool ouvre un pool Postgres et VÉRIFIE qu'il répond.
//
// Le `Ping` est indispensable : `pgxpool.New` est paresseux et réussit contre
// une adresse morte. Sans lui, le premier échec surviendrait au milieu d'un
// test, et accuserait la requête plutôt que l'absence de service.
func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := pgxpool.New(ctx, dsnPostgres(t))
	if err != nil {
		t.Fatalf("ouverture du pool Postgres: %v", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		t.Fatalf("Postgres injoignable (%v) — démarrer la pile avec `task up`", err)
	}
	t.Cleanup(p.Close)
	return p
}

// redisClient ouvre un client Redis et vérifie qu'il répond.
func redisClient(t *testing.T) *redis.Client {
	t.Helper()

	// Même défaut que `config/data.yaml`.
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("Redis injoignable (%v) — démarrer la pile avec `task up`", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// unique rend un identifiant propre à un test.
//
// Les tests de ce niveau partagent UNE base : ils tournent en parallèle et se
// rejouent. Deux tests qui écrivent la même clé se contamineraient, et le
// symptôme — un échec intermittent — coûte plus cher à diagnostiquer que le
// défaut qu'il masque.
func unique(t *testing.T, prefixe string) string {
	t.Helper()
	return fmt.Sprintf("%s-%s-%d", prefixe, t.Name(), time.Now().UnixNano())
}

// ctxTest rend un contexte borné : un test qui attend un verrou ne doit pas
// bloquer la suite entière.
func ctxTest(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}
