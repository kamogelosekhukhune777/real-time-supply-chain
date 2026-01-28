package messaging

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

type RedisIdempotencyStore interface {
	Seen(ctx context.Context, id string) (bool, error)
	Mark(ctx context.Context, id string, ttl time.Duration) error
}

// IMPORTANT:
// Returning nil MUST result in the message being ACKed by the subscriber.
func WithIdempotency[T proto.Message](store RedisIdempotencyStore, ttl time.Duration, extractID func(T) string, next Handler[T]) Handler[T] {

	return func(ctx context.Context, evt T) error {
		id := extractID(evt)
		if id == "" {
			return next(ctx, evt)
		}

		seen, _ := store.Seen(ctx, id)
		if seen {
			return nil // safe: subscriber will Ack()
		}

		if err := next(ctx, evt); err != nil {
			return err
		}

		return store.Mark(ctx, id, ttl)
	}
}
