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
	if err := consume(ctx, s, messaging.OrderPlaced, "inventory-order-placed", "inventory", func() *order.OrderPlaced { return &order.OrderPlaced{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderPlaced) string { return e.OrderId },
			s.handleOrderPlaced,
		),
	); err != nil {
		return err
	}

	// ORDER CANCELLED
	if err := consume(ctx, s, messaging.OrderCanceled, "inventory-order-cancelled", "inventory", func() *order.OrderCancelled { return &order.OrderCancelled{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderCancelled) string { return e.OrderId },
			s.handleOrderCancelled,
		),
	); err != nil {
		return err
	}

	//ORDER SHIPPED
	if err := consume(ctx, s, messaging.OrderShipped, "inventory-order-shipped", "inventory", func() *order.OrderShipped { return &order.OrderShipped{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *order.OrderShipped) string { return e.OrderId },
			s.handleOrderShipped,
		),
	); err != nil {
		return err
	}

	//SHIPMENT CREATED
	if err := consume(ctx, s, messaging.ShipmentCreated, "inventory-shipment-created", "inventory", func() *shipment.ShipmentCreated { return &shipment.ShipmentCreated{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentCreated) string { return e.ShipmentId },
			s.handleShipmentCreated,
		),
	); err != nil {
		return err
	}

	//SHIPMENT DELIVERED
	if err := consume(ctx, s, messaging.ShipmentDelivered, "inventory-shipment-delivered", "inventory", func() *shipment.ShipmentDelivered { return &shipment.ShipmentDelivered{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentDelivered) string { return e.ShipmentId },
			s.handleShipmentDelivered,
		),
	); err != nil {
		return err
	}

	//SHIPMENT CANCELLED
	if err := consume(ctx, s, messaging.ShipmentCancelled, "inventory-shipment-cancelled", "inventory", func() *shipment.ShipmentCancelled { return &shipment.ShipmentCancelled{} },
		messaging.WithIdempotency(
			s.idem,
			24*time.Hour,
			func(e *shipment.ShipmentCancelled) string { return e.ShipmentId },
			s.handleShipmentCancelled,
		),
	); err != nil {
		return err
	}

	return nil
}

//======================================================================================================================================================

func consume[T proto.Message](ctx context.Context, s *Service, subject string, durable string, queue string, newMsg func() T, handler messaging.Handler[T]) error {

	sub := messaging.NewSubscriber[T](s.js, s.pub)
	return sub.Consume(ctx, subject, durable, queue, newMsg, handler)
}
