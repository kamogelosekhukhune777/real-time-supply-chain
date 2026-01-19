package domain

import (
	"time"
)

type Inventory struct {
	LocationID string    `json:"location_id"`
	SKU        string    `json:"sku"`
	OnHand     int       `json:"on_hand"`
	Reserved   int       `json:"reserved"`
	Incoming   int       `json:"incoming"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpdatedInventory struct {
	LocationID *string `json:"location_id"`
	SKU        *string `json:"sku"`
	OnHand     *int    `json:"on_hand"`
	Reserved   *int    `json:"reserved"`
	Incoming   *int    `json:"incoming"`
}
