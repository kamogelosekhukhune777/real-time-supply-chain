package invdb

import (
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
)

type inventoryDB struct {
	LocationID string    `db:"location_id"`
	SKU        string    `db:"sku"`
	OnHand     int       `db:"on_hand"`
	Reserved   int       `db:"reserved"`
	Incoming   int       `db:"incoming"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func toDBInventory(bus domain.Inventory) inventoryDB {
	return inventoryDB{
		LocationID: bus.LocationID,
		SKU:        bus.SKU,
		OnHand:     bus.OnHand,
		Reserved:   bus.Reserved,
		Incoming:   bus.Incoming,
		UpdatedAt:  bus.UpdatedAt,
	}
}

func toBusInventory(db inventoryDB) (domain.Inventory, error) {
	return domain.Inventory{
		LocationID: db.LocationID,
		SKU:        db.SKU,
		OnHand:     db.OnHand,
		Reserved:   db.Reserved,
		Incoming:   db.Incoming,
		UpdatedAt:  db.UpdatedAt,
	}, nil
}
