package store


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
