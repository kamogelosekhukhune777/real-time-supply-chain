package shipment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
)

type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	QueryByID(ctx context.Context, Id string) (Shipment, error)
	Create(ctx context.Context, shp Shipment) error
	Update(ctx context.Context, shp Shipment) error
	Delete(ctx context.Context, shp Shipment) error

	//ListByStatus(status ShipmentStatus) ([]Shipment, error)
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// NewWithTx constructs a new domain value that will use the
// specified transaction in any store related calls.
func (b *Business) NewWithTx(tx sqldb.CommitRollbacker) (*Business, error) {
	storer, err := b.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	nb := NewBusiness(b.log, storer)

	return nb, nil
}

func (b *Business) Create(ctx context.Context, nshp NewShipment) (Shipment, error) {
	shp := Shipment{
		ID:                    uuid.New().String(),
		Carrier:               nshp.Carrier,
		TrackingNumber:        nshp.TrackingNumber,
		OriginLocationID:      nshp.OriginLocationID,
		DestinationLocationID: nshp.DestinationLocationID,
		Status:                nshp.Status,
		ETA:                   nshp.ETA,
		CreatedAt:             time.Now(),
		DispatchedAt:          nshp.DispatchedAt,
	}

	for _, item := range nshp.Items {
		shp.Items = append(shp.Items, item)
	}

	if err := b.storer.Create(ctx, shp); err != nil {
		return Shipment{}, fmt.Errorf("create shipment: %w", err)
	}

	return shp, nil
}

func (b *Business) Update(ctx context.Context, shp Shipment, ushp UpdateShipment) (Shipment, error) {
	if ushp.Carrier != nil {
		shp.Carrier = *ushp.Carrier
	}

	if ushp.TrackingNumber != nil {
		shp.TrackingNumber = *ushp.TrackingNumber
	}

	if ushp.OriginLocationID != nil {
		shp.OriginLocationID = *ushp.OriginLocationID
	}

	if ushp.DestinationLocationID != nil {
		shp.DestinationLocationID = *ushp.DestinationLocationID
	}

	if ushp.DispatchedAt != nil {
		shp.DispatchedAt = *ushp.DispatchedAt
	}

	if ushp.Status != nil {
		shp.Status = *ushp.Status
	}

	if ushp.ETA != nil {
		shp.ETA = *ushp.ETA
	}

	if ushp.Items != nil {
		shp.Items = make([]ShipmentItem, 0, len(ushp.Items))
		for _, item := range ushp.Items {
			shp.Items = append(shp.Items, *item)
		}
	}

	if ushp.TrackingLogs != nil {
		for _, log := range ushp.TrackingLogs {
			shp.TrackingLogs = append(shp.TrackingLogs, *log)
		}
	}

	if ushp.StatusHistory != nil {
		for _, status := range ushp.StatusHistory {
			shp.StatusHistory = append(shp.StatusHistory, *status)
		}
	}

	shp.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, shp); err != nil {
		return Shipment{}, fmt.Errorf("update shipment: %w", err)
	}

	return shp, nil
}

func (b *Business) Delete(ctx context.Context, shp Shipment) error {
	if err := b.storer.Delete(ctx, shp); err != nil {
		return fmt.Errorf("delete shipment: %w", err)
	}

	return nil
}

func (b *Business) QueryById(ctx context.Context, id string) (Shipment, error) {
	shp, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Shipment{}, fmt.Errorf("query shipment: ID[%s] : %w", id, err)
	}

	return shp, nil
}
