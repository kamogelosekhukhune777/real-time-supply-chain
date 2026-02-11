package messaging

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

type RedisIdempotencyStore interface {
	Seen(ctx context.Context, id string) (bool, error)
	Mark(ctx context.Context, id string, ttl time.Duration) error
	TryClaim(ctx context.Context, id string, ttl time.Duration) (bool, error)
}

func WithTryClaim[T proto.Message](store RedisIdempotencyStore, ttl time.Duration, extractID func(T) string) Middleware[T] {
	return func(next Handler[T]) Handler[T] {
		return func(ctx context.Context, evt T) error {
			id := extractID(evt)
			if id == "" {
				return next(ctx, evt)
			}

			claimed, err := store.TryClaim(ctx, id, ttl)
			if err != nil {
				return err
			}
			if !claimed {
				return nil
			}

			return next(ctx, evt)
		}
	}
}

func WithIdempotency[T proto.Message](store RedisIdempotencyStore, ttl time.Duration, extractID func(T) string, next Handler[T]) Middleware[T] {

	return func(next Handler[T]) Handler[T] {
		return func(ctx context.Context, evt T) error {
			id := extractID(evt)
			if id == "" {
				return next(ctx, evt)
			}

			seen, _ := store.Seen(ctx, id)
			if seen {
				return nil
			}

			if err := next(ctx, evt); err != nil {
				return err
			}

			return store.Mark(ctx, id, ttl)
		}
	}
}
