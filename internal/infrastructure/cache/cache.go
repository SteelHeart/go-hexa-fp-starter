// Package cache provides a typed cache on top of Redis.
//
// The API is made of function types (Getter, Setter) so that a decorator can
// receive them without depending on Redis, and so that a test can replace them
// with a closure over a map.
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

// Getter reads a value from the cache. An absence is not an error: it is the
// nominal case of a cache.
type Getter[V any] = func(ctx context.Context, key string) fp.Option[V]

// Setter writes a value into the cache. It returns nothing: a cache that goes
// down must never make a business request fail.
type Setter[V any] = func(ctx context.Context, key string, value V)

// Deleter invalidates an entry.
type Deleter = func(ctx context.Context, key string)

// New opens the Redis client and checks that it answers.
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
		return nil, fmt.Errorf("cache unreachable: %w", err)
	}
	return client, nil
}

// JSON builds a read/write/invalidate trio serialised as JSON.
//
// `namespace` prefixes every key: that is what allows a whole domain to be
// purged without touching the others, and what avoids collisions between
// features that would use the same business key.
func JSON[V any](
	client *redis.Client,
	namespace string,
	ttl time.Duration,
) (get Getter[V], set Setter[V], del Deleter) {
	full := func(key string) string { return namespace + ":" + key }

	get = func(ctx context.Context, key string) fp.Option[V] {
		raw, err := client.Get(ctx, full(key)).Bytes()
		if err != nil {
			// Absence like breakdown: in both cases we do not have the value,
			// and the caller must fall back on the source of truth.
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

// IsMiss reports a missing entry, as opposed to a cache breakdown.
func IsMiss(err error) bool { return errors.Is(err, redis.Nil) }
