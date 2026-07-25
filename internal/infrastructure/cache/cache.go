// Package cache fournit un cache typé au-dessus de Redis.
//
// L'API est faite de types fonction (Getter, Setter) pour qu'un décorateur
// puisse les recevoir sans dépendre de Redis, et qu'un test les remplace par
// une closure sur une map.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// Getter lit une valeur du cache. Une absence n'est pas une erreur : c'est le
// cas nominal d'un cache.
type Getter[V any] = func(ctx context.Context, key string) fp.Option[V]

// Setter écrit une valeur dans le cache. Elle ne retourne rien : un cache qui
// tombe ne doit jamais faire échouer une requête métier.
type Setter[V any] = func(ctx context.Context, key string, value V)

// Deleter invalide une entrée.
type Deleter = func(ctx context.Context, key string)

// New ouvre le client Redis et vérifie qu'il répond.
func New(ctx context.Context, cfg config.Cache) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache injoignable: %w", err)
	}
	return client, nil
}

// JSON construit un trio lecture/écriture/invalidation sérialisé en JSON.
//
// `namespace` préfixe toutes les clés : c'est ce qui permet de purger un
// domaine entier sans toucher aux autres, et d'éviter les collisions entre
// features qui utiliseraient la même clé métier.
func JSON[V any](
	client *redis.Client,
	namespace string,
	ttl time.Duration,
) (get Getter[V], set Setter[V], del Deleter) {
	full := func(key string) string { return namespace + ":" + key }

	get = func(ctx context.Context, key string) fp.Option[V] {
		raw, err := client.Get(ctx, full(key)).Bytes()
		if err != nil {
			// Absence comme panne : dans les deux cas on n'a pas la valeur, et
			// l'appelant doit se rabattre sur la source de vérité.
			return fp.None[V]()
		}
		var value V
		if err := json.Unmarshal(raw, &value); err != nil {
			return fp.None[V]()
		}
		return fp.Some(value)
	}

	set = func(ctx context.Context, key string, value V) {
		raw, err := json.Marshal(value)
		if err != nil {
			return
		}
		_ = client.Set(ctx, full(key), raw, ttl).Err()
	}

	del = func(ctx context.Context, key string) {
		_ = client.Del(ctx, full(key)).Err()
	}

	return get, set, del
}

// IsMiss indique une absence d'entrée, par opposition à une panne du cache.
func IsMiss(err error) bool { return errors.Is(err, redis.Nil) }
