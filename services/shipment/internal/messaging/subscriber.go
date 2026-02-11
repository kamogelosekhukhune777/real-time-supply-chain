package messaging

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type Subscriber[T proto.Message] struct {
	js    nats.JetStreamContext
	mw    []Middleware[T]
	trace trace.Tracer
	pub   *Publisher
	redis RedisIdempotencyStore
}

func NewSubscriber[T proto.Message](js nats.JetStreamContext, trace trace.Tracer, pub *Publisher, redis RedisIdempotencyStore, mw ...Middleware[T]) *Subscriber[T] {
	return &Subscriber[T]{
		js:    js,
		pub:   pub,
		redis: redis,
		mw:    mw,
		trace: trace,
	}
}

func (s *Subscriber[T]) Consume(
	ctx context.Context, subject, durable, queue string, newMsg func() T, handler Handler[T]) (*nats.Subscription, error) {

	wrapped := wrapMiddleware(s.mw, handler)

	return s.js.QueueSubscribe(subject, queue,
		func(msg *nats.Msg) {
			parentCtx := propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(msg.Header))

			ctx, span := s.trace.Start(parentCtx, "messaging.consume",
				trace.WithAttributes(
					attribute.String("messaging.system", "nats"),
					attribute.String("messaging.destination", subject),
				),
			)
			defer span.End()

			evt := newMsg()
			if err := proto.Unmarshal(msg.Data, evt); err != nil {
				span.RecordError(err)
				_ = msg.Term()
				return
			}

			if err := wrapped(ctx, evt); err != nil {
				span.RecordError(err)

				meta, _ := msg.Metadata()
				if meta != nil && meta.NumDelivered >= 5 {
					_ = s.pub.PublishDLQ(ctx, msg, err.Error())
					_ = msg.Term()
					return
				}

				_ = msg.Nak()
				return
			}

			_ = msg.Ack()
		},
		nats.Durable(durable),
		nats.ManualAck(),
	)
}
