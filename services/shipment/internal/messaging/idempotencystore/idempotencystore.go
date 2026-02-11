package idempotencystore

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyStore struct {
	rdb *redis.Client
}

func NewRedisIdempotencyStore(rdb *redis.Client) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{rdb: rdb}
}

func (r *RedisIdempotencyStore) Seen(ctx context.Context, id string) (bool, error) {
	val, err := r.rdb.Exists(ctx, "evt:"+id).Result()
	return val > 0, err
}

func (r *RedisIdempotencyStore) Mark(ctx context.Context, id string, ttl time.Duration) error {
	return r.rdb.Set(ctx, "evt:"+id, "1", ttl).Err()
}

func (r *RedisIdempotencyStore) TryClaim(ctx context.Context, id string, ttl time.Duration) (bool, error) {
    // SetNX returns true if the key was set, false if it already existed.
    // This is atomic—no other process can slip in between the check and the set.
    return r.rdb.SetNX(ctx, "evt:"+id, "1", ttl).Result()
}
