// Package shipmentdb contains product related CRUD functionality.
package shipmentdb

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment"
)

// Store manages the set of APIs for product database access.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs the api for data access.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

// NewWithTx constructs a new Store value replacing the sqlx DB
// value with a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (shipment.Storer, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log: s.log,
		db:  ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, shp shipment.Shipment) error {
	//shipment
	const q = `
	INSERT INTO shipments
		(shipment_id, carrier, tracking_number, origin_location_id, destination_location_id, status, eta, dispatched_at, delivered_at, created_at, updated_at)
	VALUES
		(:shipment_id, :carrier, :tracking_number, :origin_location_id, :destination_location_id, :status, :eta, :dispatched_at, :delivered_at, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBShipment(shp)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	// shipment item
	const qsi = `
	INSERT INTO shipment_items
		(shipment_id, sku, quantity)
	VALUES
		(:shipment_id, :sku, :quantity)`
	if err := sqldb.NamedExecContext(ctx, s.log, s.db, qsi, toDBShipmentItems(shp.Items)); err != nil {
		return fmt.Errorf("namedexeccontext:%w", err)
	}

	//shipment tracking
	const qst = `
	INSERT INTO shipment_tracking
		(shipment_id, seq, latitude, longitude, recorded_at)
	VALUES
		(:shipment_id, :seq, :latitude, :longitude, :recorded_at)`
	if err := sqldb.NamedExecContext(ctx, s.log, s.db, qst, toDBShipmentTrackings(shp.TrackingLogs)); err != nil {
		return fmt.Errorf("namedexeccontext:%w", err)
	}

	// status history
	const qsh = `
	INSERT INTO shipment_status_history
		(shipment_id, seq, old_status, new_status, reason, changed_at)
	VALUES
		(:shipment_id, :seq, :old_status, :new_status, :reason, :changed_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, qsh, toDBStatusHistoryChanges(shp.StatusHistory)); err != nil {
		return fmt.Errorf("namedexeccontext:%w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, shp shipment.Shipment) error {
	data := struct {
		ID string `db:"shipment_id"`
	}{
		ID: shp.ID,
	}

	const q = `
	DELETE FROM
		shipments
	WHERE
		shipment_id = :shipment_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) QueryByID(ctx context.Context, id string) (shipment.Shipment, error) {
	data := struct {
		ID string `db:"shipment_id"`
	}{
		ID: id,
	}

	const q = `
	SELECT
 		shipment_id, carrier, tracking_number, origin_location_id, destination_location_id, status, eta, dispatched_at, delivered_at, created_at, updated_at
	FROM
		shipments
	WHERE
		shipment_id = :shipment_id`

	var dbShp shipmentDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbShp); err != nil {
		return shipment.Shipment{}, fmt.Errorf("db: %w", err)
	}
	// Shipment Items
	const qi = `
	SELECT
		shipment_id, sku, quantity
	FROM
		shipment_items
	WHERE
		shipment_id = :shipment_id
	`
	var dbItems []shipmentItemDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, qi, data, &dbItems); err != nil {
		return shipment.Shipment{}, fmt.Errorf("db: %w", err)
	}

	// Shipment Tracking Logs
	const qT = `
	SELECT
		shipment_id, seq, latitude, longitude, recorded_at
	FROM
		shipment_tracking
	WHERE
		shipment_id = :shipment_id
	`
	var dbTls []shipmentTrackingDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, qT, data, &dbTls); err != nil {
		return shipment.Shipment{}, fmt.Errorf("db: %w", err)
	}

	// Shipment Status
	const qS = `
	SELECT
		shipment_id, seq, old_status, new_status, reason, changed_at
	FROM
		shipment_status_history
	WHERE
		shipment_id = :shipment_id
	`
	var dbShs []statusHistoryChangeDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, qS, data, &dbShs); err != nil {
		return shipment.Shipment{}, fmt.Errorf("db: %w", err)
	}

	shp := toBusShipment(dbShp)
	shp.Items = toBusShipmentItems(dbItems)
	shp.TrackingLogs = toBusShipmentTrackings(dbTls)
	shp.StatusHistory = toBusStatusHistoryChanges(dbShs)

	return shp, nil
}

func (s *Store) Update(ctx context.Context, shp shipment.Shipment) error {
	// 1. Update shipment header ONLY
	const q = `
	UPDATE shipments
	SET
		carrier = :carrier,
		tracking_number = :tracking_number,
		origin_location_id = :origin_location_id,
		destination_location_id = :destination_location_id,
		status = :status,
		eta = :eta,
		dispatched_at = :dispatched_at,
		delivered_at = :delivered_at,
		updated_at = :updated_at
	WHERE shipment_id = :shipment_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBShipment(shp)); err != nil {
		return err
	}

	// 2. Replace shipment_items
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM shipment_items WHERE shipment_id = $1`, shp.ID); err != nil {
		return err
	}
	if err := sqldb.NamedExecContext(ctx, s.log, s.db,
		`INSERT INTO shipment_items (shipment_id, sku, quantity)
		 VALUES (:shipment_id, :sku, :quantity)`,
		toDBShipmentItems(shp.Items)); err != nil {
		return err
	}

	// Replace tracking logs (or append only)
	//===============================================================================================

	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM shipment_tracking WHERE shipment_id = $1`, shp.ID); err != nil {
		return err
	}

	//===============================================================================================
	if err := sqldb.NamedExecContext(ctx, s.log, s.db,
		`INSERT INTO shipment_tracking (shipment_id, seq, latitude, longitude, recorded_at)
		 VALUES (:shipment_id, :seq, :latitude, :longitude, :recorded_at)`,
		toDBShipmentTrackings(shp.TrackingLogs)); err != nil {
		return err
	}

	// Replace status history (or append only)
	//===============================================================================================
	//???
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM shipment_status_history WHERE shipment_id = $1`, shp.ID); err != nil {
		return err
	}

	//===============================================================================================

	if err := sqldb.NamedExecContext(ctx, s.log, s.db,
		`INSERT INTO shipment_status_history (...) VALUES (...)`,
		toDBStatusHistoryChanges(shp.StatusHistory)); err != nil {
		return err
	}

	return nil
}
