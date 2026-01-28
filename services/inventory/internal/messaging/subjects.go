package messaging

const (
	ShipmentCreated   = "shipment.created"
	ShipmentCancelled = "shipment.cancelled"
	ShipmentDelivered = "shipment.delivered"

	SaleCompleted = "sale.completed"
	SaleRefunded  = "sale.refunded"

	OrderPlaced   = "order.placed"
	OrderCanceled = "order.cancelled"

	DemandReorderTriggered = "demand.reorder.triggered"
)

const (
	InventoryUpdated             = "inventory.updated"
	InventoryReserved            = "inventory.reserved"
	InventoryReservationReleased = "inventory.reservation.released"
	InventoryLowStock            = "inventory.low.stock"
	InventoryOutOfStock          = "inventory.out.of.stock"
	InventoryReallocated         = "inventory.reallocated"
	InventoryTransferStarted     = "inventory.transfer.started"
	InventoryTransferCompleted   = "inventory.transfer.completed"
	InventoryAdjusted            = "inventory.adjusted"
	InventoryReservationFailed   = "inventory.reservation.failed"
	InventoryShortageDetected    = "inventory.shortage.detected"
)
