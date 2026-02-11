package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/metrics"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

func Panics[T proto.Message](pub *Publisher) Middleware[T] {
	return func(next Handler[T]) Handler[T] {
		return func(ctx context.Context, evt T) (err error) {
			defer func() {
				if r := recover(); r != nil {
					metrics.AddPanic(ctx)

					span := trace.SpanFromContext(ctx)
					span.RecordError(fmt.Errorf("panic: %v", r))

					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx, evt)
		}
	}
}

func Timeout[T proto.Message](d time.Duration) Middleware[T] {
	return func(next Handler[T]) Handler[T] {
		return func(ctx context.Context, evt T) error {
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			return next(ctx, evt)
		}
	}
}

func Metrics[T proto.Message]() Middleware[T] {
	return func(next Handler[T]) Handler[T] {
		return func(ctx context.Context, evt T) error {
			ctx = metrics.Inject(ctx)

			err := next(ctx, evt)

			n := metrics.AddRequest(ctx)
			if n%1000 == 0 {
				metrics.UpdateGoroutines(ctx)
			}

			if err != nil {
				metrics.AddError(ctx)
			}

			return err
		}
	}
}
