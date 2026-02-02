package invcache

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/inventory/internal/domain"
	"github.com/redis/go-redis/v9"
)

// Store manages the set of APIs for user data and caching.
type Store struct {
	log    *logger.Logger
	storer domain.Storer
	cache  *redis.Client
}

func NewStore(log *logger.Logger, cache *redis.Client, storer domain.Storer) *Store{
	return &Store{
		log:    log,
		storer: storer,
		cache:  cache,
	}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (domain.Storer, error) {
	txStorer, err := s.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log:    s.log,
		storer: txStorer,
		cache:  s.cache,
	}

	return &store, nil

}

func (s *Store) GetInventory(ctx context.Context, locationID string, sku string) (domain.Inventory, error) {
	inv, err := s.readCache(ctx, locationID, sku)
	if err == nil && inv != nil {
		return domain.Inventory{}, nil // Cache Hit!
	}

	dbInv, err := s.storer.GetInventory(ctx, locationID, sku)
	if err != nil {
		return domain.Inventory{}, err
	}

	s.writeCache(ctx, dbInv)

	return dbInv, nil
}

func (s *Store) CreateInventory(ctx context.Context, inv domain.Inventory) error {
	if err := s.storer.CreateInventory(ctx, inv); err != nil {
		return err
	}

	s.writeCache(ctx, inv)

	return nil
}

func (s *Store) UpdateInventory(ctx context.Context, inv domain.Inventory) error {
	if err := s.storer.UpdateInventory(ctx, inv); err != nil {
		return err
	}

	return s.deleteCache(ctx, inv.LocationID, inv.SKU)
}

func (s *Store) DeleteInventory(ctx context.Context, inv domain.Inventory) error {
	if err := s.storer.DeleteInventory(ctx, inv); err != nil {
		return err
	}

	s.deleteCache(ctx, inv.LocationID, inv.SKU)

	return nil
}

//===================================================================================================================================

func inventoryKey(locationID, sku string) string {
	return fmt.Sprintf("inventory:%s:%s", locationID, sku)
}

func (s *Store) writeCache(ctx context.Context, inv domain.Inventory) error {
	key := inventoryKey(inv.LocationID, inv.SKU)

	return s.cache.HSet(ctx, key, map[string]any{
		"on_hand":    inv.OnHand,
		"reserved":   inv.Reserved,
		"incoming":   inv.Incoming,
		"updated_at": inv.UpdatedAt.Unix(),
	}).Err()
}

func (s *Store) readCache(ctx context.Context, locationID, sku string) (*domain.Inventory, error) {
	key := inventoryKey(locationID, sku)

	vals, err := s.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	// HGetAll returns an empty map if the key doesn't exist
	if len(vals) == 0 {
		return nil, redis.Nil
	}

	// Helper to handle parsing errors
	toInt := func(field string) (int, error) {
		val, ok := vals[field]
		if !ok {
			return 0, fmt.Errorf("missing field: %s", field)
		}
		return strconv.Atoi(val)
	}

	onHand, err1 := toInt("on_hand")
	reserved, err2 := toInt("reserved")
	incoming, err3 := toInt("incoming")
	updatedAtUnix, err4 := strconv.ParseInt(vals["updated_at"], 10, 64)

	// If any field is corrupted, treat it as a cache miss so we can fetch from DB
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		s.log.Warn(ctx, "cache_corruption", "key", key)
		return nil, redis.Nil
	}

	return &domain.Inventory{
		LocationID: locationID,
		SKU:        sku,
		OnHand:     int64(onHand),
		Reserved:   int64(reserved),
		Incoming:   int64(incoming),
		UpdatedAt:  time.Unix(updatedAtUnix, 0),
	}, nil
}

func (s *Store) deleteCache(ctx context.Context, locationID, sku string) error {
	key := inventoryKey(locationID, sku)

	return s.cache.Del(ctx, key).Err()
}

// BuildRedisURL constructs a standard redis connection string.
// Format: redis://<user>:<password>@<host>:<port>/<db>
func BuildRedisURL(host string, port int, password string, db int) string {
	auth := ""
	if password != "" {
		auth = fmt.Sprintf(":%s@", url.QueryEscape(password))
	}

	return fmt.Sprintf("redis://%s%s:%d/%d", auth, host, port, db)
}
