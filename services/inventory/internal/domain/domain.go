package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
)

var (
	ErrNotFound          = errors.New("inventory not found")
	ErrAlreadyExists     = errors.New("inventory already exists")
	ErrConcurrentUpdate  = errors.New("concurrent inventory update")
	ErrInsufficientStock = errors.New("insufficient stock to satisfy reservation")
	ErrNegativeQuantity  = errors.New("inventory levels cannot be negative")
)

type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	GetInventory(ctx context.Context, locationID string, sku string) (Inventory, error)
	CreateInventory(ctx context.Context, inv Inventory) error
	UpdateInventory(ctx context.Context, inv Inventory) error
	DeleteInventory(ctx context.Context, inv Inventory) error
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs a home business API for use.
func NewBusiness(log *logger.Logger, storer Storer) *Business {
	b := &Business{
		log:    log,
		storer: storer,
	}

	return b
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

func (b *Business) CreateInventory(ctx context.Context, inv Inventory) (Inventory, error) {
	if inv.OnHand < 0 || inv.Reserved < 0 || inv.Incoming < 0 {
		return Inventory{}, ErrInsufficientStock
	}

	if err := b.storer.CreateInventory(ctx, inv); err != nil {
		return Inventory{}, fmt.Errorf("create inventory: %w", err)
	}

	return inv, nil
}

func (b *Business) UpdateInventory(ctx context.Context, inv Inventory, uinv UpdatedInventory) (Inventory, error) {
	if uinv.OnHand != nil {
		inv.OnHand = *uinv.OnHand
	}
	if uinv.Reserved != nil {
		inv.Reserved = *uinv.Reserved
	}
	if uinv.Incoming != nil {
		inv.Incoming = *uinv.Incoming
	}

	inv.UpdatedAt = time.Now()

	if err := b.storer.UpdateInventory(ctx, inv); err != nil {
		return Inventory{}, fmt.Errorf("update inventory: %w", err)
	}

	return inv, nil
}

func (b *Business) DeleteInventory(ctx context.Context, inv Inventory) error {
	if err := b.storer.DeleteInventory(ctx, inv); err != nil {
		return fmt.Errorf("delete inventory: %w", err)
	}
	return nil
}

func (b *Business) GetInventory(ctx context.Context, locationID string, sku string) (Inventory, error) {
	inv, err := b.storer.GetInventory(ctx, locationID, sku)
	if err != nil {
		return Inventory{}, fmt.Errorf("query inventory: locationID[%s], SKU[%s]: %w", locationID, sku, err)
	}

	return inv, nil
}
