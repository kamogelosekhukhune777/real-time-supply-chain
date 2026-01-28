package messaging

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	js nats.JetStreamContext
}

func NewPublisher(js nats.JetStreamContext) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hdr := nats.Header{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(hdr))

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = p.js.PublishMsg(
		&nats.Msg{
			Subject: subject,
			Data:    data,
			Header:  hdr,
		},
		nats.Context(pubCtx),
		nats.MsgId(uuid.NewString()),
	)

	return err
}

// DLQ Support
func (p *Publisher) PublishDLQ(ctx context.Context, original *nats.Msg, reason string) error {
	hdr := nats.Header{}
	for k, v := range original.Header {
		hdr[k] = v
	}
	hdr.Set("dlq-reason", reason)
	hdr.Set("dlq-original-subject", original.Subject)

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.js.PublishMsg(
		&nats.Msg{
			Subject: "dlq." + original.Subject,
			Data:    original.Data,
			Header:  hdr,
		},
		nats.Context(pubCtx),
		nats.MsgId(uuid.NewString()),
	)

	return err
}
