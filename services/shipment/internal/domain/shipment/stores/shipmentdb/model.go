package shipmentdb

import (
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment"
)

// Shipment represents the Aggregate Root
type shipmentDB struct {
	ID                    string    `db:"shipment_id"`
	Carrier               string    `db:"carrier"`
	TrackingNumber        string    `db:"tracking_number"`
	OriginLocationID      string    `db:"origin_location_id"`
	DestinationLocationID string    `db:"destination_location_id"`
	Status                string    `db:"status"`
	ETA                   time.Time `db:"eta"` //TODO should be *time.Time to avoid using default values
	DispatchedAt          time.Time `db:"dispatched_at"`
	DeliveredAt           time.Time `db:"delivered_at"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
	Items                 []shipmentItemDB
	TrackingLogs          []shipmentTrackingDB
	StatusHistory         []statusHistoryChangeDB
}

func toDBShipment(shp shipment.Shipment) shipmentDB {
	return shipmentDB{
		ID:                    shp.ID,
		Carrier:               shp.Carrier,
		TrackingNumber:        shp.TrackingNumber,
		OriginLocationID:      shp.OriginLocationID,
		DestinationLocationID: shp.DestinationLocationID,
		Status:                string(shp.Status),
		ETA:                   shp.ETA,
		DispatchedAt:          shp.DispatchedAt,
		DeliveredAt:           shp.DeliveredAt,
		CreatedAt:             shp.CreatedAt,
		UpdatedAt:             shp.UpdatedAt,
		Items:                 toDBShipmentItems(shp.Items),
		TrackingLogs:          toDBShipmentTrackings(shp.TrackingLogs),
		StatusHistory:         toDBStatusHistoryChanges(shp.StatusHistory),
	}
}

func toBusShipment(dbshp shipmentDB) shipment.Shipment {
	return shipment.Shipment{
		ID:                    dbshp.ID,
		Carrier:               dbshp.Carrier,
		TrackingNumber:        dbshp.TrackingNumber,
		OriginLocationID:      dbshp.OriginLocationID,
		DestinationLocationID: dbshp.DestinationLocationID,
		Status:                shipment.ShipmentStatus(dbshp.Status),
		ETA:                   dbshp.ETA,
		DispatchedAt:          dbshp.DispatchedAt,
		DeliveredAt:           dbshp.DeliveredAt,
		CreatedAt:             dbshp.CreatedAt,
		UpdatedAt:             dbshp.UpdatedAt,
		Items:                 toBusShipmentItems(dbshp.Items),
		TrackingLogs:          toBusShipmentTrackings(dbshp.TrackingLogs),
		StatusHistory:         toBusStatusHistoryChanges(dbshp.StatusHistory),
	}
}

//==========================================================================================================

// ShipmentItem represents specific goods within a shipment
type shipmentItemDB struct {
	ShipmentID string `db:"shipment_id"`
	SKU        string `db:"sku"`
	Quantity   int    `db:"quantity"`
}

func toDBShipmentItems(shp []shipment.ShipmentItem) []shipmentItemDB {
	db := make([]shipmentItemDB, len(shp))

	for i, b := range shp {
		db[i] = shipmentItemDB{
			ShipmentID: b.ShipmentID,
			SKU:        b.SKU,
			Quantity:   b.Quantity,
		}
	}

	return db
}

func toBusShipmentItems(items []shipmentItemDB) []shipment.ShipmentItem {
	bus := make([]shipment.ShipmentItem, len(items))

	for i, item := range items {
		bus[i] = shipment.ShipmentItem{
			ShipmentID: item.ShipmentID,
			SKU:        item.SKU,
			Quantity:   item.Quantity,
		}
	}

	return bus
}

//==========================================================================================================

// ShipmentTracking represents GPS coordinates for a shipment
type shipmentTrackingDB struct {
	ShipmentID string    `db:"shipment_id"`
	Sequence   int64     `db:"seq"`
	Latitude   float64   `db:"latitude"`
	Longitude  float64   `db:"longitude"`
	RecordedAt time.Time `db:"recorded_at"`
}

func toDBShipmentTrackings(bus []shipment.ShipmentTracking) []shipmentTrackingDB {
	db := make([]shipmentTrackingDB, len(bus))

	for i, b := range bus {
		db[i] = shipmentTrackingDB{
			ShipmentID: b.ShipmentID,
			Sequence:   b.Sequence,
			Latitude:   b.Latitude,
			Longitude:  b.Longitude,
			RecordedAt: b.RecordedAt,
		}
	}

	return db
}

func toBusShipmentTrackings(dbs []shipmentTrackingDB) []shipment.ShipmentTracking {
	bus := make([]shipment.ShipmentTracking, len(dbs))

	for i, db := range dbs {
		bus[i] = shipment.ShipmentTracking{
			ShipmentID: db.ShipmentID,
			Sequence:   db.Sequence,
			Latitude:   db.Latitude,
			Longitude:  db.Longitude,
			RecordedAt: db.RecordedAt,
		}
	}

	return bus
}

//==========================================================================================================

// StatusHistoryChange logs the transition of shipment states
type statusHistoryChangeDB struct {
	ShipmentID string    `db:"shipment_id"`
	Sequence   int64     `db:"seq"`
	OldStatus  string    `db:"old_status"`
	NewStatus  string    `db:"new_status"`
	Reason     string    `db:"reason"`
	ChangedAt  time.Time `db:"changed_at"`
}

func toDBStatusHistoryChanges(bus []shipment.StatusHistoryChange) []statusHistoryChangeDB {
	db := make([]statusHistoryChangeDB, len(bus))

	for i, b := range bus {
		db[i] = statusHistoryChangeDB{
			ShipmentID: b.ShipmentID,
			Sequence:   b.Sequence,
			OldStatus:  string(b.OldStatus),
			NewStatus:  string(b.NewStatus),
			Reason:     b.Reason,
			ChangedAt:  b.ChangedAt,
		}
	}

	return db
}

func toBusStatusHistoryChanges(dbs []statusHistoryChangeDB) []shipment.StatusHistoryChange {
	bus := make([]shipment.StatusHistoryChange, len(dbs))

	for i, db := range dbs {
		bus[i] = shipment.StatusHistoryChange{
			ShipmentID: db.ShipmentID,
			Sequence:   db.Sequence,
			OldStatus:  shipment.ShipmentStatus(db.OldStatus),
			NewStatus:  shipment.ShipmentStatus(db.NewStatus),
			Reason:     db.Reason,
			ChangedAt:  db.ChangedAt,
		}
	}

	return bus
}
