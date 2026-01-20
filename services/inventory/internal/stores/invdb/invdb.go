package invdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
)

// ==================================================================
//
// Store manages the set of APIs for user database access.
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
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (*Store, error) {
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

func (s *Store) GetInventory(ctx context.Context, locationID string, sku string) (domain.Inventory, error) {
	data := struct {
		LocationID string `db:"location_id"`
		SKU        string `db:"sku"`
	}{
		LocationID: locationID,
		SKU:        sku,
	}

	const q = `
	SELECT
        location_id, sku, on_hand, reserved, incoming, updated_at
	FROM
		inventory
	WHERE
		location_id = :location_id AND sku = :sku`

	var dbInv inventoryDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbInv); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return domain.Inventory{}, fmt.Errorf("db: %w", domain.ErrNotFound)
		}
		return domain.Inventory{}, fmt.Errorf("db: %w", err)
	}

	return toBusInventory(dbInv)
}

func (s *Store) CreateInventory(ctx context.Context, inv domain.Inventory) (domain.Inventory, error) {
	const q = `
	INSERT INTO inventory
		( location_id, sku, on_hand, reserved, incoming, updated_at)
	VALUES
		(:location_id, :sku, :on_hand, :reserved, :incoming, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBInventory(inv)); err != nil {
		return domain.Inventory{}, fmt.Errorf("named exec context: %w", err)
	}

	return inv, nil
}

func (s *Store) UpdateInventory(ctx context.Context, inv domain.Inventory) (domain.Inventory, error) {
	const q = `
	UPDATE
		inventory
	SET
		"location_id" = :location_id,
		"sku" = :sku,
		"on_hand" = :on_hand,
		"reserved" = :reserved,
		"incoming" = :incoming,
		"updated_at" = :updated_at
	WHERE
		location_id = :location_id AND sku = :sku`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBInventory(inv)); err != nil {
		return domain.Inventory{}, fmt.Errorf("named exec context: %w", err)
	}

	return inv, nil
}

func (s *Store) DeleteInventory(ctx context.Context, inv domain.Inventory) error {
	const q = `
	DELETE FROM
		inventory
	WHERE
		location_id = :location_id AND sku = :sku`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBInventory(inv)); err != nil {
		return fmt.Errorf("named exec context: %w", err)
	}

	return nil
}
