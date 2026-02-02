package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/messaging"
	common "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/common/v1"
	inventory "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/inventory/v1"
	order "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/order/v1"
	shipment "github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/service/events/shipment/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newMetadata(eventType string) *common.EventMetadata {
	return &common.EventMetadata{
		EventId:       uuid.NewString(),
		EventType:     eventType,
		OccurredAt:    timestamppb.Now(),
		Producer:      "inventory-service",
		SchemaVersion: 1,
	}
}

//======================================================= ORDER ======================================================================

func (s *Service) handleOrderPlaced(ctx context.Context, evt *order.OrderPlaced) error {
	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	available := inv.OnHand - inv.Reserved
	if available < evt.Quantity {
		if err := s.pub.Publish(ctx, messaging.InventoryReservationFailed,
			&inventory.InventoryReservationFailed{
				Metadata:          newMetadata(messaging.InventoryReservationFailed),
				WarehouseId:       evt.WarehouseId,
				Sku:               evt.Sku,
				OrderId:           evt.OrderId,
				RequestedQuantity: evt.Quantity,
				AvailableQuantity: available,
				Reason:            inventory.InventoryReservationFailed_INSUFFICIENT_STOCK,
			},
		); err != nil {
			return err
		}
		return nil
	}

	newReserved := inv.Reserved + evt.Quantity
	updated, err := s.invBus.UpdateInventory(ctx, inv, domain.UpdatedInventory{
		Reserved: &newReserved,
	})
	if err != nil {
		return err
	}

	reservationID := uuid.NewString()

	if err := s.pub.Publish(ctx, messaging.InventoryReserved,
		&inventory.InventoryReserved{
			Metadata:               newMetadata(messaging.InventoryReserved),
			WarehouseId:            evt.WarehouseId,
			Sku:                    evt.Sku,
			ReservationId:          reservationID,
			OrderId:                evt.OrderId,
			ReservedQuantity:       evt.Quantity,
			QuantityAvailableAfter: updated.OnHand - updated.Reserved,
			QuantityReservedAfter:  updated.Reserved,
		},
	); err != nil {
		return err
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated, &inventory.InventoryUpdated{
		Metadata:          newMetadata(messaging.InventoryUpdated),
		WarehouseId:       updated.LocationID,
		Sku:               updated.SKU,
		QuantityAvailable: updated.OnHand - updated.Reserved,
		QuantityReserved:  updated.Reserved,
		QuantityIncoming:  updated.Incoming,
		Reason:            inventory.InventoryUpdated_RESERVATION,
	},
	); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleOrderCancelled(ctx context.Context, evt *order.OrderCancelled) error {
	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	if inv.Reserved == 0 {
		return nil
	}

	releaseQty := evt.Quantity
	if releaseQty > inv.Reserved {
		releaseQty = inv.Reserved
	}

	newReserved := inv.Reserved - releaseQty

	updated, err := s.invBus.UpdateInventory(ctx, inv,
		domain.UpdatedInventory{
			Reserved: &newReserved,
		},
	)
	if err != nil {
		return err
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated,
		&inventory.InventoryUpdated{
			Metadata:          newMetadata(messaging.InventoryUpdated),
			WarehouseId:       updated.LocationID,
			Sku:               updated.SKU,
			QuantityAvailable: updated.OnHand - updated.Reserved,
			QuantityReserved:  updated.Reserved,
			QuantityIncoming:  updated.Incoming,
			Reason:            inventory.InventoryUpdated_RELEASE,
		},
	); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleOrderShipped(ctx context.Context, evt *order.OrderShipped) error {
	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	if inv.Reserved == 0 {
		return nil
	}

	shipQty := evt.Quantity

	if shipQty > inv.Reserved {
		shipQty = inv.Reserved
	}
	if shipQty > inv.OnHand {
		shipQty = inv.OnHand
	}

	newReserved := inv.Reserved - shipQty
	newOnHand := inv.OnHand - shipQty

	updated, err := s.invBus.UpdateInventory(ctx, inv,
		domain.UpdatedInventory{
			Reserved: &newReserved,
			OnHand:   &newOnHand,
		},
	)
	if err != nil {
		return err
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated,
		&inventory.InventoryUpdated{
			Metadata:          newMetadata(messaging.InventoryUpdated),
			WarehouseId:       updated.LocationID,
			Sku:               updated.SKU,
			QuantityAvailable: updated.OnHand - updated.Reserved,
			QuantityReserved:  updated.Reserved,
			QuantityIncoming:  updated.Incoming,
			Reason:            inventory.InventoryUpdated_SHIPMENT_DISPATCHED,
		},
	); err != nil {
		return err
	}

	return nil
}

//======================================================= SHIPMENT ======================================================================

func (s *Service) handleShipmentCreated(ctx context.Context, evt *shipment.ShipmentCreated) error {
	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	if inv.Reserved < evt.Quantity {
		return nil
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated,
		&inventory.InventoryUpdated{
			Metadata:          newMetadata(messaging.InventoryUpdated),
			WarehouseId:       inv.LocationID,
			Sku:               inv.SKU,
			QuantityAvailable: inv.OnHand - inv.Reserved,
			QuantityReserved:  inv.Reserved,
			QuantityIncoming:  inv.Incoming,
			Reason:            inventory.InventoryUpdated_SHIPMENT_CREATED, //??ShipmentCreated
		},
	); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleShipmentCancelled(ctx context.Context, evt *shipment.ShipmentCancelled) error {
	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	restoreQty := evt.Quantity
	if restoreQty <= 0 {
		return nil
	}

	newOnHand := inv.OnHand + restoreQty

	updated, err := s.invBus.UpdateInventory(
		ctx,
		inv,
		domain.UpdatedInventory{
			OnHand: &newOnHand,
		},
	)
	if err != nil {
		return err
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated,
		&inventory.InventoryUpdated{
			Metadata:          newMetadata(messaging.InventoryUpdated),
			WarehouseId:       updated.LocationID,
			Sku:               updated.SKU,
			QuantityAvailable: updated.OnHand - updated.Reserved,
			QuantityReserved:  updated.Reserved,
			QuantityIncoming:  updated.Incoming,
			Reason:            inventory.InventoryUpdated_SHIPMENT_CANCELLED,
		},
	); err != nil {
		return err
	}

	return nil
}

func (s *Service) handleShipmentDelivered(ctx context.Context, evt *shipment.ShipmentDelivered) error {

	inv, err := s.invBus.GetInventory(ctx, evt.WarehouseId, evt.Sku)
	if err != nil {
		return err
	}

	if err := s.pub.Publish(ctx, messaging.InventoryUpdated,
		&inventory.InventoryUpdated{
			Metadata:          newMetadata(messaging.InventoryUpdated),
			WarehouseId:       inv.LocationID,
			Sku:               inv.SKU,
			QuantityAvailable: inv.OnHand - inv.Reserved,
			QuantityReserved:  inv.Reserved,
			QuantityIncoming:  inv.Incoming,
			Reason:            inventory.InventoryUpdated_SHIPMENT_DELIVERED,
		},
	); err != nil {
		return err
	}

	return nil
}
