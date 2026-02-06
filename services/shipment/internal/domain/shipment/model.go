package shipment

import (
	"time"
)

// ShipmentStatus represents the lifecycle of a shipment
type ShipmentStatus string

const (
	StatusCreated   ShipmentStatus = "CREATED"
	StatusPickedUp  ShipmentStatus = "PICKED_UP"
	StatusInTransit ShipmentStatus = "IN_TRANSIT"
	StatusDelayed   ShipmentStatus = "DELAYED"
	StatusDelivered ShipmentStatus = "DELIVERED"
	StatusCancelled ShipmentStatus = "CANCELLED"
)

// Shipment represents the Aggregate Root
type Shipment struct {
	ID                    string
	Carrier               string
	TrackingNumber        string
	OriginLocationID      string
	DestinationLocationID string
	Status                ShipmentStatus
	ETA                   time.Time
	DispatchedAt          time.Time
	DeliveredAt           time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Items                 []ShipmentItem
	TrackingLogs          []ShipmentTracking
	StatusHistory         []StatusHistoryChange
}

// ShipmentItem represents specific goods within a shipment
type ShipmentItem struct {
	ShipmentID string
	SKU        string
	Quantity   int
}

// ShipmentTracking represents GPS coordinates for a shipment
type ShipmentTracking struct {
	ShipmentID string
	Sequence   int64
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

// StatusHistoryChange logs the transition of shipment states
type StatusHistoryChange struct {
	ShipmentID string
	Sequence   int64
	OldStatus  ShipmentStatus
	NewStatus  ShipmentStatus
	Reason     string
	ChangedAt  time.Time
}

//==================================================================================================

type UpdateShipment struct {
	Carrier               *string
	TrackingNumber        *string
	OriginLocationID      *string
	DestinationLocationID *string
	Status                *ShipmentStatus
	ETA                   *time.Time
	DispatchedAt          *time.Time
	DeliveredAt           *time.Time
	UpdatedAt             *time.Time
	Items                 []*ShipmentItem
	TrackingLogs          []*ShipmentTracking
	StatusHistory         []*StatusHistoryChange
}

//==================================================================================================

type NewShipment struct {
	Carrier               string
	TrackingNumber        string
	OriginLocationID      string
	DestinationLocationID string
	Status                ShipmentStatus
	ETA                   time.Time
	DispatchedAt          time.Time
	Items                 []ShipmentItem
}
