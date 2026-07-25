// Package cache fournit un cache typÃ© au-dessus de Redis.
//
// L'API est faite de types fonction (Getter, Setter) pour qu'un dÃ©corateur
// puisse les recevoir sans dÃ©pendre de Redis, et qu'un test les remplace par
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

// Setter Ã©crit une valeur dans le cache. Elle ne retourne rien : un cache qui
// tombe ne doit jamais faire Ã©chouer une requÃªte mÃ©tier.
type Setter[V any] = func(ctx context.Context, key string, value V)

// Deleter invalide une entrÃ©e.
type Deleter = func(ctx context.Context, key string)

// New ouvre le client Redis et vÃ©rifie qu'il rÃ©pond.
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

// JSON construit un trio lecture/Ã©criture/invalidation sÃ©rialisÃ© en JSON.
//
// `namespace` prÃ©fixe toutes les clÃ©s : c'est ce qui permet de purger un
// domaine entier sans toucher aux autres, et d'Ã©viter les collisions entre
// features qui utiliseraient la mÃªme clÃ© mÃ©tier.
func JSON[V any](
	client *redis.Client,
	namespace string,
	ttl time.Duration,
) (Getter[V], Setter[V], Deleter) {
	full := func(key string) string { return namespace + ":" + key }

	get := func(ctx context.Context, key string) fp.Option[V] {
		raw, err := client.Get(ctx, full(key)).Bytes()
		if err != nil {
			// Absence comme panne : dans les deux cas on n'a pas la valeur, et
			// l'appelant doit se rabattre sur la source de vÃ©ritÃ©.
			return fp.None[V]()
		}
		var value V
		if err := json.Unmarshal(raw, &value); err != nil {
			return fp.None[V]()
		}
		return fp.Some(value)
	}

	set := func(ctx context.Context, key string, value V) {
		raw, err := json.Marshal(value)
		if err != nil {
			return
		}
		_ = client.Set(ctx, full(key), raw, ttl).Err()
	}

	del := func(ctx context.Context, key string) {
		_ = client.Del(ctx, full(key)).Err()
	}

	return get, set, del
}

// IsMiss indique une absence d'entrÃ©e, par opposition Ã  une panne du cache.
func IsMiss(err error) bool { return errors.Is(err, redis.Nil) }
