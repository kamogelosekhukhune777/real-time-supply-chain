package service

import (
	"context"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/messaging"
	order "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/order/v1"
	shipment "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/shipment/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type Service struct {
	log    *logger.Logger
	invBus *domain.Business
	js     nats.JetStreamContext
	pub    *messaging.Publisher
	idem   messaging.RedisIdempotencyStore
	subs   []*nats.Subscription
}

func NewInventoryService(log *logger.Logger, invBus *domain.Business, js nats.JetStreamContext, pub *messaging.Publisher, idem messaging.RedisIdempotencyStore) *Service {
	return &Service{
		log:    log,
		invBus: invBus,
		js:     js,
		pub:    pub,
		idem:   idem,
	}
}

func (s *Service) Start(ctx context.Context) error {

	// ORDER PLACED
	sub, err := consume(ctx, s, messaging.OrderPlaced, "inventory-order-placed", "inventory", func() *order.OrderPlaced { return &order.OrderPlaced{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderPlaced) string { return e.OrderId },
			s.handleOrderPlaced,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub)

	// ORDER CANCELLED
	sub1, err := consume(ctx, s, messaging.OrderCanceled, "inventory-order-cancelled", "inventory", func() *order.OrderCancelled { return &order.OrderCancelled{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderCancelled) string { return e.OrderId },
			s.handleOrderCancelled,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub1)

	//ORDER SHIPPED
	sub2, err := consume(ctx, s, messaging.OrderShipped, "inventory-order-shipped", "inventory", func() *order.OrderShipped { return &order.OrderShipped{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderShipped) string { return e.OrderId },
			s.handleOrderShipped,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub2)

	//SHIPMENT CREATED
	sub3, err := consume(ctx, s, messaging.ShipmentCreated, "inventory-shipment-created", "inventory", func() *shipment.ShipmentCreated { return &shipment.ShipmentCreated{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentCreated) string { return e.ShipmentId },
			s.handleShipmentCreated,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub3)

	//SHIPMENT DELIVERED
	sub4, err := consume(ctx, s, messaging.ShipmentDelivered, "inventory-shipment-delivered", "inventory", func() *shipment.ShipmentDelivered { return &shipment.ShipmentDelivered{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentDelivered) string { return e.ShipmentId },
			s.handleShipmentDelivered,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub4)

	//SHIPMENT CANCELLED
	sub5, err := consume(ctx, s, messaging.ShipmentCancelled, "inventory-shipment-cancelled", "inventory", func() *shipment.ShipmentCancelled { return &shipment.ShipmentCancelled{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentCancelled) string { return e.ShipmentId },
			s.handleShipmentCancelled,
		),
	)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, sub5)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if len(s.subs) == 0 {
		return nil
	}

	s.log.Info(ctx, "stopping inventory service consumers...")

	done := make(chan struct{})
	go func() {
		defer close(done)

		for _, sub := range s.subs {
			if sub == nil {
				continue
			}

			// Drain = stop receiving new msgs, process inflight before closing
			if err := sub.Drain(); err != nil {
				s.log.Warn(ctx, "failed to drain subscription, forcing unsubscribe", err)
				_ = sub.Unsubscribe()
				continue
			}

			// Ensure fully closed
			_ = sub.Unsubscribe()
		}
	}()

	select {
	case <-ctx.Done():
		s.log.Warn(ctx, "stop cancelled by context, forcing unsubscribe on remaining consumers")
		for _, sub := range s.subs {
			if sub != nil {
				_ = sub.Unsubscribe()
			}
		}
		return ctx.Err()

	case <-done:
		s.subs = nil
		s.log.Info(ctx, "inventory service stopped successfully")
		return nil
	}
}

//======================================================================================================================================================

func consume[T proto.Message](
	ctx context.Context,
	s *Service,
	subject string,
	durable string,
	queue string,
	newMsg func() T,
	handler messaging.Handler[T],
) (*nats.Subscription, error) {

	sub := messaging.NewSubscriber[T](s.js, s.pub)
	return sub.Consume(ctx, subject, durable, queue, newMsg, handler)
}
