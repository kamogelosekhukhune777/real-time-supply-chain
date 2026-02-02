package domain

import (
	"time"
)

type Inventory struct {
	LocationID string    `json:"location_id"`
	SKU        string    `json:"sku"`
	OnHand     int64     `json:"on_hand"`
	Reserved   int64     `json:"reserved"`
	Incoming   int64     `json:"incoming"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpdatedInventory struct {
	LocationID *string `json:"location_id"`
	SKU        *string `json:"sku"`
	OnHand     *int64  `json:"on_hand"`
	Reserved   *int64  `json:"reserved"`
	Incoming   *int64  `json:"incoming"`
}
