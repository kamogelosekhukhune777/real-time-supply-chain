package messaging

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"
)

type Handler[T proto.Message] func(ctx context.Context, evt T) error

type Subscriber[T proto.Message] struct {
	js  nats.JetStreamContext
	pub *Publisher
}

func NewSubscriber[T proto.Message](js nats.JetStreamContext, pub *Publisher) *Subscriber[T] {
	return &Subscriber[T]{js: js, pub: pub}
}

func (s *Subscriber[T]) Consumes(ctx context.Context, subject, durable, queue string, newMsg func() T, handler Handler[T]) error {

	_, err := s.js.QueueSubscribe(subject, queue,
		func(msg *nats.Msg) {

			defer func() {
				if r := recover(); r != nil {
					_ = s.pub.PublishDLQ(ctx, msg, "panic")
					_ = msg.Term()
				}
			}()

			parentCtx := otel.GetTextMapPropagator().
				Extract(ctx, propagation.HeaderCarrier(msg.Header))

			msgCtx, span := otel.GetTracerProvider().
				Tracer("nats").
				Start(parentCtx, "nats.consume")
			defer span.End()

			evt := newMsg()
			if err := proto.Unmarshal(msg.Data, evt); err != nil {
				_ = s.pub.PublishDLQ(msgCtx, msg, "unmarshal_error")
				_ = msg.Term()
				return
			}

			execCtx, cancel := context.WithTimeout(msgCtx, 30*time.Second)
			defer cancel()

			if err := handler(execCtx, evt); err != nil {
				meta, _ := msg.Metadata()
				if meta != nil && meta.NumDelivered >= 5 {
					_ = s.pub.PublishDLQ(msgCtx, msg, err.Error())
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
		nats.AckWait(60*time.Second),
		nats.MaxAckPending(1024),
	)

	return err
}

func (s *Subscriber[T]) Consume(
	ctx context.Context,
	subject, durable, queue string,
	newMsg func() T,
	handler Handler[T],
) (*nats.Subscription, error) {

	sub, err := s.js.QueueSubscribe(subject, queue,
		func(msg *nats.Msg) {

			defer func() {
				if r := recover(); r != nil {
					_ = s.pub.PublishDLQ(ctx, msg, "panic")
					_ = msg.Term()
				}
			}()

			parentCtx := otel.GetTextMapPropagator().
				Extract(ctx, propagation.HeaderCarrier(msg.Header))

			msgCtx, span := otel.GetTracerProvider().
				Tracer("nats").
				Start(parentCtx, "nats.consume")
			defer span.End()

			evt := newMsg()
			if err := proto.Unmarshal(msg.Data, evt); err != nil {
				_ = s.pub.PublishDLQ(msgCtx, msg, "unmarshal_error")
				_ = msg.Term()
				return
			}

			execCtx, cancel := context.WithTimeout(msgCtx, 30*time.Second)
			defer cancel()

			if err := handler(execCtx, evt); err != nil {
				meta, _ := msg.Metadata()
				if meta != nil && meta.NumDelivered >= 5 {
					_ = s.pub.PublishDLQ(msgCtx, msg, err.Error())
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
		nats.AckWait(60*time.Second),
		nats.MaxAckPending(1024),
	)

	return sub, err
}
