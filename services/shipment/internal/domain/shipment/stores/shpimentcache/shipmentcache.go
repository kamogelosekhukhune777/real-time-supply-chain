package shipmentcache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/logger"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/pkg/sdks/sqldb"
	"github.com/kamogelosekhukhune777/real-time-supply-chain/services/shipment/internal/domain/shipment"
	"github.com/redis/go-redis/v9"
)

// Store manages the set of APIs for user data and caching.
type Store struct {
	log    *logger.Logger
	storer shipment.Storer
	cache  *redis.Client
}

func NewStore(log *logger.Logger, cache *redis.Client, storer shipment.Storer) *Store {
	return &Store{
		log:    log,
		storer: storer,
		cache:  cache,
	}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (shipment.Storer, error) {
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

func (s *Store) QueryByID(ctx context.Context, id string) (shipment.Shipment, error) {
	shp, err := s.readCache(ctx, id)
	if err == nil {
		return shp, nil
	}

	if err != redis.Nil {
		s.log.Error(ctx, "redis error", "error", err)
	}

	dbShp, err := s.storer.QueryByID(ctx, id)
	if err != nil {
		return shipment.Shipment{}, err
	}

	if err := s.writeCache(ctx, dbShp); err != nil {
		s.log.Error(ctx, "failed to populate cache", "err", err)
	}

	return dbShp, nil
}

func (s *Store) Create(ctx context.Context, shp shipment.Shipment) error {
	if err := s.storer.Create(ctx, shp); err != nil {
		return err
	}

	//in case of a rollback(transaction) the shipment in cache won't be available in the
	// Database(that's) an error?? so:
	// return s.storer.Create(ctx, shp)
	return s.writeCache(ctx, shp)
}

func (s *Store) Update(ctx context.Context, shp shipment.Shipment) error {
	if err := s.storer.Update(ctx, shp); err != nil {
		return err
	}

	return s.deleteCache(ctx, shp.ID)
}

func (s *Store) Delete(ctx context.Context, shp shipment.Shipment) error {
	if err := s.storer.Delete(ctx, shp); err != nil {
		return err
	}

	return s.deleteCache(ctx, shp.ID)
}

//===========================================================================================================

func (s *Store) writeCache(ctx context.Context, shp shipment.Shipment) error {
	data, err := json.Marshal(shp)
	if err != nil {
		return fmt.Errorf("marshal shipment: %w", err)
	}

	key := fmt.Sprintf("shipment:%s", shp.ID)

	return s.cache.Set(ctx, key, data, 24*time.Hour).Err()
}

func (s *Store) readCache(ctx context.Context, id string) (shipment.Shipment, error) {
	key := fmt.Sprintf("shipment:%s", id)

	data, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return shipment.Shipment{}, err
	}

	var shp shipment.Shipment
	if err := json.Unmarshal(data, &shp); err != nil {
		return shipment.Shipment{}, fmt.Errorf("unmarshal cache: %w", err)
	}

	return shp, nil
}

func (s *Store) deleteCache(ctx context.Context, id string) error {
	key := fmt.Sprintf("shipment:%s", id)
	return s.cache.Del(ctx, key).Err()

}
