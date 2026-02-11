package messaging

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	log   *logger.Logger
	trace trace.Tracer
	js    nats.JetStreamContext
}

func NewPublisher(log *logger.Logger, js nats.JetStreamContext, tracer trace.Tracer) *Publisher {
	return &Publisher{
		log:   log,
		js:    js,
		trace: tracer,
	}
}

func (p *Publisher) Publish(ctx context.Context, subject string, msg proto.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ctx, span := p.trace.Start(ctx, "messaging.publish",
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination", subject),
		),
	)
	defer span.End()

	data, err := proto.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "proto marshal failed")
		return err
	}

	hdr := nats.Header{}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(hdr))

	_, err = p.js.PublishMsg(
		&nats.Msg{
			Subject: subject,
			Data:    data,
			Header:  hdr,
		},
		nats.Context(ctx),
		nats.MsgId(uuid.NewString()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "published")
	return nil
}

func (p *Publisher) PublishDLQ(ctx context.Context, original *nats.Msg, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	parts := strings.SplitN(original.Subject, ".", 2)
	dlqSubject := "dlq." + original.Subject
	if len(parts) > 1 {
		dlqSubject = fmt.Sprintf("%s.dlq.%s", parts[0], parts[1])
	}

	ctx, span := p.trace.Start(ctx, "messaging.publish.dlq",
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.dlq.reason", reason),
			attribute.String("messaging.original.subject", original.Subject),
			attribute.String("messaging.destination", dlqSubject),
		),
	)
	defer span.End()

	hdr := nats.Header{}
	for k, v := range original.Header {
		hdr[k] = append([]string(nil), v...)
	}

	hdr.Set("dlq-reason", reason)
	hdr.Set("dlq-original-subject", original.Subject)
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(hdr))

	_, err := p.js.PublishMsg(
		&nats.Msg{
			Subject: dlqSubject,
			Data:    original.Data,
			Header:  hdr,
		},
		nats.Context(ctx),
		nats.MsgId(uuid.NewString()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "published to dlq")
	return nil
}
